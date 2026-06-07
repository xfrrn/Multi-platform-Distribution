import { type FormEvent, useState } from "react";
import { Settings, Save, Archive } from "lucide-react";
import type { ApiClient, DesktopApp } from "../../api";
import { CHANNELS } from "../../lib/constants";
import { readError } from "../../lib/utils";
import { Panel, PanelTitle, Field, Button } from "../ui";

export function SettingsTab({ api, app, onSaved, onArchived, onError }: {
  api: ApiClient;
  app: DesktopApp;
  onSaved: (a: DesktopApp) => void;
  onArchived: () => void;
  onError: (e: string) => void;
}) {
  const [name, setName] = useState(app.name);
  const [desc, setDesc] = useState(app.description);
  const [iconURL, setIconURL] = useState(app.icon_url);
  const [channel, setChannel] = useState(app.default_channel);
  const [saving, setSaving] = useState(false);

  async function save(e: FormEvent) {
    e.preventDefault(); setSaving(true); onError("");
    try { const a = await api.updateApp(app.id, { name, description: desc, icon_url: iconURL, default_channel: channel }); onSaved(a); }
    catch (err) { onError(readError(err)); }
    finally { setSaving(false); }
  }

  async function archive() {
    if (!window.confirm(`归档应用 ${app.name}？`)) return;
    onError("");
    try { await api.archiveApp(app.id); onArchived(); } catch (err) { onError(readError(err)); }
  }

  return (
    <Panel>
      <PanelTitle icon={<Settings size={19} />} title="应用设置" />
      <form className="stack-form" onSubmit={save}>
        <Field label="应用名称"><input value={name} onChange={(e) => setName(e.target.value)} /></Field>
        <Field label="描述"><textarea value={desc} onChange={(e) => setDesc(e.target.value)} /></Field>
        <Field label="图标 URL"><input value={iconURL} onChange={(e) => setIconURL(e.target.value)} /></Field>
        <Field label="默认渠道">
          <select value={channel} onChange={(e) => setChannel(e.target.value)}>{CHANNELS.map((c) => <option key={c}>{c}</option>)}</select>
        </Field>
        <div className="row-actions">
          <Button variant="primary" disabled={saving}><Save size={17} />保存设置</Button>
          <Button variant="danger" type="button" onClick={() => void archive()}><Archive size={17} />归档应用</Button>
        </div>
      </form>
    </Panel>
  );
}
