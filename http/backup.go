package fbhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

const (
	backupConfigFile = "/etc/fan-files.conf"
	backupPrefix     = "FanFiles-"
	backupExt        = ".tar.gz"
)

type backupInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

type backupListResponse struct {
	Backups []backupInfo `json:"backups"`
	Dir     string       `json:"dir"`
}

type backupRequest struct {
	Dir      string `json:"dir,omitempty"`
	KeepNum  int    `json:"keepNum,omitempty"`
	Async    bool   `json:"async,omitempty"`
}

type backupResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	JobID    string `json:"jobId,omitempty"`
	Status   string `json:"status,omitempty"`
	Progress int    `json:"progress,omitempty"`
}

type backupJob struct {
	ID        string
	Type      string // "backup" or "restore"
	Status    string // "pending", "running", "completed", "failed"
	Progress  int
	Message   string
	File      string
	CreatedAt time.Time
	mu        sync.Mutex
}

var (
	backupJobs   = make(map[string]*backupJob)
	backupJobsMu sync.Mutex
)

func getBackupConfig() (dataDir, backupDir string, err error) {
	dataDir = "/var/lib/fan-files"

	if _, err := os.Stat(backupConfigFile); err == nil {
		content, err := os.ReadFile(backupConfigFile)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATA_DIR=") {
					dataDir = strings.TrimPrefix(line, "DATA_DIR=")
				} else if strings.HasPrefix(line, "BACKUP_DIR=") {
					backupDir = strings.TrimPrefix(line, "BACKUP_DIR=")
				}
			}
		}
	}

	if backupDir == "" {
		backupDir = filepath.Join(dataDir, "backup")
	}
	return dataDir, backupDir, nil
}

func listBackupFiles(backupDir string) ([]backupInfo, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []backupInfo{}, nil
		}
		return nil, err
	}

	var backups []backupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, backupPrefix) || !strings.HasSuffix(name, backupExt) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, backupInfo{
			Name:    name,
			Path:    filepath.Join(backupDir, name),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	return backups, nil
}

func newJob(jobType string) *backupJob {
	id := fmt.Sprintf("%s-%d", jobType, time.Now().UnixNano())
	job := &backupJob{
		ID:        id,
		Type:      jobType,
		Status:    "pending",
		Progress:  0,
		CreatedAt: time.Now(),
	}
	backupJobsMu.Lock()
	backupJobs[id] = job
	backupJobsMu.Unlock()
	return job
}

func getJob(id string) *backupJob {
	backupJobsMu.Lock()
	defer backupJobsMu.Unlock()
	return backupJobs[id]
}

func (j *backupJob) update(status, message string, progress int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = status
	j.Message = message
	j.Progress = progress
}

var backupListHandler = withAdmin(func(w http.ResponseWriter, r *http.Request, _ *data) (int, error) {
	_, backupDir, err := getBackupConfig()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	backups, err := listBackupFiles(backupDir)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return renderJSON(w, r, backupListResponse{
		Backups: backups,
		Dir:     backupDir,
	})
})

var backupCreateHandler = withAdmin(func(w http.ResponseWriter, r *http.Request, _ *data) (int, error) {
	var req backupRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return http.StatusBadRequest, err
		}
	}

	dataDir, backupDir, err := getBackupConfig()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if req.Dir != "" {
		backupDir = req.Dir
	}
	if req.KeepNum <= 0 {
		req.KeepNum = 6
	}

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return http.StatusBadRequest, fmt.Errorf("data directory not found: %s", dataDir)
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return http.StatusInternalServerError, err
	}

	job := newJob("backup")

	if req.Async {
		go func() {
			runBackup(job, dataDir, backupDir, req.KeepNum)
		}()
		return renderJSON(w, r, backupResponse{
			Success: true,
			Message: "Backup started",
			JobID:   job.ID,
			Status:  "pending",
		})
	}

	// Synchronous backup (for CLI/script use)
	job.update("running", "Creating backup...", 10)
	now := time.Now().Format("2006-01-02_15-04-05")
	backupFile := filepath.Join(backupDir, backupPrefix+now+backupExt)

	backupRel, _ := filepath.Rel(dataDir, backupDir)
	if err := createTarGz(dataDir, backupFile, backupRel); err != nil {
		job.update("failed", fmt.Sprintf("Backup failed: %v", err), 0)
		return http.StatusInternalServerError, fmt.Errorf("backup failed: %v", err)
	}

	job.update("running", "Cleaning old backups...", 80)
	cleanOldBackups(backupDir, req.KeepNum)

	job.update("completed", "Backup completed successfully", 100)
	job.mu.Lock()
	job.File = backupFile
	job.mu.Unlock()

	return renderJSON(w, r, backupResponse{
		Success: true,
		Message: "Backup created successfully",
		File:    backupFile,
	})
})

