import { useState, useEffect } from "react";
import { SlidersHorizontal, BarChart3, RefreshCw } from "lucide-react";
import type { ApiClient, StatsSummary, UpdateRequestEvent, DownloadEvent } from "../../api";
import { PLATFORMS, ARCHES, CHANNELS } from "../../lib/constants";
import { readError, statsParams } from "../../lib/utils";
import { Panel, PanelTitle, Field, Button, EmptyState } from "../ui";
import { StatsContent } from "./StatsContent";
import "./StatsView.css";

export function StatsView({ api }: { api: ApiClient }) {
  const [summary, setSummary] = useState<StatsSummary | null>(null);
  const [updates, setUpdates] = useState<UpdateRequestEvent[]>([]);
  const [downloads, setDownloads] = useState<DownloadEvent[]>([]);
  const [channel, setChannel] = useState("");
  const [platform, setPlatform] = useState("");
  const [arch, setArch] = useState("");
  const [clientID, setClientID] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => { void refresh(); }, []);

  async function refresh() {
    setLoading(true); setError("");
    const params = statsParams(channel, platform, arch, clientID, 20);
    try {
      const [s, u, d] = await Promise.all([api.getStatsSummary(params), api.listUpdateRequests(undefined, params), api.listDownloads(undefined, params)]);
      setSummary(s); setUpdates(u); setDownloads(d);
    } catch (err) { setError(readError(err)); } finally { setLoading(false); }
  }

  return (
    <section className="stats-page">
      <Panel>
        <PanelTitle icon={<SlidersHorizontal size={19} />} title="筛选" />
        <div className="stats-filter-grid">
          <Field label="渠道"><select value={channel} onChange={(e) => setChannel(e.target.value)}><option value="">全部渠道</option>{CHANNELS.map((c) => <option key={c}>{c}</option>)}</select></Field>
          <Field label="平台"><select value={platform} onChange={(e) => setPlatform(e.target.value)}><option value="">全部平台</option>{PLATFORMS.map((p) => <option key={p}>{p}</option>)}</select></Field>
          <Field label="架构"><select value={arch} onChange={(e) => setArch(e.target.value)}><option value="">全部架构</option>{ARCHES.map((a) => <option key={a}>{a}</option>)}</select></Field>
          <Field label="Client ID"><input value={clientID} onChange={(e) => setClientID(e.target.value)} placeholder="按客户端过滤" /></Field>
          <Button variant="primary" className="stats-apply" onClick={() => void refresh()}><RefreshCw size={17} />应用筛选</Button>
        </div>
      </Panel>
      {error && <div className="notice error">{error}</div>}
      {loading && <EmptyState compact>正在加载统计数据...</EmptyState>}
      {summary && <StatsContent summary={summary} updates={updates} downloads={downloads} />}
    </section>
  );
}

import type { DesktopApp } from "../../api";

export function AppStatsTab({ api, app }: { api: ApiClient; app: DesktopApp }) {
  const [summary, setSummary] = useState<StatsSummary | null>(null);
  const [updates, setUpdates] = useState<UpdateRequestEvent[]>([]);
  const [downloads, setDownloads] = useState<DownloadEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => { void refresh(); }, [app.id]);

  async function refresh() {
    setLoading(true); setError("");
    const params = statsParams("", "", "", "", 20);
    try {
      const [s, u, d] = await Promise.all([api.getAppStatsSummary(app.id, params), api.listUpdateRequests(app.id, params), api.listDownloads(app.id, params)]);
      setSummary(s); setUpdates(u); setDownloads(d);
    } catch (err) { setError(readError(err)); } finally { setLoading(false); }
  }

  return (
    <section className="stats-page">
      <div className="stats-toolbar">
        <PanelTitle icon={<BarChart3 size={19} />} title="应用统计" />
        <Button variant="ghost" onClick={() => void refresh()}><RefreshCw size={17} />刷新统计</Button>
      </div>
      {error && <div className="notice error">{error}</div>}
      {loading && <EmptyState compact>正在加载统计数据...</EmptyState>}
      {summary && <StatsContent summary={summary} updates={updates} downloads={downloads} appScoped />}
    </section>
  );
}
