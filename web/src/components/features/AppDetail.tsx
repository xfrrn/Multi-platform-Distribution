import { useEffect, useState } from "react";
import { Layers3, PackagePlus, FileArchive, MonitorDown, BarChart3, Settings, RefreshCw } from "lucide-react";
import type { ApiClient, DesktopApp, Release, Artifact, ServerConfig } from "../../api";
import type { DetailTab } from "../../lib/types";
import { readError } from "../../lib/utils";
import { TabBar, TabButton, Button, EmptyState } from "../ui";
import { OverviewTab } from "./OverviewTab";
import { ReleasesTab } from "./ReleasesTab";
import { ArtifactsTab } from "./ArtifactsTab";
import { MetadataTab } from "./MetadataTab";
import { AppStatsTab } from "./AppStatsTab";
import { SettingsTab } from "./SettingsTab";
import "./AppDetail.css";

export function AppDetail({ api, initialApp, onAppChanged, onArchived, onCopy }: {
  api: ApiClient;
  initialApp: DesktopApp;
  onAppChanged: (app: DesktopApp) => void;
  onArchived: () => void;
  onCopy: (v: string) => void | Promise<void>;
}) {
  const [app, setApp] = useState(initialApp);
  const [tab, setTab] = useState<DetailTab>("overview");
  const [releases, setReleases] = useState<Release[]>([]);
  const [artifacts, setArtifacts] = useState<Record<string, Artifact[]>>({});
  const [selectedRelease, setSelectedRelease] = useState("");
  const [config, setConfig] = useState<ServerConfig>({ anyshare_enabled: false, public_base_url: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => { setApp(initialApp); void refresh(); }, [initialApp.id]);

  async function refresh() {
    setLoading(true); setError("");
    try {
      const [cfg, releases] = await Promise.all([
        api.getConfig().catch(() => ({ anyshare_enabled: false, public_base_url: "" })),
        api.listReleases(initialApp.id),
      ]);
      setConfig(cfg); setReleases(releases);
      setSelectedRelease((cur) => cur || releases[0]?.id || "");
      const map: Record<string, Artifact[]> = {};
      await Promise.all(releases.map(async (r) => { map[r.id] = await api.listArtifacts(r.id); }));
      setArtifacts(map);
    } catch (err) { setError(readError(err)); }
    finally { setLoading(false); }
  }

  const allArtifacts = Object.values(artifacts).flat();

  return (
    <section className="detail-view">
      {error && <div className="notice error">{error}</div>}
      <TabBar>
        <TabButton active={tab === "overview"} icon={<Layers3 size={17} />} label="概览" onClick={() => setTab("overview")} />
        <TabButton active={tab === "releases"} icon={<PackagePlus size={17} />} label="版本" onClick={() => setTab("releases")} />
        <TabButton active={tab === "artifacts"} icon={<FileArchive size={17} />} label="安装包" onClick={() => setTab("artifacts")} />
        <TabButton active={tab === "metadata"} icon={<MonitorDown size={17} />} label="元数据" onClick={() => setTab("metadata")} />
        <TabButton active={tab === "stats"} icon={<BarChart3 size={17} />} label="统计" onClick={() => setTab("stats")} />
        <TabButton active={tab === "settings"} icon={<Settings size={17} />} label="设置" onClick={() => setTab("settings")} />
        <Button variant="ghost" className="tab-refresh" onClick={() => void refresh()}><RefreshCw size={17} />刷新详情</Button>
      </TabBar>

      {loading && <EmptyState compact>正在同步应用数据...</EmptyState>}

      {tab === "overview" && <OverviewTab app={app} publicBaseURL={config.public_base_url} releases={releases} artifacts={allArtifacts} />}
      {tab === "releases" && <ReleasesTab api={api} app={app} releases={releases} onChanged={refresh} onError={setError} />}
      {tab === "artifacts" && <ArtifactsTab api={api} releases={releases} artifacts={artifacts} anyshareEnabled={config.anyshare_enabled} selectedRelease={selectedRelease} onRelease={setSelectedRelease} onChanged={refresh} onError={setError} onCopy={onCopy} />}
      {tab === "metadata" && <MetadataTab api={api} app={app} publicBaseURL={config.public_base_url} onCopy={onCopy} />}
      {tab === "stats" && <AppStatsTab api={api} app={app} />}
      {tab === "settings" && <SettingsTab api={api} app={app} onSaved={(a) => { setApp(a); onAppChanged(a); }} onArchived={onArchived} onError={setError} />}
    </section>
  );
}
