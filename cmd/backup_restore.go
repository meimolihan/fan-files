package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(bipBackupCmd)
	rootCmd.AddCommand(bipRestoreCmd)
}

var bipBackupCmd = &cobra.Command{
	Use:   "backup [输出.zip]",
	Short: "备份数据目录为 zip",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dest := ""
		if len(args) > 0 {
			dest = args[0]
		}
		os.Exit(bipMSBackup(dest, cmd))
	},
}

var bipRestoreCmd = &cobra.Command{
	Use:   "restore [-y] <备份.zip>",
	Short: "从备份 zip 还原数据目录（-y 免确认）",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		yes, _ := cmd.Flags().GetBool("yes")
		os.Exit(bipMSRestore(args[0], yes, cmd))
	},
}

func init() {
	bipRestoreCmd.Flags().BoolP("yes", "y", false, "免确认，直接还原覆盖数据")
}

// bipDataArg 返回命令行 -data 参数值（未设置则空串）。
func bipDataArg(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	d, _ := cmd.Flags().GetString("data")
	return d
}

// bipResolveDataDir 解析实际数据目录：-data 参数 > /etc/fan-files.conf DATA_DIR > 默认值。
func bipResolveDataDir(cmd *cobra.Command) string {
	if d := bipDataArg(cmd); d != "" {
		return d
	}
	if d, ok := bipReadRecord("DATA_DIR"); ok && d != "" {
		return d
	}
	return bipDefaultData
}

// bipDetectDataFile 返回数据目录中作为备份校验标志的唯一文件（数据库文件）。
func bipDetectDataFile(dataDir string) string {
	entries, err := os.ReadDir(dataDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.HasSuffix(e.Name(), ".db") {
				return e.Name()
			}
		}
	}
	return bipBinName + ".db"
}

var sqliteMagic = "SQLite format 3\x00"

func isSQLiteFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 16)
	n, _ := io.ReadFull(f, buf)
	return n == len(buf) && string(buf) == sqliteMagic
}

// snapshotSQLite 生成一致的 SQLite 快照：优先调用 sqlite3 CLI，否则退化为直接拷贝。
func snapshotSQLite(dbPath, snapPath string) error {
	if cmd, err := exec.LookPath("sqlite3"); err == nil {
		if err := exec.Command(cmd, dbPath, ".backup '"+snapPath+"'").Run(); err == nil {
			return nil
		}
	}
	return bipCopyFile(dbPath, snapPath)
}

func bipCopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// bipMSBackup 打包数据目录为 zip。Usage: fan-files backup [输出.zip]
func bipMSBackup(dest string, cmd *cobra.Command) int {
	cliBanner("备份")
	cliSep()

	dataDir := bipResolveDataDir(cmd)
	info, err := os.Stat(dataDir)
	if err != nil || !info.IsDir() {
		cliErr("数据目录不存在: %s", dataDir)
		cliSep()
		return 1
	}

	if dest == "" {
		dest = fmt.Sprintf("fan-files-backup-%s.zip", time.Now().Format("20060102-150405"))
	}

	cliSection("备份数据目录")
	cliKV("数据目录", dataDir)
	cliKV("目标文件", dest)

	if abs, err := filepath.Abs(dest); err == nil {
		parent := filepath.Dir(abs)
		if parent == dataDir || strings.HasPrefix(parent, dataDir+string(os.PathSeparator)) {
			cliErr("备份文件不能输出到数据目录内部")
			cliSep()
			return 1
		}
	}

	running := false
	for _, p := range bipRunningProcs() {
		if p.data == dataDir {
			running = true
			break
		}
	}
	if running {
		cliHint("检测到 fan-files 正在使用数据目录，包含 SQLite 时将生成数据库快照保证一致性。")
	}

	marker := bipDetectDataFile(dataDir)
	snaps := make(map[string]string)
	if marker != "" {
		dbPath := filepath.Join(dataDir, marker)
		if isSQLiteFile(dbPath) {
			cliHint("正在生成数据库快照 ...")
			snap, cErr := os.CreateTemp("", "fan-files-backup-*.db")
			if cErr != nil {
				cliErr("创建临时文件失败: %v", cErr)
				cliSep()
				return 1
			}
			snapName := snap.Name()
			snap.Close()
			defer os.Remove(snapName)
			if sErr := snapshotSQLite(dbPath, snapName); sErr != nil {
				cliErr("生成数据库快照失败: %v", sErr)
				cliSep()
				return 1
			}
			snaps[marker] = snapName
		}
	}

	done := make(chan struct{})
	go cliSpinner(done, "正在打包数据 ...")
	files, err := bipZipDir(dataDir, dest, marker, snaps)
	close(done)
	if err != nil {
		cliErr("备份失败: %v", err)
		cliSep()
		return 1
	}
	if abs, err := filepath.Abs(dest); err == nil {
		dest = abs
	}
	var size int64
	if st, err := os.Stat(dest); err == nil {
		size = st.Size()
	}
	cliSuccessBox("备份完成")
	cliKV("备份文件", dest)
	cliKV("文件大小", cliHumanSize(size))
	cliKV("文件数", fmt.Sprintf("%d 个", files))
	cliSep()
	return 0
}

