package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// cliMSUninstall 卸载 fan-files：停止并移除 systemd 服务、移除容器、关闭防火墙端口、
// 删除二进制与安装记录，可选删除数据目录。
func cliMSUninstall(yes, purge, keep bool) int {
	if os.Geteuid() != 0 {
		cliErr("请以 root 身份运行：sudo fan-files uninstall")
		return 1
	}
	if purge && keep {
		cliErr("--purge 与 --keep-data 不能同时使用")
		return 1
	}

	cliBanner("卸载")
	cliSep()

	if !yes && !cliConfirm("卸载将停止并移除 fan-files 服务与程序，是否继续", false) {
		fmt.Println(cliPaintColor("已取消卸载。", cliYellow))
		return 0
	}

	if port, ok := bipReadRecord("PORT"); ok {
		if p, err := strconv.Atoi(port); err == nil && p > 0 && p <= 65535 {
			bipMSCloseFirewallPort(p)
		}
	} else {
		bipMSCloseFirewallPort(bipDefaultPort)
	}

	if _, err := os.Stat(bipServiceFile); err == nil {
		cliDone("正在停止并移除 systemd 服务 fan-files ...")
		bipMSRun("systemctl", "stop", bipServiceName)
		bipMSRun("systemctl", "disable", bipServiceName)
		_ = os.Remove(bipServiceFile)
		bipMSRun("systemctl", "daemon-reload")
		bipMSRun("systemctl", "reset-failed")
	}

	if cmd, err := exec.LookPath("docker"); err == nil {
		if out, e := exec.Command(cmd, "ps", "-a", "--filter", "name="+bipBinName, "--format", "{{.Names}}").Output(); e == nil {
			for _, n := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				n = strings.TrimSpace(n)
				if n == bipBinName {
					cliDone("正在移除容器 %s ...", n)
					bipMSRun(cmd, "rm", "-f", n)
				}
			}
		}
	}

	pids := make([]int, 0)
	for _, p := range bipRunningProcs() {
		pids = append(pids, p.pid)
	}
	if len(pids) > 0 {
		cliDone("正在停止 fan-files 进程: %v ...", pids)
		for _, pid := range pids {
			if pr, err := os.FindProcess(pid); err == nil {
				_ = pr.Signal(syscall.SIGTERM)
			}
		}
		time.Sleep(time.Second)
		for _, pid := range pids {
			if _, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid))); err == nil {
				if pr, err := os.FindProcess(pid); err == nil {
					_ = pr.Signal(syscall.SIGKILL)
				}
			}
		}
	}

	bipMSRemoveBinary()

	dataDir := bipDetectDataDir(bipDefaultData)
	info, statErr := os.Stat(dataDir)
	if statErr != nil || !info.IsDir() {
		cliDone("未检测到数据目录 %s，跳过删除。", dataDir)
	} else {
		if size, err := bipMSDirSize(dataDir); err == nil {
			cliDone("检测到数据目录: %s（约 %s）", dataDir, cliHumanSize(size))
		}
		remove := false
		switch {
		case purge:
			remove = true
		case keep:
			remove = false
		case yes:
			remove = false
		default:
			remove = cliConfirm(fmt.Sprintf("是否删除数据目录 %s（完全卸载）", dataDir), true)
		}
		if remove {
			if err := os.RemoveAll(dataDir); err != nil {
				cliWarn("删除数据目录失败: %v", err)
			} else {
				cliDone("已删除数据目录 %s", dataDir)
			}
		} else {
			cliDone("已保留数据目录 %s", dataDir)
		}
	}

	_ = os.Remove(bipRecordFile)
	cliDone("已删除安装记录 %s", bipRecordFile)
	_ = os.RemoveAll(bipSettingsDir)
	cliDone("已删除配置目录 %s", bipSettingsDir)

	cliDone("fan-files 卸载完成")
	fmt.Println(cliPaintColor("如需重新安装，请重新部署（install.sh / docker compose）。", cliGrey))
	return 0
}

func bipMSRemoveBinary() {
	paths := map[string]string{"/usr/local/bin/fan-files": "/usr/local/bin/fan-files"}
	if exe, err := os.Executable(); err == nil {
		paths[exe] = exe
	}
	for _, p := range paths {
		if _, err := os.Lstat(p); err != nil {
			continue
		}
		if err := os.Remove(p); err != nil {
			cliWarn("删除二进制文件 %s 失败: %v", p, err)
		} else {
			cliDone("已删除二进制文件 %s", p)
		}
	}
}

func bipMSCloseFirewallPort(port int) {
	portStr := strconv.Itoa(port)
	if cmd, err := exec.LookPath("firewall-cmd"); err == nil {
		if out, _ := exec.Command(cmd, "--state").Output(); strings.TrimSpace(string(out)) == "running" {
			bipMSRun(cmd, "--permanent", "--remove-port="+portStr+"/tcp")
			bipMSRun(cmd, "--reload")
			cliDone("已通过 firewalld 关闭端口 %s/tcp", portStr)
			return
		}
	}
	if cmd, err := exec.LookPath("ufw"); err == nil {
		if out, _ := exec.Command(cmd, "status").Output(); strings.Contains(string(out), "active") {
			bipMSRun(cmd, "delete", "allow", portStr+"/tcp")
			cliDone("已通过 ufw 关闭端口 %s/tcp", portStr)
			return
		}
	}
	if cmd, err := exec.LookPath("iptables"); err == nil {
		if err := exec.Command(cmd, "-D", "INPUT", "-p", "tcp", "--dport", portStr, "-j", "ACCEPT").Run(); err == nil {
			cliDone("已通过 iptables 关闭端口 %s/tcp", portStr)
		}
	}
}

func bipMSRun(name string, args ...string) {
	if _, err := exec.LookPath(name); err != nil {
		return
	}
	_ = exec.Command(name, args...).Run()
}

func bipMSDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
