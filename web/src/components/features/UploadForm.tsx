import { type FormEvent, useState, useEffect } from "react";
import { UploadCloud } from "lucide-react";
import type { Release } from "../../api";
import { PLATFORMS, ARCHES, FILE_TYPES } from "../../lib/constants";
import { Field, Button } from "../ui";
import type { ArtifactSource, ArtifactUploadPayload } from "../../lib/types";

export function UploadForm({ releases, selectedRelease, anyshareEnabled, onRelease, onUpload }: {
  releases: Release[];
  selectedRelease: string;
  anyshareEnabled: boolean;
  onRelease: (id: string) => void;
  onUpload: (p: ArtifactUploadPayload) => void | Promise<void>;
}) {
  const [platform, setPlatform] = useState("windows");
  const [arch, setArch] = useState("x64");
  const [fileType, setFileType] = useState("exe");
  const [sourceType, setSourceType] = useState<ArtifactSource>("managed");
  const [file, setFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => { if (!anyshareEnabled && sourceType === "anyshare") setSourceType("managed"); }, [anyshareEnabled, sourceType]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!file) return;
    setSubmitting(true);
    try { await onUpload({ platform, arch, file_type: fileType, source_type: sourceType, file }); setFile(null); }
    finally { setSubmitting(false); }
  }

  return (
    <form className="stack-form" onSubmit={submit}>
      <Field label="版本">
        <select value={selectedRelease} onChange={(e) => onRelease(e.target.value)}>
          <option value="">选择版本</option>
          {releases.map((r) => <option key={r.id} value={r.id}>{r.version} / {r.channel}</option>)}
        </select>
      </Field>
      <Field label="存储来源">
        <select value={sourceType} onChange={(e) => setSourceType(e.target.value as ArtifactSource)}>
          <option value="managed">默认存储</option>
          {anyshareEnabled && <option value="anyshare">Anyshare 实验</option>}
        </select>
      </Field>
      <div className="three-cols">
        <Field label="平台"><select value={platform} onChange={(e) => setPlatform(e.target.value)}>{PLATFORMS.map((p) => <option key={p}>{p}</option>)}</select></Field>
        <Field label="架构"><select value={arch} onChange={(e) => setArch(e.target.value)}>{ARCHES.map((a) => <option key={a}>{a}</option>)}</select></Field>
        <Field label="类型"><select value={fileType} onChange={(e) => setFileType(e.target.value)}>{FILE_TYPES.map((f) => <option key={f}>{f}</option>)}</select></Field>
      </div>
      <label className="file-drop">
        <UploadCloud size={22} />
        <span>{file ? file.name : "选择安装包文件"}</span>
        <input type="file" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
      </label>
      <Button variant="primary" disabled={!file || !selectedRelease || submitting} style={{ width: "100%", justifyContent: "center", minHeight: 42 }}>
        {submitting ? "上传中..." : "上传安装包"}
      </Button>
    </form>
  );
}
