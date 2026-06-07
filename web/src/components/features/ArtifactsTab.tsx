import { useState } from "react";
import { FileUp, FileArchive, Copy, Pencil, UploadCloud, Archive, Save, ArrowLeft } from "lucide-react";
import type { ApiClient, Release, Artifact } from "../../api";
import { PLATFORMS, ARCHES } from "../../lib/constants";
import { readError } from "../../lib/utils";
import { useUploadTasks, startUploadTask } from "../../hooks/useUploadTasks";
import { Panel, PanelTitle, Button, EmptyState } from "../ui";
import { UploadForm } from "./UploadForm";
import { UploadTaskList } from "./UploadTaskList";
import { ArtifactTable } from "./ArtifactTable";
import "./ArtifactsTab.css";

export function ArtifactsTab({ api, releases, artifacts, anyshareEnabled, selectedRelease, onRelease, onChanged, onError, onCopy }: {
  api: ApiClient;
  releases: Release[];
  artifacts: Record<string, Artifact[]>;
  anyshareEnabled: boolean;
  selectedRelease: string;
  onRelease: (id: string) => void;
  onChanged: () => Promise<void>;
  onError: (e: string) => void;
  onCopy: (v: string) => void | Promise<void>;
}) {
  const [platformFilter, setPlatformFilter] = useState("");
  const [archFilter, setArchFilter] = useState("");
  const uploadTasks = useUploadTasks();
  const releaseIds = new Set(releases.map((r) => r.id));
  const visibleTasks = uploadTasks.filter((t) => releaseIds.has(t.releaseId));
  const visible = (selectedRelease ? artifacts[selectedRelease] ?? [] : Object.values(artifacts).flat())
    .filter((a) => !platformFilter || a.platform === platformFilter)
    .filter((a) => !archFilter || a.arch === archFilter);

  return (
    <div className="detail-grid">
      <Panel className="upload-panel">
        <PanelTitle icon={<FileUp size={19} />} title="上传安装包" />
        <UploadForm
          releases={releases}
          selectedRelease={selectedRelease}
          anyshareEnabled={anyshareEnabled}
          onRelease={onRelease}
          onUpload={(payload) => {
            if (!selectedRelease) { onError("请先创建或选择一个版本"); return; }
            onError("");
            startUploadTask(api, selectedRelease, payload, onChanged, (err) => onError(readError(err)));
          }}
        />
        <UploadTaskList tasks={visibleTasks} releases={releases} />
      </Panel>

      <Panel className="wide-panel">
        <PanelTitle icon={<FileArchive size={19} />} title="安装包列表" />
        <div className="filter-row">
          <select value={selectedRelease} onChange={(e) => onRelease(e.target.value)}>
            <option value="">全部版本</option>
            {releases.map((r) => <option key={r.id} value={r.id}>{r.version} / {r.channel}</option>)}
          </select>
          <select value={platformFilter} onChange={(e) => setPlatformFilter(e.target.value)}>
            <option value="">全部平台</option>
            {PLATFORMS.map((p) => <option key={p}>{p}</option>)}
          </select>
          <select value={archFilter} onChange={(e) => setArchFilter(e.target.value)}>
            <option value="">全部架构</option>
            {ARCHES.map((a) => <option key={a}>{a}</option>)}
          </select>
        </div>
        {visible.length === 0 ? (
          <EmptyState compact>暂无安装包</EmptyState>
        ) : (
          <ArtifactTable
            artifacts={visible} releases={releases} onCopy={onCopy}
            onArchive={async (a) => { if (!window.confirm(`归档 ${a.file_name || a.id}？`)) return; onError(""); try { await api.archiveArtifact(a.id); await onChanged(); } catch (err) { onError(readError(err)); } }}
            onSave={async (a, patch) => { onError(""); try { await api.updateArtifact(a.id, patch); await onChanged(); } catch (err) { onError(readError(err)); } }}
            onReplace={async (a, file) => { onError(""); try { await api.replaceArtifactFile(a.id, file); await onChanged(); } catch (err) { onError(readError(err)); } }}
          />
        )}
      </Panel>
    </div>
  );
}
