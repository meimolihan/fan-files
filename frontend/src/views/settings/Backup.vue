<template>
  <div class="settings-page backup-settings">
    <div class="settings-section">
      <h2>{{ t("backup.backupRestore") }}</h2>
      <p class="description">{{ t("backup.description") }}</p>
    </div>

    <div class="settings-section">
      <h3>{{ t("backup.createBackup") }}</h3>
      <div class="backup-form">
        <div class="form-group">
          <label>{{ t("backup.backupDir") }}</label>
          <input
            type="text"
            v-model="backupDir"
            :placeholder="defaultBackupDir"
            @blur="loadBackups"
          />
          <span class="help-text">{{ t("backup.backupDirHelp") }}</span>
        </div>
        <div class="form-group">
          <label>{{ t("backup.keepNum") }}</label>
          <input type="number" v-model.number="keepNum" min="1" max="100" />
          <span class="help-text">{{ t("backup.keepNumHelp") }}</span>
        </div>
        <button class="btn btn-primary" @click="createBackup" :disabled="creating">
          <span v-if="creating">{{ t("backup.creating") }}...</span>
          <span v-else>{{ t("backup.createBackup") }}</span>
        </button>
      </div>
    </div>

    <div class="settings-section">
      <h3>{{ t("backup.backupList") }}</h3>
      <div v-if="loadingBackups" class="loading">{{ t("files.loading") }}</div>
      <div v-else-if="backups.length === 0" class="empty-state">
        {{ t("backup.noBackups") }}
      </div>
      <table v-else class="backup-table">
        <thead>
          <tr>
            <th>{{ t("backup.name") }}</th>
            <th>{{ t("backup.size") }}</th>
            <th>{{ t("backup.date") }}</th>
            <th>{{ t("backup.actions") }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="backup in backups" :key="backup.name">
            <td>{{ backup.name }}</td>
            <td>{{ formatSize(backup.size) }}</td>
            <td>{{ formatDate(backup.modTime) }}</td>
            <td class="actions">
              <button
                class="btn btn-sm btn-secondary"
                @click="restoreBackup(backup.name)"
                :disabled="restoring === backup.name"
              >
                <span v-if="restoring === backup.name">{{ t("backup.restoring") }}...</span>
                <span v-else>{{ t("backup.restore") }}</span>
              </button>
              <button
                class="btn btn-sm btn-danger"
                @click="deleteBackup(backup.name)"
                :disabled="deleting === backup.name"
              >
                <span v-if="deleting === backup.name">{{ t("backup.deleting") }}...</span>
                <span v-else>{{ t("backup.delete") }}</span>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="settings-section" v-if="jobStatus">
      <h3>{{ t("backup.jobStatus") }}</h3>
      <div class="job-status" :class="jobStatus.status">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: jobStatus.progress + '%' }"></div>
        </div>
        <p>{{ jobStatus.message }}</p>
        <small>{{ t("backup.progress") }}: {{ jobStatus.progress }}%</small>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import * as backupApi from "@/api/backup";

const { t } = useI18n();
const authStore = useAuthStore();

const backupDir = ref("");
const keepNum = ref(6);
const creating = ref(false);
const restoring = ref<string | null>(null);
const deleting = ref<string | null>(null);
const loadingBackups = ref(false);
const backups = ref<BackupInfo[]>([]);
const jobStatus = ref<JobStatus | null>(null);

const defaultBackupDir = "";

interface BackupInfo {
  name: string;
  path: string;
  size: number;
  modTime: string;
}

interface JobStatus {
  jobId: string;
  status: string;
  message: string;
  progress: number;
  file?: string;
}

interface BackupListResponse {
  backups: BackupInfo[];
  dir: string;
}

interface ApiResponse {
  success: boolean;
  message: string;
  jobId?: string;
  status?: string;
  progress?: number;
  file?: string;
}

async function loadBackups() {
  loadingBackups.value = true;
  try {
    const res = await backupApi.listBackups();
    backups.value = res.backups;
    if (backupDir.value === "" && res.dir) {
      backupDir.value = res.dir;
    }
  } catch (error) {
    console.error("Failed to load backups:", error);
  } finally {
    loadingBackups.value = false;
  }
}

