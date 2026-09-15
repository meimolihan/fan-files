package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// fan-files 安装/运行约定（与 scripts/install.sh、scripts/uninstall.sh 保持一致）。
const (
	bipBinName     = "fan-files"
	bipServiceName = "fan-files"
	bipServiceFile = "/etc/systemd/system/fan-files.service"
	bipRecordFile  = "/etc/fan-files.conf"
	bipSettingsDir = "/etc/fan-files"
	bipDefaultData = "/var/lib/fan-files"
	bipDefaultPort = 8678
)

type bipProc struct {
	pid  int
	data string
}

var (
	bipDataArgRe = regexp.MustCompile(`(-d|--database)\s+(?:([^"' \t]+)|"([^"]+)"|'([^']+)')`)
)

func init() {
	rootCmd.AddCommand(bipStatusCmd)
	rootCmd.AddCommand(bipUninstallCmd)
}

var bipStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "显示 fan-files 服务运行状态",
	Run: func(_ *cobra.Command, _ []string) {
		os.Exit(cliMSStatus())
	},
}

var bipUninstallCmd = &cobra.Command{
	Use:   "uninstall [flags]",
	Short: "卸载 fan-files（停止服务并移除程序）",
	Run: func(cmd *cobra.Command, _ []string) {
		yes, _ := cmd.Flags().GetBool("yes")
		purge, _ := cmd.Flags().GetBool("purge")
		keep, _ := cmd.Flags().GetBool("keep-data")
		os.Exit(cliMSUninstall(yes, purge, keep))
	},
}

func init() {
	bipUninstallCmd.Flags().BoolP("yes", "y", false, "免确认，静默卸载（默认保留数据目录）")
	bipUninstallCmd.Flags().Bool("purge", false, "卸载时同时删除数据目录")
	bipUninstallCmd.Flags().Bool("keep-data", false, "卸载时保留数据目录")
}

// cliMSStatus 显示服务状态：systemd > Docker 容器 > /proc 进程。
func cliMSStatus() int {
	cliBanner("服务状态")
	cliSep()

	if port, ok := bipReadRecord("PORT"); ok {
		cliKV("install 记录端口", port)
	}
	if dir, ok := bipReadRecord("DATA_DIR"); ok {
		cliKV("install 记录数据目录", dir)
	}

	typ, pid := bipFindProcess("")
	if typ == "" || pid == 0 {
		cliErr("%s 服务未运行", bipBinName)
		cliSep()
		return 1
	}

	cliSection("服务方式")
	cliKV("运行方式", typ)
	cliSection("进程信息")
	cliKV("进程 PID", strconv.Itoa(pid))
	cliKV("线程数量", bipProcField(pid, "Threads"))
	cliSection("网络")
	if port := bipListenPort(pid); port != "" {
		cliKV("监听端口", port)
	} else {
		cliKV("监听端口", "未找到")
	}
	cliSection("运行时间")
	cliKV("已运行", bipFormatUptime(bipProcUptimeSec(pid)))
	cliSection("内存")
	cliKV("虚拟内存", bipFormatMem(bipProcField(pid, "VmSize")))
	cliKV("物理内存", bipFormatMem(bipProcField(pid, "VmRSS")))
	if entries, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid)); err == nil {
		cliKV("打开文件", strconv.Itoa(len(entries)))
	}
	cliSection("路径")
	cliKV("数据目录", bipDetectDataDir(bipDefaultData))
	cliKV("配置文件", bipSettingsDir)
	if exe, err := os.Executable(); err == nil {
		cliKV("当前二进制", exe)
	}
	cliSep()
	return 0
}

// bipFindProcess 返回 (运行方式, PID)。
func bipFindProcess(_ string) (string, int) {
	out, _ := exec.Command("systemctl", "show", "-p", "MainPID", "--value", bipServiceName).Output()
	if pid, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && pid > 0 {
		if _, e := os.Stat(fmt.Sprintf("/proc/%d", pid)); e == nil {
			return "systemd (" + bipServiceName + ".service)", pid
		}
	}
	if line, err := exec.Command("docker", "ps", "--filter", "name="+bipBinName, "--format", "{{.Names}}|{{.Status}}").Output(); err == nil {
		for _, s := range strings.Split(string(line), "\n") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			parts := strings.SplitN(s, "|", 2)
			if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
				continue
			}
			if p := bipContainerPID(strings.TrimSpace(parts[0])); p > 0 {
				return "docker（容器 " + strings.TrimSpace(parts[0]) + "）", p
			}
		}
	}
	if pid := bipProcScan(); pid > 0 {
		return "直接运行（/proc）", pid
	}
	return "", 0
}

