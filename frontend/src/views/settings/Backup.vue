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
        <button
          class="btn btn-primary"
          @click="createBackup"
          :disabled="creating"
        >
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
      <div v-else class="table-wrap">
        <table class="backup-table">
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
              <td :data-label="t('backup.size')">
                {{ formatSize(backup.size) }}
              </td>
              <td :data-label="t('backup.date')">
                {{ formatDate(backup.modTime) }}
              </td>
              <td class="actions">
                <button
                  class="btn btn-sm btn-secondary"
                  @click="restoreBackup(backup.name)"
                  :disabled="restoring === backup.name"
                >
                  <span v-if="restoring === backup.name"
                    >{{ t("backup.restoring") }}...</span
                  >
                  <span v-else>{{ t("backup.restore") }}</span>
                </button>
                <button
                  class="btn btn-sm btn-danger"
                  @click="deleteBackup(backup.name)"
                  :disabled="deleting === backup.name"
                >
                  <span v-if="deleting === backup.name"
                    >{{ t("backup.deleting") }}...</span
                  >
                  <span v-else>{{ t("backup.delete") }}</span>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="settings-section" v-if="jobStatus">
      <h3>{{ t("backup.jobStatus") }}</h3>
      <div class="job-status" :class="jobStatus.status">
        <div class="job-status__head">
          <span class="job-status__label">
            <span class="job-status__dot"></span>
            {{ t("backup." + jobStatus.status) }}
          </span>
          <span class="job-status__percent">{{ jobStatus.progress }}%</span>
        </div>
        <div class="progress-bar">
          <div
            class="progress-fill"
            :style="{ width: jobStatus.progress + '%' }"
          ></div>
        </div>
        <p>{{ jobMessage(jobStatus.message) }}</p>
        <small v-if="jobStatus.file">{{ jobStatus.file }}</small>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import * as backupApi from "@/api/backup";

const { t } = useI18n();

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

const jobMessageMap: Record<string, string> = {
  "Backup started": "backup.msgBackupStarted",
  "Creating backup...": "backup.msgCreatingBackup",
  "Cleaning old backups...": "backup.msgCleaningBackups",
  "Backup completed successfully": "backup.msgBackupCompleted",
  "Restore started": "backup.msgRestoreStarted",
  "Stopping service...": "backup.msgStoppingService",
  "Restoring data...": "backup.msgRestoringData",
  "Starting service...": "backup.msgStartingService",
  "Restore completed successfully": "backup.msgRestoreCompleted",
};

const jobFailedPrefixMap: Record<string, string> = {
  "Backup failed": "backup.msgBackupFailed",
  "Restore failed": "backup.msgRestoreFailed",
};

function jobMessage(msg: string): string {
  for (const [prefix, key] of Object.entries(jobFailedPrefixMap)) {
    if (msg.startsWith(prefix)) {
      const detail = msg.slice(prefix.length).replace(/^:\s*/, "");
      return detail ? `${t(key)}: ${detail}` : t(key);
    }
  }
  const key = jobMessageMap[msg];
  return key ? t(key) : msg;
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
  margin-bottom: 24px;
  padding: 20px;
  background: var(--surfacePrimary);
  border: 1px solid var(--borderPrimary);
  border-radius: var(--ui-radius);
  box-shadow: var(--ui-shadow-1);
}

.settings-section h2 {
  margin: 0 0 8px;
  color: var(--text-primary);
  font-size: 1.35rem;
}

.settings-section h3 {
  margin: 8px 0 16px;
  color: var(--text-primary);
  font-size: 1.05rem;
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
  padding: 11px 14px;
  border: 1px solid var(--borderPrimary);
  border-radius: var(--ui-radius-sm);
  background: var(--surfacePrimary);
  color: var(--textPrimary);
  font-size: 14px;
  transition:
    border-color 0.2s,
    box-shadow 0.2s;
}

.form-group input:focus {
  outline: none;
  border-color: var(--blue);
  box-shadow: 0 0 0 3px rgba(33, 150, 243, 0.15);
}

.help-text {
  font-size: 12px;
  color: var(--text-secondary);
}

.btn {
  padding: 10px 20px;
  border: none;
  border-radius: var(--ui-radius-sm);
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
  flex-wrap: wrap;
}

.actions .btn {
  white-space: nowrap;
}

.table-wrap {
  overflow-x: auto;
}

.backup-table {
  width: 100%;
  table-layout: fixed;
  border-collapse: collapse;
}

.backup-table th:first-child {
  width: 36%;
}

.backup-table th:nth-child(2) {
  width: 14%;
}

.backup-table th:nth-child(3) {
  width: 22%;
}

.backup-table th:nth-child(4) {
  width: 28%;
}

.backup-table th,
.backup-table td {
  padding: 12px 16px;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

.backup-table td:first-child {
  overflow-wrap: anywhere;
  word-break: break-word;
}

.backup-table td:nth-child(2),
.backup-table td:nth-child(3) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 736px) {
  .settings-page {
    padding: 12px;
  }

  .settings-section {
    padding: 16px;
  }

  .backup-form {
    max-width: none;
  }

  .backup-form .form-group input {
    font-size: 16px;
  }

  .backup-form .btn-primary {
    width: 100%;
  }

  .table-wrap {
    overflow: visible;
  }

  .backup-table thead {
    display: none;
  }

  .backup-table,
  .backup-table tbody,
  .backup-table tr {
    display: block;
    width: 100%;
  }

  .backup-table tbody {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .backup-table tr {
    margin: 0;
    padding: 12px 14px;
    border: 1px solid var(--border-color);
    border-radius: 10px;
    background: var(--surfacePrimary);
    box-shadow: var(--ui-shadow-1);
  }

  .backup-table td {
    display: block;
    width: 100%;
    padding: 2px 0;
    border-bottom: none;
  }

  .backup-table td:first-child {
    margin-bottom: 4px;
    font-size: 1.05em;
    font-weight: 600;
    word-break: break-all;
  }

  .backup-table td:nth-child(2),
  .backup-table td:nth-child(3) {
    display: inline-block;
    width: auto;
    margin-right: 12px;
    font-size: 0.85em;
    color: var(--text-secondary);
  }

  .backup-table td::before {
    content: attr(data-label);
    margin-right: 4px;
    color: var(--text-secondary);
  }

  .backup-table .actions {
    display: flex;
    gap: 8px;
    margin-top: 8px;
  }

  .backup-table .actions .btn {
    flex: 1;
  }
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
  border-radius: var(--ui-radius-sm);
  background: var(--surfacePrimary);
  border: 1px solid var(--borderPrimary);
}

.job-status__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.job-status__label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: var(--textPrimary);
}

.job-status__dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--icon-orange);
}

.job-status.running .job-status__dot {
  background: var(--blue);
}

.job-status.completed .job-status__dot {
  background: var(--icon-green);
}

.job-status.failed .job-status__dot {
  background: var(--red);
}

.job-status__percent {
  font-weight: 600;
  color: var(--textPrimary);
}

.job-status p {
  margin: 10px 0 4px;
  color: var(--textSecondary);
}

.progress-bar {
  height: 8px;
  background: var(--surfaceSecondary);
  border-radius: 999px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--blue);
  border-radius: 999px;
  transition: width 0.3s ease;
}

.job-status.completed .progress-fill {
  background: var(--icon-green);
}

.job-status.failed .progress-fill {
  background: var(--red);
}

.job-status small {
  display: block;
  margin-top: 2px;
  color: var(--textPrimary);
  font-size: 12px;
  word-break: break-all;
}
</style>
