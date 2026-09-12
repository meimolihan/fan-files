import { fetchJSON } from "./utils";

export interface BackupInfo {
  name: string;
  path: string;
  size: number;
  modTime: string;
}

export interface BackupListResponse {
  backups: BackupInfo[];
  dir: string;
}

export interface BackupRequest {
  dir?: string;
  keepNum?: number;
  async?: boolean;
}

export interface BackupRestoreRequest {
  dir?: string;
  file?: string;
}

export interface BackupResponse {
  success: boolean;
  message: string;
  jobId?: string;
  status?: string;
  progress?: number;
  file?: string;
}

export interface JobStatusResponse {
  success: boolean;
  message: string;
  jobId?: string;
  status?: string;
  progress?: number;
  file?: string;
}

export function listBackups(): Promise<BackupListResponse> {
  return fetchJSON("/api/backup");
}

export function createBackup(data: BackupRequest): Promise<BackupResponse> {
  return fetchJSON("/api/backup", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export function restoreBackup(data: BackupRestoreRequest): Promise<BackupResponse> {
  return fetchJSON("/api/backup/restore", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export function deleteBackup(filename: string): Promise<BackupResponse> {
  return fetchJSON(`/api/backup/${encodeURIComponent(filename)}`, {
    method: "DELETE",
  });
}

export function getJobStatus(jobId: string): Promise<JobStatusResponse> {
  return fetchJSON(`/api/backup/job/${jobId}`);
}