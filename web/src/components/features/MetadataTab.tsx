import { useState, useEffect } from "react";
import { MonitorDown, Copy, RefreshCw } from "lucide-react";
import type { ApiClient, DesktopApp } from "../../api";
import { CHANNELS, PLATFORMS, ARCHES } from "../../lib/constants";
import { readError, metadataParams, absoluteMetadataURL } from "../../lib/utils";
import type { MetaFormat } from "../../lib/types";
import { Panel, PanelTitle, Field, Button } from "../ui";
import "./MetadataTab.css";

export function MetadataTab({ api, app, publicBaseURL, onCopy }: {
  api: ApiClient;
  app: DesktopApp;
  publicBaseURL: string;
  onCopy: (v: string) => void | Promise<void>;
}) {
  const [format, setFormat] = useState<MetaFormat>("json");
  const [channel, setChannel] = useState(app.default_channel);
  const [platform, setPlatform] = useState("");
  const [arch, setArch] = useState("");
  const [clientID, setClientID] = useState("");
  const [text, setText] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function preview(fmt = format) {
    setFormat(fmt); setLoading(true); setError("");
    const params = metadataParams(channel, platform, arch, clientID);
    try {
      if (fmt === "json") {
        const m = await api.getManifest(app.slug, params);
        setText(JSON.stringify(m, null, 2));
      } else {
        const suffix = fmt === "yml" ? "latest.yml" : "appcast.xml";
        setText(await api.getText(`/api/latest/${app.slug}/${suffix}?${params}`));
      }
    } catch (err) { setText(""); setError(readError(err)); }
    finally { setLoading(false); }
  }

  const url = (() => {
    const suffix = format === "json" ? "update.json" : format === "yml" ? "latest.yml" : "appcast.xml";
    const q = metadataParams(channel, platform, arch, clientID).toString();
    return `${absoluteMetadataURL(publicBaseURL, app.slug, suffix)}${q ? `?${q}` : ""}`;
  })();

  return (
    <Panel className="meta-panel">
      <PanelTitle icon={<MonitorDown size={19} />} title="元数据预览" />
      {error && <div className="notice error">{error}</div>}
      <div className="meta-filter-grid">
        <Field label="渠道">
          <select value={channel} onChange={(e) => setChannel(e.target.value)}>{CHANNELS.map((c) => <option key={c}>{c}</option>)}</select>
        </Field>
        <Field label="平台">
          <select value={platform} onChange={(e) => setPlatform(e.target.value)}>
            <option value="">全部平台</option>{PLATFORMS.map((p) => <option key={p}>{p}</option>)}
          </select>
        </Field>
        <Field label="架构">
          <select value={arch} onChange={(e) => setArch(e.target.value)}>
            <option value="">全部架构</option>{ARCHES.map((a) => <option key={a}>{a}</option>)}
          </select>
        </Field>
        <Field label="Client ID">
          <input value={clientID} onChange={(e) => setClientID(e.target.value)} placeholder="灰度测试客户端标识" />
        </Field>
      </div>
      <div className="meta-controls">
        {(["json", "yml", "xml"] as MetaFormat[]).map((f) => (
          <button key={f} className={`meta-seg ${format === f ? "meta-seg-active" : ""}`} onClick={() => { void preview(f); }}>
            {f === "json" ? "update.json" : f === "yml" ? "latest.yml" : "appcast.xml"}
          </button>
        ))}
        <button className="icon-btn" title="复制 URL" onClick={() => { void onCopy(url); }}><Copy size={16} /></button>
      </div>
      <div className="url-line">{url}</div>
      <pre className="meta-box">{loading ? "正在读取..." : text || "选择格式后预览该应用的更新元数据。"}</pre>
    </Panel>
  );
}