async function createBackup() {
  creating.value = true;
  jobStatus.value = null;
  try {
    const res = await backupApi.createBackup({
      dir: backupDir.value || undefined,
      keepNum: keepNum.value,
      async: true,
    });
    if (res.jobId) {
      pollJobStatus(res.jobId);
    }
    await loadBackups();
  } catch (error) {
    console.error("Failed to create backup:", error);
  } finally {
    creating.value = false;
  }
}

async function restoreBackup(filename: string) {
  if (!confirm(t("backup.confirmRestore"))) return;
  restoring.value = filename;
  jobStatus.value = null;
  try {
    const res = await backupApi.restoreBackup({
      file: filename,
    });
    if (res.jobId) {
      pollJobStatus(res.jobId);
    }
  } catch (error) {
    console.error("Failed to restore backup:", error);
  } finally {
    restoring.value = null;
  }
}

async function deleteBackup(filename: string) {
  if (!confirm(t("backup.confirmDelete"))) return;
  deleting.value = filename;
  try {
    await backupApi.deleteBackup(filename);
    await loadBackups();
  } catch (error) {
    console.error("Failed to delete backup:", error);
  } finally {
    deleting.value = null;
  }
}

function pollJobStatus(jobId: string) {
  const checkStatus = async () => {
    try {
      const res = await backupApi.getJobStatus(jobId);
      jobStatus.value = {
        jobId,
        status: res.status || "unknown",
        message: res.message || "",
        progress: res.progress || 0,
        file: res.file,
      };
      if (res.status === "completed" || res.status === "failed") {
        await loadBackups();
        return;
      }
      setTimeout(checkStatus, 2000);
    } catch (error) {
      console.error("Failed to poll job status:", error);
    }
  };
  checkStatus();
}

function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function formatDate(dateString: string): string {
  try {
    const date = new Date(dateString);
    return date.toLocaleString();
  } catch {
    return dateString;
  }
}

onMounted(() => {
  loadBackups();
});
</script>

<style scoped>
.settings-page {
  padding: 24px;
  max-width: 1000px;
}

.settings-section {
  margin-bottom: 32px;
  padding: 20px;
  background: var(--surface-color);
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.settings-section h2 {
  margin: 0 0 8px;
  color: var(--text-primary);
}

.settings-section h3 {
  margin: 24px 0 16px;
  color: var(--text-primary);
  font-size: 1.1rem;
}

.description {
  color: var(--text-secondary);
  margin: 0;
}

.backup-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 500px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-group label {
  font-weight: 500;
  color: var(--text-primary);
}

.form-group input {
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--input-bg);
  color: var(--text-primary);
  font-size: 14px;
}

.form-group input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(var(--primary-rgb), 0.2);
}

.help-text {
  font-size: 12px;
  color: var(--text-secondary);
}

.btn {
  padding: 10px 20px;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  filter: brightness(1.1);
}

.btn-secondary {
  background: var(--secondary-bg);
  color: var(--text-primary);
}

.btn-danger {
  background: var(--danger-color);
  color: white;
}

.btn-sm {
  padding: 6px 12px;
  font-size: 13px;
}

.actions {
  display: flex;
  gap: 8px;
}

.backup-table {
  width: 100%;
  border-collapse: collapse;
}

.backup-table th,
.backup-table td {
  padding: 12px 16px;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

.backup-table th {
  font-weight: 600;
  color: var(--text-secondary);
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.backup-table tbody tr:hover {
  background: var(--hover-bg);
}

.empty-state {
  text-align: center;
  padding: 40px;
  color: var(--text-secondary);
}

.loading {
  text-align: center;
  padding: 40px;
  color: var(--text-secondary);
}

.job-status {
  padding: 16px;
  border-radius: 8px;
  background: var(--surface-color);
}

.job-status.pending,
.job-status.running {
  border-left: 4px solid var(--warning-color);
}

.job-status.completed {
  border-left: 4px solid var(--success-color);
}

.job-status.failed {
  border-left: 4px solid var(--danger-color);
}

.progress-bar {
  height: 6px;
  background: var(--border-color);
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 12px;
}

.progress-fill {
  height: 100%;
  background: var(--primary-color);
  transition: width 0.3s ease;
}

.job-status small {
  color: var(--text-secondary);
}
</style>