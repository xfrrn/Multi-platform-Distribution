import { useState } from "react";
import { PackagePlus, Layers3, Save, Archive, RefreshCw } from "lucide-react";
import type { ApiClient, DesktopApp, Release } from "../../api";
import { CHANNELS } from "../../lib/constants";
import { readError } from "../../lib/utils";
import { Panel, PanelTitle, Field, Button, EmptyState } from "../ui";
import "./ReleasesTab.css";

export function ReleasesTab({ api, app, releases, onChanged, onError }: {
  api: ApiClient;
  app: DesktopApp;
  releases: Release[];
  onChanged: () => Promise<void>;
  onError: (e: string) => void;
}) {
  const [creating, setCreating] = useState(false);

  return (
    <div className="detail-grid">
      <Panel className="release-panel">
        <PanelTitle icon={<PackagePlus size={19} />} title="创建版本" />
        <ReleaseForm
          creating={creating}
          onCreate={async (payload) => {
            setCreating(true); onError("");
            try { await api.createRelease(app.id, payload); await onChanged(); }
            catch (err) { onError(readError(err)); }
            finally { setCreating(false); }
          }}
        />
      </Panel>

      <Panel className="release-wide">
        <PanelTitle icon={<Layers3 size={19} />} title="版本管理" />
        {releases.length === 0 ? (
          <EmptyState compact>暂无版本</EmptyState>
        ) : (
          <div className="release-editor-list">
            {releases.map((r) => <ReleaseEditor key={r.id} api={api} release={r} onChanged={onChanged} onError={onError} />)}
          </div>
        )}
      </Panel>
    </div>
  );
}

function ReleaseForm({ creating, onCreate }: {
  creating: boolean;
  onCreate: (p: { version: string; channel: string; changelog: string; is_forced: boolean; staging_percent: number }) => Promise<void>;
}) {
  const [version, setVersion] = useState("");
  const [channel, setChannel] = useState("stable");
  const [changelog, setChangelog] = useState("");
  const [forced, setForced] = useState(false);
  const [staging, setStaging] = useState(100);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    await onCreate({ version, channel, changelog, is_forced: forced, staging_percent: staging });
    setVersion(""); setChangelog("");
  }

  return (
    <form className="stack-form" onSubmit={submit}>
      <div className="two-cols">
        <Field label="版本号"><input value={version} onChange={(e) => setVersion(e.target.value)} placeholder="1.0.0" /></Field>
        <Field label="渠道">
          <select value={channel} onChange={(e) => setChannel(e.target.value)}>
            {CHANNELS.map((c) => <option key={c}>{c}</option>)}
          </select>
        </Field>
      </div>
      <Field label="更新日志"><textarea value={changelog} onChange={(e) => setChangelog(e.target.value)} /></Field>
      <Field label={`灰度比例：${staging}%`}>
        <input type="range" min="0" max="100" value={staging} onChange={(e) => setStaging(Number(e.target.value))} />
      </Field>
      <label className="check-line"><input type="checkbox" checked={forced} onChange={(e) => setForced(e.target.checked)} />强制更新</label>
      <Button variant="primary" disabled={creating} style={{ width: "100%", justifyContent: "center", minHeight: 42 }}>
        {creating ? "创建中..." : "创建版本"}
      </Button>
    </form>
  );
}

function ReleaseEditor({ api, release, onChanged, onError }: {
  api: ApiClient;
  release: Release;
  onChanged: () => Promise<void>;
  onError: (e: string) => void;
}) {
  const [version, setVersion] = useState(release.version);
  const [channel, setChannel] = useState(release.channel);
  const [changelog, setChangelog] = useState(release.changelog);
  const [forced, setForced] = useState(release.is_forced);
  const [staging, setStaging] = useState(release.staging_percent);
  const [saving, setSaving] = useState(false);

  async function save() {
    setSaving(true); onError("");
    try {
      await api.updateRelease(release.id, { version, channel, changelog, is_forced: forced, staging_percent: staging, published_at: release.published_at });
      await onChanged();
    } catch (err) { onError(readError(err)); }
    finally { setSaving(false); }
  }

  async function archive() {
    if (!window.confirm(`归档版本 ${release.version}？`)) return;
    onError("");
    try { await api.archiveRelease(release.id); await onChanged(); }
    catch (err) { onError(readError(err)); }
  }

  return (
    <div className="release-editor">
      <div className="two-cols">
        <Field label="版本号"><input value={version} onChange={(e) => setVersion(e.target.value)} /></Field>
        <Field label="渠道">
          <select value={channel} onChange={(e) => setChannel(e.target.value)}>
            {CHANNELS.map((c) => <option key={c}>{c}</option>)}
          </select>
        </Field>
      </div>
      <Field label="更新日志"><textarea value={changelog} onChange={(e) => setChangelog(e.target.value)} /></Field>
      <div className="two-cols">
        <Field label={`灰度比例：${staging}%`}>
          <input type="range" min="0" max="100" value={staging} onChange={(e) => setStaging(Number(e.target.value))} />
        </Field>
        <label className="check-line"><input type="checkbox" checked={forced} onChange={(e) => setForced(e.target.checked)} />强制更新</label>
      </div>
      <div className="row-actions">
        <Button variant="primary" onClick={() => void save()} disabled={saving}><Save size={17} />保存版本</Button>
        <Button variant="danger" onClick={() => void archive()}><Archive size={17} />归档</Button>
      </div>
    </div>
  );
}
