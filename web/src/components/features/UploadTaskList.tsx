import type { Release } from "../../api";
import { formatBytes, uploadTaskStatus } from "../../lib/utils";
import type { ArtifactUploadTask } from "../../lib/types";

export function UploadTaskList({ tasks, releases }: { tasks: ArtifactUploadTask[]; releases: Release[] }) {
  if (tasks.length === 0) return null;
  const labels = new Map(releases.map((r) => [r.id, `${r.version} / ${r.channel}`]));
  return (
    <div className="upload-task-list" aria-live="polite">
      {tasks.map((t) => (
        <div className={`upload-task ${t.status}`} key={t.id}>
          <div className="upload-task-hd"><strong>{t.fileName}</strong><span>{uploadTaskStatus(t)}</span></div>
          <div className="upload-task-meta">
            <span>{labels.get(t.releaseId) ?? t.releaseId}</span>
            <span>{t.target}</span>
            <span>{formatBytes(t.fileSize)}</span>
          </div>
          <div className="upload-progress" role="progressbar" aria-valuenow={t.progress} aria-valuemin={0} aria-valuemax={100}>
            <span style={{ width: `${Math.max(t.progress, (t.status === "uploading" || t.status === "processing") ? 2 : 0)}%` }} />
          </div>
          {t.error && <div className="upload-task-error">{t.error}</div>}
        </div>
      ))}
    </div>
  );
}
