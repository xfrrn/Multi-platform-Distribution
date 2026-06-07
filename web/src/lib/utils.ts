import { ApiError } from "../api";
import type { ArtifactUploadTask } from "./types";

/** Human-readable upload task status */
export function uploadTaskStatus(task: ArtifactUploadTask): string {
  if (task.status === "processing" && task.serverPhase === "received") return "Anyshare 准备中";
  if (task.status === "processing" && task.serverPhase === "anyshare") return `Anyshare ${task.progress}%`;
  if (task.status === "processing" && task.serverPhase === "finalizing") return "Anyshare 收尾中";
  if (task.status === "done") return "完成";
  if (task.status === "error") return "失败";
  if (task.status === "processing") return "服务器处理中";
  if (task.progress > 0) return `${task.progress}%`;
  return task.loaded > 0 ? formatBytes(task.loaded) : "等待上传";
}

/** File size to human string */
export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

/** ISO string → locale date-time */
export function formatDateTime(value: string): string {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

/** Extract error message from any thrown value */
export function readError(err: unknown): string {
  if (err instanceof ApiError && err.status === 401) {
    return "登录已失效或没有权限";
  }
  if (err instanceof Error) {
    return err.message;
  }
  return "操作失败";
}

/** Build absolute metadata URL from public base + path */
export function absoluteMetadataURL(publicBaseURL: string, slug: string, suffix: string): string {
  const path = `/api/latest/${slug}/${suffix}`;
  if (!publicBaseURL) return path;
  return `${publicBaseURL.replace(/\/+$/, "")}${path}`;
}

/** Build stats query params */
export function statsParams(
  channel: string,
  platform: string,
  arch: string,
  clientID: string,
  limit: number
): URLSearchParams {
  const params = new URLSearchParams();
  if (channel) params.set("channel", channel);
  if (platform) params.set("platform", platform);
  if (arch) params.set("arch", arch);
  if (clientID) params.set("client_id", clientID);
  params.set("limit", String(limit));
  return params;
}

/** Build metadata query params */
export function metadataParams(
  channel: string,
  platform: string,
  arch: string,
  clientID: string
): URLSearchParams {
  const params = new URLSearchParams();
  if (channel) params.set("channel", channel);
  if (platform) params.set("platform", platform);
  if (arch) params.set("arch", arch);
  if (clientID) params.set("client_id", clientID);
  return params;
}