var backupRestoreHandler = withAdmin(func(w http.ResponseWriter, r *http.Request, _ *data) (int, error) {
	var req struct {
		Dir  string `json:"dir,omitempty"`
		File string `json:"file,omitempty"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return http.StatusBadRequest, err
		}
	}

	dataDir, backupDir, err := getBackupConfig()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if req.Dir != "" {
		backupDir = req.Dir
	}

	var restoreFile string
	if req.File != "" {
		restoreFile = filepath.Join(backupDir, req.File)
		if _, err := os.Stat(restoreFile); os.IsNotExist(err) {
			return http.StatusBadRequest, fmt.Errorf("backup file not found: %s", restoreFile)
		}
	} else {
		backups, err := listBackupFiles(backupDir)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		if len(backups) == 0 {
			return http.StatusBadRequest, fmt.Errorf("no backup files found")
		}
		restoreFile = backups[0].Path
	}

	job := newJob("restore")

	go func() {
		runRestore(job, dataDir, restoreFile)
	}()

	return renderJSON(w, r, backupResponse{
		Success: true,
		Message: "Restore started",
		JobID:   job.ID,
		Status:  "pending",
	})
})

var backupDeleteHandler = withAdmin(func(w http.ResponseWriter, r *http.Request, _ *data) (int, error) {
	vars := mux.Vars(r)
	filename := vars["name"]
	if filename == "" {
		return http.StatusBadRequest, fmt.Errorf("filename required")
	}

	_, backupDir, err := getBackupConfig()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	filePath := filepath.Join(backupDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return http.StatusNotFound, fmt.Errorf("backup file not found")
	}

	if err := os.Remove(filePath); err != nil {
		return http.StatusInternalServerError, err
	}

	return renderJSON(w, r, backupResponse{
		Success: true,
		Message: "Backup deleted successfully",
	})
})

var backupJobStatusHandler = withAdmin(func(w http.ResponseWriter, r *http.Request, _ *data) (int, error) {
	vars := mux.Vars(r)
	jobID := vars["id"]
	if jobID == "" {
		return http.StatusBadRequest, fmt.Errorf("job id required")
	}

	job := getJob(jobID)
	if job == nil {
		return http.StatusNotFound, fmt.Errorf("job not found")
	}

	job.mu.Lock()
	defer job.mu.Unlock()

	return renderJSON(w, r, backupResponse{
		Success:  true,
		Message:  job.Message,
		File:     job.File,
		JobID:    job.ID,
		Status:   job.Status,
		Progress: job.Progress,
	})
})

func runBackup(job *backupJob, dataDir, backupDir string, keepNum int) {
	job.update("running", "Creating backup...", 10)
	now := time.Now().Format("2006-01-02_15-04-05")
	backupFile := filepath.Join(backupDir, backupPrefix+now+backupExt)

	// Calculate relative path of backupDir from dataDir for exclusion
	backupRel, _ := filepath.Rel(dataDir, backupDir)

	if err := createTarGz(dataDir, backupFile, backupRel); err != nil {
		job.update("failed", fmt.Sprintf("Backup failed: %v", err), 0)
		return
	}

	job.update("running", "Cleaning old backups...", 80)
	cleanOldBackups(backupDir, keepNum)

	job.mu.Lock()
	job.File = backupFile
	job.mu.Unlock()

	job.update("completed", "Backup completed successfully", 100)
}

func runRestore(job *backupJob, dataDir, restoreFile string) {
	job.update("running", "Stopping service...", 10)
	stopService("fan-files")

	job.update("running", "Restoring data...", 40)
	if err := extractTarGz(restoreFile, dataDir); err != nil {
		job.update("failed", fmt.Sprintf("Restore failed: %v", err), 0)
		startService("fan-files")
		return
	}

	job.update("running", "Starting service...", 80)
	startService("fan-files")

	// Wait for service to be ready
	for i := 0; i < 10; i++ {
		time.Sleep(2 * time.Second)
		if isServiceActive("fan-files") {
			break
		}
	}

	job.update("completed", "Restore completed successfully", 100)
}

func createTarGz(srcDir, destFile, excludeRel string) error {
	args := []string{"-czf", destFile, "-C", srcDir}
	if excludeRel != "" && excludeRel != "." {
		args = append(args, "--exclude="+excludeRel)
	}
	args = append(args, ".")
	cmd := exec.Command("tar", args...)
	return cmd.Run()
}

func extractTarGz(srcFile, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	cmd := exec.Command("tar", "-xzf", srcFile, "-C", destDir)
	return cmd.Run()
}

func cleanOldBackups(backupDir string, keepNum int) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}

	type backupFile struct {
		name    string
		modTime time.Time
	}

	var files []backupFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, backupPrefix) || !strings.HasSuffix(name, backupExt) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, backupFile{name: name, modTime: info.ModTime()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	for i := keepNum; i < len(files); i++ {
		os.Remove(filepath.Join(backupDir, files[i].name))
	}
}

func stopService(name string) {
	_ = exec.Command("systemctl", "stop", name).Run()
}

func startService(name string) {
	_ = exec.Command("systemctl", "start", name).Run()
}

func isServiceActive(name string) bool {
	out, err := exec.Command("systemctl", "is-active", name).Output()
	return err == nil && strings.TrimSpace(string(out)) == "active"
}