// bipMSRestore 从备份 zip 还原数据目录。Usage: fan-files restore [-y] <备份.zip>
func bipMSRestore(backupFile string, yes bool, cmd *cobra.Command) int {
	if _, err := os.Stat(backupFile); err != nil {
		cliErr("备份文件不存在: %s", backupFile)
		cliSep()
		return 1
	}

	dataDir := bipResolveDataDir(cmd)
	marker := bipDetectDataFile(dataDir)

	if err := bipCheckZip(backupFile, marker); err != nil {
		cliErr("无效的备份文件: %v", err)
		cliSep()
		return 1
	}

	cliBanner("还原")
	cliSep()
	cliSection("还原备份")
	cliKV("备份文件", backupFile)
	cliKV("数据目录", dataDir)

	if !yes && !cliConfirm(fmt.Sprintf("还原将覆盖数据目录 %s 中的现有数据，是否继续", dataDir), false) {
		fmt.Println(cliPaintColor("已取消还原。", cliYellow))
		return 0
	}

	cliSection("停止服务")
	if _, err := os.Stat(bipServiceFile); err == nil {
		cliOK("正在停止 %s systemd 服务 ...", bipServiceName)
		bipMSRun("systemctl", "stop", bipServiceName)
	}
	bipStopUsingDataDir(dataDir)

	backupPath := dataDir + ".backup-" + time.Now().Format("20060102-150405")
	if bipIsDir(dataDir) {
		if err := os.Rename(dataDir, backupPath); err != nil {
			cliWarn("备份现有数据失败: %v", err)
		} else {
			cliOK("现有数据已备份到 %s", backupPath)
		}
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		cliErr("创建数据目录失败: %v", err)
		cliSep()
		return 1
	}

	cliSection("还原数据")
	done := make(chan struct{})
	go cliSpinner(done, "正在还原数据 ...")
	files, err := bipUnzipDir(backupFile, dataDir, marker)
	close(done)
	if err != nil {
		cliErr("还原失败: %v", err)
		if bipIsDir(backupPath) {
			cliHint("原数据已回退到 %s，可手动恢复。", backupPath)
		}
		cliSep()
		return 1
	}
	cliSuccessBox(fmt.Sprintf("还原完成（共 %d 个文件）", files))
	cliKV("启动命令", "systemctl start "+bipServiceName)
	cliSep()
	return 0
}

// bipStopUsingDataDir 终止使用指定数据目录的 fan-files 进程。
func bipStopUsingDataDir(dataDir string) {
	var pids []int
	for _, p := range bipRunningProcs() {
		if p.data == dataDir {
			pids = append(pids, p.pid)
		}
	}
	if len(pids) == 0 {
		return
	}
	cliOK("正在停止使用数据目录 %s 的 fan-files 进程: %v ...", dataDir, pids)
	for _, pid := range pids {
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	time.Sleep(time.Second)
	for _, pid := range pids {
		if _, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid))); err == nil {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
}

// bipZipDir 归档 src 到 dest，条目以 src 为根相对存储。snaps 提供 sqlite 快照替换。
// 返回归档的文件数。
func bipZipDir(src, dest, marker string, snaps map[string]string) (int, error) {
	out, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()

	files := 0
	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(rel, marker+"-") {
			// 跳过实时 WAL/SHM/Journal 伴随文件；快照才是权威数据
			return nil
		}
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			hdr := &zip.FileHeader{Name: rel + "/", Method: zip.Deflate}
			hdr.SetMode(0755)
			_, err := zw.CreateHeader(hdr)
			return err
		}
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = rel
		hdr.Method = zip.Deflate
		hdr.SetMode(info.Mode())
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		file := path
		if snap, ok := snaps[rel]; ok {
			file = snap
		}
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		_, cErr := io.Copy(w, f)
		f.Close()
		if cErr != nil {
			return cErr
		}
		files++
		return nil
	})
	return files, err
}

// bipCheckZip 校验 path 是有效的 zip 且包含备份标志文件 marker。
func bipCheckZip(path, marker string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("无法解压 %s（不是有效的 zip 文件？）", path)
	}
	defer r.Close()
	for _, f := range r.File {
		if filepath.Base(f.Name) == marker {
			return nil
		}
	}
	return fmt.Errorf("压缩包中未找到 %s，可能不是 fan-files 备份", marker)
}

// bipUnzipDir 解压归档到 dest，防 zip-slip，返回解压的文件数。
func bipUnzipDir(src, dest, marker string) (int, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	files := 0
	for _, f := range r.File {
		name := filepath.Clean(filepath.FromSlash(f.Name))
		if name == ".." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || filepath.IsAbs(name) {
			return 0, fmt.Errorf("压缩包中包含非法路径: %s", f.Name)
		}
		if strings.HasPrefix(name, marker+"-") {
			// 陈旧的 WAL/SHM/Journal 文件会破坏还原后的数据库
			continue
		}
		target := filepath.Join(dest, name)
		if !strings.HasPrefix(target, dest+string(os.PathSeparator)) {
			return 0, fmt.Errorf("压缩包路径越界: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return 0, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return 0, err
		}
		rc, err := f.Open()
		if err != nil {
			return 0, err
		}
		mode := f.Mode() & 0777
		if mode == 0 {
			mode = 0600
		}
		w, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			rc.Close()
			return 0, err
		}
		_, cErr := io.Copy(w, rc)
		rc.Close()
		w.Close()
		if cErr != nil {
			return 0, cErr
		}
		files++
	}
	return files, nil
}

func bipIsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}