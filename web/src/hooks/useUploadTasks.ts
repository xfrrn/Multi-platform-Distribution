import { useState, useEffect } from "react";
import type { ApiClient, ServerUploadProgress } from "../api";
import type { ArtifactUploadTask, ArtifactUploadPayload, UploadProgress } from "../lib/types";
import { readError } from "../lib/utils";
import { ApiError } from "../api";

const listeners = new Set<() => void>();
let tasks: ArtifactUploadTask[] = [];

function notify() {
  for (const fn of listeners) fn();
}

export function addUploadTask(task: ArtifactUploadTask) {
  tasks = [task, ...tasks].slice(0, 8);
  notify();
}

export function updateUploadTask(id: string, patch: Partial<ArtifactUploadTask>) {
  tasks = tasks.map((t) => (t.id === id ? { ...t, ...patch } : t));
  notify();
}

export function useUploadTasks() {
  const [state, setState] = useState(tasks);
  useEffect(() => {
    const fn = () => setState([...tasks]);
    listeners.add(fn);
    return () => { listeners.delete(fn); };
  }, []);
  return state;
}

export function startUploadTask(
  api: ApiClient,
  releaseId: string,
  payload: ArtifactUploadPayload,
  onFinished: () => void | Promise<void>,
  onFailed: (err: unknown) => void
) {
  const id = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
  const uploadPayload = payload.source_type === "anyshare" ? { ...payload, upload_id: id } : payload;
  let progressTimer: number | undefined;

  addUploadTask({
    id, releaseId,
    fileName: payload.file.name,
    fileSize: payload.file.size,
    target: `${payload.platform}/${payload.arch}/${payload.file_type}`,
    loaded: 0, total: payload.file.size, progress: 0,
    status: "uploading",
  });

  if (payload.source_type === "anyshare") {
    progressTimer = window.setInterval(() => {
      void api.getUploadProgress(id).then((p) => applyServerProgress(id, payload.file.size, p)).catch((err) => {
        if (err instanceof ApiError && err.status === 404) return;
        window.clearInterval(progressTimer);
      });
    }, 800);
  }

  const handleProgress = (p: UploadProgress) => {
    updateUploadTask(id, {
      loaded: p.loaded, total: p.total || payload.file.size, progress: p.percent,
      status: p.lengthComputable && p.percent >= 100 ? "processing" : "uploading",
      serverPhase: p.lengthComputable && p.percent >= 100 && payload.source_type === "anyshare" ? "received" : undefined,
    });
  };

  void api.uploadArtifact(releaseId, uploadPayload, handleProgress)
    .then(() => {
      if (progressTimer !== undefined) window.clearInterval(progressTimer);
      updateUploadTask(id, { loaded: payload.file.size, progress: 100, status: "done", serverPhase: "done" });
      void Promise.resolve(onFinished()).catch(onFailed);
    })
    .catch((err) => {
      if (progressTimer !== undefined) window.clearInterval(progressTimer);
      updateUploadTask(id, { status: "error", error: readError(err) });
      onFailed(err);
    });
}

function applyServerProgress(id: string, fileSize: number, p: ServerUploadProgress) {
  if (p.status === "error") {
    updateUploadTask(id, { status: "error", serverPhase: p.phase, error: p.error || "Anyshare 上传失败" });
    return;
  }
  updateUploadTask(id, {
    loaded: p.loaded, total: p.total || fileSize, progress: p.percent,
    status: p.status === "done" ? "done" : "processing",
    serverPhase: p.phase,
  });
}