func bipContainerPID(name string) int {
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Pid}}", name).Output()
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

func bipProcScan() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	for _, e := range entries {
		if !e.IsDir() || !bipIsNum(e.Name()) {
			continue
		}
		pid, _ := strconv.Atoi(e.Name())
		if pid <= 0 || pid == os.Getpid() {
			continue
		}
		if t, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil && filepath.Base(t) == bipBinName {
			return pid
		}
	}
	return 0
}

func bipRunningProcs() []bipProc {
	var out []bipProc
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() || !bipIsNum(e.Name()) {
			continue
		}
		pid, _ := strconv.Atoi(e.Name())
		if pid <= 0 {
			continue
		}
		if t, err := os.Readlink(filepath.Join("/proc", e.Name(), "exe")); err != nil || filepath.Base(t) != bipBinName {
			continue
		}
		p := bipProc{pid: pid}
		if raw, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline")); err == nil {
			cmdline := strings.ReplaceAll(string(raw), "\x00", " ")
			if m := bipDataArgRe.FindStringSubmatch(cmdline); m != nil {
				for _, g := range m[1:] {
					if g != "" {
						p.data = g
						break
					}
				}
			}
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pid < out[j].pid })
	return out
}

func bipProcField(pid int, name string) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, name+":") {
			return strings.TrimSpace(strings.TrimPrefix(line, name+":"))
		}
	}
	return ""
}

func bipProcUptimeSec(pid int) int64 {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	upt, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	content := string(stat)
	closeIdx := strings.LastIndex(content, ")")
	if closeIdx < 0 || closeIdx+2 >= len(content) {
		return 0
	}
	fields := strings.Fields(content[closeIdx+2:])
	if len(fields) < 20 {
		return 0
	}
	startTicks, _ := strconv.ParseInt(fields[19], 10, 64)
	start := startTicks / 100
	upSec, _ := strconv.ParseFloat(strings.Fields(string(upt))[0], 64)
	elapsed := int64(upSec) - start
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

func bipFormatUptime(sec int64) string {
	if sec <= 0 {
		return "未知"
	}
	return fmt.Sprintf("%d小时 %d分钟 %d秒", sec/3600, (sec%3600)/60, sec%60)
}

func bipFormatMem(raw string) string {
	if raw == "" {
		return "未知"
	}
	parts := strings.Fields(raw)
	if len(parts) < 2 {
		return raw
	}
	kb, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return raw
	}
	mb := kb / 1024.0
	if mb >= 1024 {
		return fmt.Sprintf("%s（%.2f GB）", raw, mb/1024.0)
	}
	return fmt.Sprintf("%s（%.2f MB）", raw, mb)
}

func bipListenPort(pid int) string {
	inodes := bipSocketInodes(pid)
	if len(inodes) == 0 {
		return ""
	}
	for _, proto := range []string{"tcp", "tcp6"} {
		if p := bipPortInTCP(pid, proto, inodes); p != "" {
			return p
		}
	}
	return ""
}

func bipSocketInodes(pid int) map[int64]bool {
	dir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	inodes := make(map[int64]bool)
	for _, e := range entries {
		t, err := os.Readlink(filepath.Join(dir, e.Name()))
		if err != nil || !strings.HasPrefix(t, "socket:[") || !strings.HasSuffix(t, "]") {
			continue
		}
		num := strings.TrimSuffix(strings.TrimPrefix(t, "socket:["), "]")
		if v, err := strconv.ParseInt(num, 10, 64); err == nil {
			inodes[v] = true
		}
	}
	return inodes
}

func bipPortInTCP(pid int, proto string, inodes map[int64]bool) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/net/%s", pid, proto))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 {
		lines = lines[1:]
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 10 || fields[3] != "0A" {
			continue
		}
		inode, err := strconv.ParseInt(fields[9], 10, 64)
		if err != nil || !inodes[inode] {
			continue
		}
		parts := strings.Split(fields[1], ":")
		if len(parts) < 2 {
			continue
		}
		port, err := strconv.ParseInt(parts[1], 16, 64)
		if err != nil || port <= 0 {
			continue
		}
		return strconv.FormatInt(port, 10)
	}
	return ""
}

func bipIsNum(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func bipReadRecord(key string) (string, bool) {
	content, err := os.ReadFile(bipRecordFile)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(k) == key {
			v = strings.TrimSpace(v)
			return v, v != ""
		}
	}
	return "", false
}

// bipDetectDataDir 解析实际数据目录：记录 > 默认值。
func bipDetectDataDir(fallback string) string {
	if dir, ok := bipReadRecord("DATA_DIR"); ok {
		return dir
	}
	return fallback
}
