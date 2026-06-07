import { useState } from "react";
import { Copy, Pencil, UploadCloud, Archive, Save, ArrowLeft } from "lucide-react";
import type { Release, Artifact } from "../../api";
import { PLATFORMS, ARCHES, FILE_TYPES } from "../../lib/constants";
import { formatBytes } from "../../lib/utils";
import { Badge } from "../ui";

type Patch = { platform: string; arch: string; file_type: string; file_name: string };

export function ArtifactTable({ artifacts, releases, readonly, onCopy, onArchive, onSave, onReplace }: {
  artifacts: Artifact[];
  releases: Release[];
  readonly?: boolean;
  onCopy?: (v: string) => void | Promise<void>;
  onArchive?: (a: Artifact) => Promise<void>;
  onSave?: (a: Artifact, p: Patch) => Promise<void>;
  onReplace?: (a: Artifact, f: File) => Promise<void>;
}) {
  return (
    <div className="table-wrap">
      <table className="data-table">
        <thead>
          <tr>
            <th>文件名</th>
            <th>版本</th>
            <th>目标</th>
            <th>大小</th>
            <th>来源</th>
            <th>SHA512</th>
            {!readonly && <th>操作</th>}
          </tr>
        </thead>
        <tbody>
          {artifacts.map((a) => {
            const r = releases.find((x) => x.id === a.release_id);
            return <ArtifactRow key={a.id} artifact={a} release={r} readonly={readonly} onCopy={onCopy} onArchive={onArchive} onSave={onSave} onReplace={onReplace} />;
          })}
        </tbody>
      </table>
    </div>
  );
}

function ArtifactRow({ artifact, release, readonly, onCopy, onArchive, onSave, onReplace }: {
  artifact: Artifact;
  release?: Release;
  readonly?: boolean;
  onCopy?: (v: string) => void | Promise<void>;
  onArchive?: (a: Artifact) => Promise<void>;
  onSave?: (a: Artifact, p: Patch) => Promise<void>;
  onReplace?: (a: Artifact, f: File) => Promise<void>;
}) {
  const [editing, setEditing] = useState(false);
  const [fileName, setFileName] = useState(artifact.file_name);
  const [platform, setPlatform] = useState(artifact.platform);
  const [arch, setArch] = useState(artifact.arch);
  const [fileType, setFileType] = useState(artifact.file_type);

  if (!editing || readonly) {
    return (
      <tr>
        <td>{artifact.file_name || artifact.storage_key}</td>
        <td>{release?.version ?? "-"}</td>
        <td>{artifact.platform}/{artifact.arch}/{artifact.file_type}</td>
        <td>{formatBytes(artifact.file_size)}</td>
        <td><Badge>{artifact.source_type === "anyshare" ? "Anyshare" : "默认"}</Badge></td>
        <td className="mono">{artifact.sha512.slice(0, 28)}...</td>
        {!readonly && (
          <td>
            <div className="table-actions">
              <button className="icon-btn" title="复制下载链接" onClick={() => { void onCopy?.(artifact.file_url); }}><Copy size={15} /></button>
              <button className="icon-btn" title="编辑" onClick={() => setEditing(true)}><Pencil size={15} /></button>
              <label className="icon-btn" title="重新上传" style={{ cursor: "pointer", margin: 0 }}>
                <UploadCloud size={15} />
                <input type="file" style={{ display: "none" }} onChange={(e) => {
                  const f = e.target.files?.[0]; if (f) { void onReplace?.(artifact, f); } e.currentTarget.value = "";
                }} />
              </label>
              <button className="icon-btn" title="归档" style={{ color: "var(--danger)" }} onClick={() => { void onArchive?.(artifact); }}><Archive size={15} /></button>
            </div>
          </td>
        )}
      </tr>
    );
  }

  return (
    <tr>
      <td><input value={fileName} onChange={(e) => setFileName(e.target.value)} style={{ width: "100%", minHeight: 32, padding: "4px 6px" }} /></td>
      <td>{release?.version ?? "-"}</td>
      <td>
        <div className="artifact-edit-grid">
          <select value={platform} onChange={(e) => setPlatform(e.target.value)}>{PLATFORMS.map((p) => <option key={p}>{p}</option>)}</select>
          <select value={arch} onChange={(e) => setArch(e.target.value)}>{ARCHES.map((a) => <option key={a}>{a}</option>)}</select>
          <select value={fileType} onChange={(e) => setFileType(e.target.value)}>{FILE_TYPES.map((f) => <option key={f}>{f}</option>)}</select>
        </div>
      </td>
      <td>{formatBytes(artifact.file_size)}</td>
      <td><Badge>{artifact.source_type === "anyshare" ? "Anyshare" : "默认"}</Badge></td>
      <td className="mono">{artifact.sha512.slice(0, 28)}...</td>
      <td>
        <div className="table-actions">
          <button className="icon-btn" title="保存" onClick={() => { setEditing(false); void onSave?.(artifact, { file_name: fileName, platform, arch, file_type: fileType }); }}><Save size={15} /></button>
          <button className="icon-btn" title="取消" onClick={() => { setEditing(false); setFileName(artifact.file_name); setPlatform(artifact.platform); setArch(artifact.arch); setFileType(artifact.file_type); }}><ArrowLeft size={15} /></button>
        </div>
      </td>
    </tr>
  );
}
