import {
  Archive,
  ArrowLeft,
  BarChart3,
  Boxes,
  ChevronRight,
  Copy,
  Download,
  FileArchive,
  FileUp,
  KeyRound,
  Layers3,
  LogOut,
  MonitorDown,
  PackagePlus,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Search,
  Server,
  Settings,
  SlidersHorizontal,
  UploadCloud
} from "lucide-react";
import { FormEvent, ReactNode, useEffect, useMemo, useState } from "react";
import {
  ApiClient,
  ApiError,
  Artifact,
  DesktopApp,
  DownloadEvent,
  Release,
  ServerConfig,
  StatsBreakdown,
  StatsPoint,
  StatsSummary,
  ServerUploadProgress,
  UpdateManifest,
  UpdateRequestEvent,
  UploadProgress
} from "./api";

type View = "home" | "detail" | "stats";
type DetailTab = "overview" | "releases" | "artifacts" | "metadata" | "stats" | "settings";
type MetaFormat = "json" | "yml" | "xml";
type ArtifactSource = "managed" | "anyshare";
type UploadTaskStatus = "uploading" | "processing" | "done" | "error";
type ArtifactUploadPayload = { platform: string; arch: string; file_type: string; source_type: ArtifactSource; upload_id?: string; file: File };
type ArtifactUploadTask = {
  id: string;
  releaseId: string;
  fileName: string;
  fileSize: number;
  target: string;
  loaded: number;
  total: number;
  progress: number;
  status: UploadTaskStatus;
  serverPhase?: string;
  error?: string;
};

const tokenKey = "mpd.adminToken";
const channels = ["stable", "beta", "internal"];
const platforms = ["windows", "macos", "linux"];
const arches = ["x64", "arm64", "universal"];
const fileTypes = ["exe", "dmg", "msi", "zip", "AppImage"];
const uploadTaskListeners = new Set<() => void>();
let artifactUploadTasks: ArtifactUploadTask[] = [];

type AppStats = {
  latestVersion: string;
  artifactCount: number;
  platforms: string[];
};

function useArtifactUploadTasks() {
  const [tasks, setTasks] = useState<ArtifactUploadTask[]>(artifactUploadTasks);

  useEffect(() => {
    const listener = () => setTasks([...artifactUploadTasks]);
    uploadTaskListeners.add(listener);
    return () => {
      uploadTaskListeners.delete(listener);
    };
  }, []);

  return tasks;
}

function notifyArtifactUploadTasks() {
  for (const listener of uploadTaskListeners) {
    listener();
  }
}

function addArtifactUploadTask(task: ArtifactUploadTask) {
  artifactUploadTasks = [task, ...artifactUploadTasks].slice(0, 8);
  notifyArtifactUploadTasks();
}

function updateArtifactUploadTask(id: string, patch: Partial<ArtifactUploadTask>) {
  artifactUploadTasks = artifactUploadTasks.map((task) => task.id === id ? { ...task, ...patch } : task);
  notifyArtifactUploadTasks();
}

function startArtifactUpload(
  api: ApiClient,
  releaseId: string,
  payload: ArtifactUploadPayload,
  onFinished: () => void | Promise<void>,
  onFailed: (err: unknown) => void
) {
  const id = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
  const uploadPayload = payload.source_type === "anyshare" ? { ...payload, upload_id: id } : payload;
  let progressTimer: number | undefined;
  addArtifactUploadTask({
    id,
    releaseId,
    fileName: payload.file.name,
    fileSize: payload.file.size,
    target: `${payload.platform}/${payload.arch}/${payload.file_type}`,
    loaded: 0,
    total: payload.file.size,
    progress: 0,
    status: "uploading"
  });

  if (payload.source_type === "anyshare") {
    progressTimer = window.setInterval(() => {
      void api.getUploadProgress(id)
        .then((progress) => {
          applyServerUploadProgress(id, payload.file.size, progress);
        })
        .catch((err) => {
          if (err instanceof ApiError && err.status === 404) return;
          window.clearInterval(progressTimer);
        });
    }, 800);
  }

  const handleProgress = (progress: UploadProgress) => {
    updateArtifactUploadTask(id, {
      loaded: progress.loaded,
      total: progress.total || payload.file.size,
      progress: progress.percent,
      status: progress.lengthComputable && progress.percent >= 100 ? "processing" : "uploading",
      serverPhase: progress.lengthComputable && progress.percent >= 100 && payload.source_type === "anyshare" ? "received" : undefined
    });
  };

  void api.uploadArtifact(releaseId, uploadPayload, handleProgress)
    .then(() => {
      if (progressTimer !== undefined) window.clearInterval(progressTimer);
      updateArtifactUploadTask(id, { loaded: payload.file.size, progress: 100, status: "done", serverPhase: "done" });
      void Promise.resolve(onFinished()).catch(onFailed);
    })
    .catch((err) => {
      if (progressTimer !== undefined) window.clearInterval(progressTimer);
      updateArtifactUploadTask(id, { status: "error", error: readError(err) });
      onFailed(err);
    });
}

function applyServerUploadProgress(id: string, fileSize: number, progress: ServerUploadProgress) {
  if (progress.status === "error") {
    updateArtifactUploadTask(id, {
      status: "error",
      serverPhase: progress.phase,
      error: progress.error || "Anyshare 上传失败"
    });
    return;
  }
  updateArtifactUploadTask(id, {
    loaded: progress.loaded,
    total: progress.total || fileSize,
    progress: progress.percent,
    status: progress.status === "done" ? "done" : "processing",
    serverPhase: progress.phase
  });
}

export function App() {
  const [token, setToken] = useState(() => localStorage.getItem(tokenKey));
  const api = useMemo(() => new ApiClient(token), [token]);
  const [apps, setApps] = useState<DesktopApp[]>([]);
  const [stats, setStats] = useState<Record<string, AppStats>>({});
  const [selectedApp, setSelectedApp] = useState<DesktopApp | null>(null);
  const [view, setView] = useState<View>("home");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [showCreateApp, setShowCreateApp] = useState(false);

  useEffect(() => {
    api.setToken(token);
  }, [api, token]);

  useEffect(() => {
    if (token) {
      void refreshHome();
    }
  }, [token]);

  async function refreshHome() {
    setLoading(true);
    setError("");
    try {
      const items = await api.listApps();
      setApps(items);
      const nextStats: Record<string, AppStats> = {};
      await Promise.all(items.map(async (app) => {
        try {
          const releases = await api.listReleases(app.id);
          const latest = releases[0];
          if (!latest) {
            nextStats[app.id] = { latestVersion: "暂无版本", artifactCount: 0, platforms: [] };
            return;
          }
          const artifacts = await api.listArtifacts(latest.id);
          nextStats[app.id] = {
            latestVersion: latest.version,
            artifactCount: artifacts.length,
            platforms: Array.from(new Set(artifacts.map((artifact) => artifact.platform)))
          };
        } catch {
          nextStats[app.id] = { latestVersion: "读取失败", artifactCount: 0, platforms: [] };
        }
      }));
      setStats(nextStats);
    } catch (err) {
      setError(readError(err));
    } finally {
      setLoading(false);
    }
  }

  function handleLogin(nextToken: string) {
    localStorage.setItem(tokenKey, nextToken);
    setToken(nextToken);
  }

  function logout() {
    localStorage.removeItem(tokenKey);
    setToken(null);
    setApps([]);
    setSelectedApp(null);
    setView("home");
  }

  function openApp(app: DesktopApp) {
    setSelectedApp(app);
    setView("detail");
  }

  function openStats() {
    setSelectedApp(null);
    setView("stats");
  }

  function goHome() {
    setSelectedApp(null);
    setView("home");
    void refreshHome();
  }

  if (!token) {
    return <LoginScreen api={api} onLogin={handleLogin} />;
  }

  const filteredApps = apps.filter((app) => {
    const needle = query.trim().toLowerCase();
    if (!needle) return true;
    return `${app.name} ${app.slug} ${app.description}`.toLowerCase().includes(needle);
  });

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brandMark">
          <Boxes size={24} />
        </div>
        <button className={view === "home" || view === "detail" ? "sideButton active" : "sideButton"} title="应用" onClick={goHome}>
          <Layers3 size={21} />
        </button>
        <button className={view === "stats" ? "sideButton active" : "sideButton"} title="统计" onClick={openStats}>
          <BarChart3 size={21} />
        </button>
        <button className="sideButton bottom" title="退出登录" onClick={logout}>
          <LogOut size={21} />
        </button>
      </aside>

      <main className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">Distribution Console</p>
            <h1>{view === "home" ? "应用发布中心" : view === "stats" ? "统计面板" : selectedApp?.name}</h1>
          </div>
          <div className="topActions">
            {view === "detail" && (
              <button className="iconTextButton ghost" onClick={goHome}>
                <ArrowLeft size={18} />
                返回首页
              </button>
            )}
            {view === "stats" && (
              <button className="iconTextButton ghost" onClick={goHome}>
                <ArrowLeft size={18} />
                返回应用
              </button>
            )}
            <button className="iconTextButton" onClick={() => (view === "home" ? void refreshHome() : undefined)}>
              <RefreshCw size={18} />
              刷新
            </button>
            {view === "home" && (
              <button className="iconTextButton primary" onClick={() => setShowCreateApp(true)}>
                <Plus size={18} />
                创建应用
              </button>
            )}
          </div>
        </header>

        {error && <div className="notice error">{error}</div>}

        {view === "home" && (
          <HomeView
            apps={filteredApps}
            stats={stats}
            loading={loading}
            query={query}
            onQuery={setQuery}
            onCreate={() => setShowCreateApp(true)}
            onOpen={openApp}
          />
        )}

        {view === "detail" && selectedApp && (
          <AppDetail
            api={api}
            initialApp={selectedApp}
            onAppChanged={(app) => setSelectedApp(app)}
            onArchived={goHome}
          />
        )}

        {view === "stats" && <StatsView api={api} />}
      </main>

      {showCreateApp && (
        <CreateAppDialog
          api={api}
          onClose={() => setShowCreateApp(false)}
          onCreated={(app) => {
            setApps((items) => [app, ...items]);
            setShowCreateApp(false);
            openApp(app);
          }}
        />
      )}
    </div>
  );
}

function LoginScreen({ api, onLogin }: { api: ApiClient; onLogin: (token: string) => void }) {
  const [email, setEmail] = useState("admin@example.com");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const result = await api.login(email, password);
      onLogin(result.access_token);
    } catch (err) {
      setError(readError(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="loginPage">
      <form className="loginPanel" onSubmit={submit}>
        <div className="loginIcon">
          <KeyRound size={28} />
        </div>
        <h1>登录管理台</h1>
        <p>使用管理员账号进入应用发布中心。</p>
        <label>
          邮箱
          <input value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="email" />
        </label>
        <label>
          密码
          <input
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
          />
        </label>
        {error && <div className="notice error">{error}</div>}
        <button className="wideButton primary" disabled={submitting}>
          {submitting ? "登录中..." : "登录"}
        </button>
      </form>
    </div>
  );
}

function HomeView({
  apps,
  stats,
  loading,
  query,
  onQuery,
  onCreate,
  onOpen
}: {
  apps: DesktopApp[];
  stats: Record<string, AppStats>;
  loading: boolean;
  query: string;
  onQuery: (value: string) => void;
  onCreate: () => void;
  onOpen: (app: DesktopApp) => void;
}) {
  return (
    <section className="homeView">
      <div className="homeTools">
        <div className="searchBox">
          <Search size={18} />
          <input placeholder="搜索应用名称、slug 或描述" value={query} onChange={(event) => onQuery(event.target.value)} />
        </div>
      </div>

      {loading ? (
        <div className="emptyState">正在加载应用...</div>
      ) : apps.length === 0 ? (
        <button className="createFirstCard" onClick={onCreate}>
          <Plus size={26} />
          <span>创建第一个应用</span>
        </button>
      ) : (
        <div className="appGrid">
          {apps.map((app) => {
            const stat = stats[app.id] ?? { latestVersion: "读取中", artifactCount: 0, platforms: [] };
            return (
              <button className="appCard" key={app.id} onClick={() => onOpen(app)}>
                <div className="appIcon">{app.icon_url ? <img src={app.icon_url} alt="" /> : app.name.slice(0, 1)}</div>
                <div>
                  <h2>{app.name}</h2>
                  <p>{app.slug}</p>
                </div>
                <div className="cardMeta">
                  <span>最新：{stat.latestVersion}</span>
                  <span>安装包：{stat.artifactCount}</span>
                  <span>平台：{stat.platforms.length ? stat.platforms.join(", ") : "未覆盖"}</span>
                </div>
                <span className="channelPill">{app.default_channel}</span>
                <ChevronRight className="cardArrow" size={19} />
              </button>
            );
          })}
        </div>
      )}
    </section>
  );
}

function AppDetail({
  api,
  initialApp,
  onAppChanged,
  onArchived
}: {
  api: ApiClient;
  initialApp: DesktopApp;
  onAppChanged: (app: DesktopApp) => void;
  onArchived: () => void;
}) {
  const [app, setApp] = useState(initialApp);
  const [tab, setTab] = useState<DetailTab>("overview");
  const [releases, setReleases] = useState<Release[]>([]);
  const [artifacts, setArtifacts] = useState<Record<string, Artifact[]>>({});
  const [selectedRelease, setSelectedRelease] = useState("");
  const [serverConfig, setServerConfig] = useState<ServerConfig>({ anyshare_enabled: false });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setApp(initialApp);
    void refreshDetail();
  }, [initialApp.id]);

  async function refreshDetail() {
    setLoading(true);
    setError("");
    try {
      const [config, releaseItems] = await Promise.all([
        api.getConfig().catch(() => ({ anyshare_enabled: false })),
        api.listReleases(initialApp.id)
      ]);
      setServerConfig(config);
      setReleases(releaseItems);
      setSelectedRelease((current) => current || releaseItems[0]?.id || "");
      const nextArtifacts: Record<string, Artifact[]> = {};
      await Promise.all(releaseItems.map(async (release) => {
        nextArtifacts[release.id] = await api.listArtifacts(release.id);
      }));
      setArtifacts(nextArtifacts);
    } catch (err) {
      setError(readError(err));
    } finally {
      setLoading(false);
    }
  }

  async function updateApp(nextApp: DesktopApp) {
    setApp(nextApp);
    onAppChanged(nextApp);
  }

  const allArtifacts = Object.values(artifacts).flat();
  const latest = releases[0];

  return (
    <section className="detailView">
      {error && <div className="notice error">{error}</div>}
      <div className="tabBar">
        <TabButton active={tab === "overview"} icon={<Layers3 size={17} />} label="概览" onClick={() => setTab("overview")} />
        <TabButton active={tab === "releases"} icon={<PackagePlus size={17} />} label="版本" onClick={() => setTab("releases")} />
        <TabButton active={tab === "artifacts"} icon={<FileArchive size={17} />} label="安装包" onClick={() => setTab("artifacts")} />
        <TabButton active={tab === "metadata"} icon={<MonitorDown size={17} />} label="元数据" onClick={() => setTab("metadata")} />
        <TabButton active={tab === "stats"} icon={<BarChart3 size={17} />} label="统计" onClick={() => setTab("stats")} />
        <TabButton active={tab === "settings"} icon={<Settings size={17} />} label="设置" onClick={() => setTab("settings")} />
        <button className="iconTextButton ghost tabRefresh" onClick={() => void refreshDetail()}>
          <RefreshCw size={17} />
          刷新详情
        </button>
      </div>

      {loading && <div className="emptyState compact">正在同步应用数据...</div>}

      {tab === "overview" && (
        <OverviewTab app={app} latest={latest} releases={releases} artifacts={allArtifacts} />
      )}
      {tab === "releases" && (
        <ReleasesTab
          api={api}
          app={app}
          releases={releases}
          onError={setError}
          onChanged={refreshDetail}
        />
      )}
      {tab === "artifacts" && (
        <ArtifactsTab
          api={api}
          releases={releases}
          artifacts={artifacts}
          anyshareEnabled={serverConfig.anyshare_enabled}
          selectedRelease={selectedRelease}
          onRelease={setSelectedRelease}
          onError={setError}
          onChanged={refreshDetail}
        />
      )}
      {tab === "metadata" && <MetadataTab api={api} app={app} />}
      {tab === "stats" && <AppStatsTab api={api} app={app} />}
      {tab === "settings" && (
        <SettingsTab
          api={api}
          app={app}
          onSaved={updateApp}
          onArchived={onArchived}
          onError={setError}
        />
      )}
    </section>
  );
}

function OverviewTab({
  app,
  latest,
  releases,
  artifacts
}: {
  app: DesktopApp;
  latest?: Release;
  releases: Release[];
  artifacts: Artifact[];
}) {
  const platforms = Array.from(new Set(artifacts.map((artifact) => artifact.platform)));
  const recentArtifacts = artifacts.slice(0, 5);

  return (
    <div className="detailGrid">
      <section className="panel appSummary">
        <div className="summaryIcon">{app.icon_url ? <img src={app.icon_url} alt="" /> : app.name.slice(0, 1)}</div>
        <div>
          <h2>{app.name}</h2>
          <p>{app.description || "暂无应用描述"}</p>
        </div>
        <dl>
          <InfoItem label="Slug" value={app.slug} />
          <InfoItem label="默认渠道" value={app.default_channel} />
          <InfoItem label="接入地址" value={`/api/latest/${app.slug}/update.json`} />
        </dl>
      </section>
      <section className="panel">
        <PanelTitle icon={<MonitorDown size={19} />} title="发布状态" />
        <div className="metricGrid">
          <Metric label="最新版本" value={latest?.version ?? "暂无"} />
          <Metric label="版本数量" value={String(releases.length)} />
          <Metric label="安装包数量" value={String(artifacts.length)} />
          <Metric label="平台覆盖" value={platforms.length ? platforms.join(", ") : "暂无"} />
        </div>
      </section>
      <section className="panel metadataPanel">
        <PanelTitle icon={<FileArchive size={19} />} title="最近安装包" />
        {recentArtifacts.length === 0 ? (
          <div className="emptyState compact">暂无上传记录</div>
        ) : (
          <ArtifactTable artifacts={recentArtifacts} releases={releases} readonly />
        )}
      </section>
    </div>
  );
}

function ReleasesTab({
  api,
  app,
  releases,
  onError,
  onChanged
}: {
  api: ApiClient;
  app: DesktopApp;
  releases: Release[];
  onError: (error: string) => void;
  onChanged: () => Promise<void>;
}) {
  const [creating, setCreating] = useState(false);

  return (
    <div className="detailGrid">
      <section className="panel releasePanel">
        <PanelTitle icon={<PackagePlus size={19} />} title="创建版本" />
        <ReleaseForm
          creating={creating}
          onCreate={async (payload) => {
            setCreating(true);
            onError("");
            try {
              await api.createRelease(app.id, payload);
              await onChanged();
            } catch (err) {
              onError(readError(err));
            } finally {
              setCreating(false);
            }
          }}
        />
      </section>
      <section className="panel widePanel">
        <PanelTitle icon={<Layers3 size={19} />} title="版本管理" />
        {releases.length === 0 ? (
          <div className="emptyState compact">暂无版本</div>
        ) : (
          <div className="releaseEditorList">
            {releases.map((release) => (
              <ReleaseEditor key={release.id} api={api} release={release} onError={onError} onChanged={onChanged} />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

function ReleaseEditor({
  api,
  release,
  onError,
  onChanged
}: {
  api: ApiClient;
  release: Release;
  onError: (error: string) => void;
  onChanged: () => Promise<void>;
}) {
  const [version, setVersion] = useState(release.version);
  const [channel, setChannel] = useState(release.channel);
  const [changelog, setChangelog] = useState(release.changelog);
  const [forced, setForced] = useState(release.is_forced);
  const [staging, setStaging] = useState(release.staging_percent);
  const [saving, setSaving] = useState(false);

  async function save() {
    setSaving(true);
    onError("");
    try {
      await api.updateRelease(release.id, {
        version,
        channel,
        changelog,
        is_forced: forced,
        staging_percent: staging,
        published_at: release.published_at
      });
      await onChanged();
    } catch (err) {
      onError(readError(err));
    } finally {
      setSaving(false);
    }
  }

  async function archive() {
    if (!window.confirm(`归档版本 ${release.version}？`)) return;
    onError("");
    try {
      await api.archiveRelease(release.id);
      await onChanged();
    } catch (err) {
      onError(readError(err));
    }
  }

  return (
    <div className="releaseEditor">
      <div className="twoCols">
        <label>
          版本号
          <input value={version} onChange={(event) => setVersion(event.target.value)} />
        </label>
        <label>
          渠道
          <select value={channel} onChange={(event) => setChannel(event.target.value)}>
            {channels.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
      </div>
      <label>
        更新日志
        <textarea value={changelog} onChange={(event) => setChangelog(event.target.value)} />
      </label>
      <div className="twoCols">
        <label>
          灰度比例：{staging}%
          <input type="range" min="0" max="100" value={staging} onChange={(event) => setStaging(Number(event.target.value))} />
        </label>
        <label className="checkLine">
          <input type="checkbox" checked={forced} onChange={(event) => setForced(event.target.checked)} />
          强制更新
        </label>
      </div>
      <div className="rowActions">
        <button className="iconTextButton primary" onClick={() => void save()} disabled={saving}>
          <Save size={17} />
          保存版本
        </button>
        <button className="iconTextButton danger" onClick={() => void archive()}>
          <Archive size={17} />
          归档
        </button>
      </div>
    </div>
  );
}

function ArtifactsTab({
  api,
  releases,
  artifacts,
  anyshareEnabled,
  selectedRelease,
  onRelease,
  onError,
  onChanged
}: {
  api: ApiClient;
  releases: Release[];
  artifacts: Record<string, Artifact[]>;
  anyshareEnabled: boolean;
  selectedRelease: string;
  onRelease: (id: string) => void;
  onError: (error: string) => void;
  onChanged: () => Promise<void>;
}) {
  const [platformFilter, setPlatformFilter] = useState("");
  const [archFilter, setArchFilter] = useState("");
  const uploadTasks = useArtifactUploadTasks();
  const releaseIds = new Set(releases.map((release) => release.id));
  const visibleUploadTasks = uploadTasks.filter((task) => releaseIds.has(task.releaseId));
  const visibleArtifacts = (selectedRelease ? artifacts[selectedRelease] ?? [] : Object.values(artifacts).flat())
    .filter((artifact) => !platformFilter || artifact.platform === platformFilter)
    .filter((artifact) => !archFilter || artifact.arch === archFilter);

  return (
    <div className="detailGrid">
      <section className="panel uploadPanel">
        <PanelTitle icon={<FileUp size={19} />} title="上传安装包" />
        <UploadForm
          releases={releases}
          selectedRelease={selectedRelease}
          anyshareEnabled={anyshareEnabled}
          onRelease={onRelease}
          onUpload={(payload) => {
            if (!selectedRelease) {
              onError("请先创建或选择一个版本");
              return;
            }
            onError("");
            startArtifactUpload(
              api,
              selectedRelease,
              payload,
              onChanged,
              (err) => onError(readError(err))
            );
          }}
        />
        <UploadTaskList tasks={visibleUploadTasks} releases={releases} />
      </section>
      <section className="panel widePanel">
        <PanelTitle icon={<FileArchive size={19} />} title="安装包列表" />
        <div className="filterRow">
          <select value={selectedRelease} onChange={(event) => onRelease(event.target.value)}>
            <option value="">全部版本</option>
            {releases.map((release) => (
              <option value={release.id} key={release.id}>{release.version} / {release.channel}</option>
            ))}
          </select>
          <select value={platformFilter} onChange={(event) => setPlatformFilter(event.target.value)}>
            <option value="">全部平台</option>
            {platforms.map((item) => <option key={item}>{item}</option>)}
          </select>
          <select value={archFilter} onChange={(event) => setArchFilter(event.target.value)}>
            <option value="">全部架构</option>
            {arches.map((item) => <option key={item}>{item}</option>)}
          </select>
        </div>
        {visibleArtifacts.length === 0 ? (
          <div className="emptyState compact">暂无安装包</div>
        ) : (
          <ArtifactTable
            artifacts={visibleArtifacts}
            releases={releases}
            onArchive={async (artifact) => {
              if (!window.confirm(`归档安装包 ${artifact.file_name || artifact.id}？`)) return;
              onError("");
              try {
                await api.archiveArtifact(artifact.id);
                await onChanged();
              } catch (err) {
                onError(readError(err));
              }
            }}
            onSave={async (artifact, patch) => {
              onError("");
              try {
                await api.updateArtifact(artifact.id, patch);
                await onChanged();
              } catch (err) {
                onError(readError(err));
              }
            }}
            onReplace={async (artifact, file) => {
              onError("");
              try {
                await api.replaceArtifactFile(artifact.id, file);
                await onChanged();
              } catch (err) {
                onError(readError(err));
              }
            }}
          />
        )}
      </section>
    </div>
  );
}

function ArtifactTable({
  artifacts,
  releases,
  readonly,
  onArchive,
  onSave,
  onReplace
}: {
  artifacts: Artifact[];
  releases: Release[];
  readonly?: boolean;
  onArchive?: (artifact: Artifact) => Promise<void>;
  onSave?: (artifact: Artifact, patch: { platform: string; arch: string; file_type: string; file_name: string }) => Promise<void>;
  onReplace?: (artifact: Artifact, file: File) => Promise<void>;
}) {
  return (
    <div className="tableWrap">
      <table className="dataTable">
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
          {artifacts.map((artifact) => {
            const release = releases.find((item) => item.id === artifact.release_id);
            return (
              <ArtifactRow
                artifact={artifact}
                release={release}
                readonly={readonly}
                key={artifact.id}
                onArchive={onArchive}
                onSave={onSave}
                onReplace={onReplace}
              />
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function ArtifactRow({
  artifact,
  release,
  readonly,
  onArchive,
  onSave,
  onReplace
}: {
  artifact: Artifact;
  release?: Release;
  readonly?: boolean;
  onArchive?: (artifact: Artifact) => Promise<void>;
  onSave?: (artifact: Artifact, patch: { platform: string; arch: string; file_type: string; file_name: string }) => Promise<void>;
  onReplace?: (artifact: Artifact, file: File) => Promise<void>;
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
        <td><span className="sourcePill">{artifact.source_type === "anyshare" ? "Anyshare" : "默认"}</span></td>
        <td className="mono">{artifact.sha512.slice(0, 28)}...</td>
        {!readonly && (
          <td>
            <div className="tableActions">
              <button className="copyButton" title="复制下载链接" onClick={() => void navigator.clipboard.writeText(artifact.file_url)}>
                <Copy size={15} />
              </button>
              <button className="copyButton" title="编辑" onClick={() => setEditing(true)}>
                <Pencil size={15} />
              </button>
              <label className="copyButton uploadMini" title="重新上传">
                <UploadCloud size={15} />
                <input type="file" onChange={(event) => {
                  const file = event.target.files?.[0];
                  if (file) void onReplace?.(artifact, file);
                  event.currentTarget.value = "";
                }} />
              </label>
              <button className="copyButton dangerIcon" title="归档" onClick={() => void onArchive?.(artifact)}>
                <Archive size={15} />
              </button>
            </div>
          </td>
        )}
      </tr>
    );
  }

  return (
    <tr>
      <td><input value={fileName} onChange={(event) => setFileName(event.target.value)} /></td>
      <td>{release?.version ?? "-"}</td>
      <td>
        <div className="artifactEditGrid">
          <select value={platform} onChange={(event) => setPlatform(event.target.value)}>
            {platforms.map((item) => <option key={item}>{item}</option>)}
          </select>
          <select value={arch} onChange={(event) => setArch(event.target.value)}>
            {arches.map((item) => <option key={item}>{item}</option>)}
          </select>
          <select value={fileType} onChange={(event) => setFileType(event.target.value)}>
            {fileTypes.map((item) => <option key={item}>{item}</option>)}
          </select>
        </div>
      </td>
      <td>{formatBytes(artifact.file_size)}</td>
      <td><span className="sourcePill">{artifact.source_type === "anyshare" ? "Anyshare" : "默认"}</span></td>
      <td className="mono">{artifact.sha512.slice(0, 28)}...</td>
      <td>
        <div className="tableActions">
          <button className="copyButton" title="保存" onClick={() => {
            setEditing(false);
            void onSave?.(artifact, { file_name: fileName, platform, arch, file_type: fileType });
          }}>
            <Save size={15} />
          </button>
          <button className="copyButton" title="取消" onClick={() => setEditing(false)}>
            <ArrowLeft size={15} />
          </button>
        </div>
      </td>
    </tr>
  );
}

function MetadataTab({ api, app }: { api: ApiClient; app: DesktopApp }) {
  const [format, setFormat] = useState<MetaFormat>("json");
  const [channel, setChannel] = useState(app.default_channel);
  const [platform, setPlatform] = useState("");
  const [arch, setArch] = useState("");
  const [clientID, setClientID] = useState("");
  const [text, setText] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function preview(nextFormat = format) {
    setFormat(nextFormat);
    setLoading(true);
    setError("");
    const params = metadataParams(channel, platform, arch, clientID);
    try {
      if (nextFormat === "json") {
        const manifest: UpdateManifest = await api.getManifest(app.slug, params);
        setText(JSON.stringify(manifest, null, 2));
      } else {
        const suffix = nextFormat === "yml" ? "latest.yml" : "appcast.xml";
        setText(await api.getText(`/api/latest/${app.slug}/${suffix}?${params}`));
      }
    } catch (err) {
      setText("");
      setError(readError(err));
    } finally {
      setLoading(false);
    }
  }

  const url = metadataURL(app.slug, format, channel, platform, arch, clientID);

  return (
    <section className="panel metadataPanel">
      <PanelTitle icon={<MonitorDown size={19} />} title="元数据预览" />
      {error && <div className="notice error">{error}</div>}
      <div className="metaFilterGrid">
        <label>
          渠道
          <select value={channel} onChange={(event) => setChannel(event.target.value)}>
            {channels.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
        <label>
          平台
          <select value={platform} onChange={(event) => setPlatform(event.target.value)}>
            <option value="">全部平台</option>
            {platforms.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
        <label>
          架构
          <select value={arch} onChange={(event) => setArch(event.target.value)}>
            <option value="">全部架构</option>
            {arches.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
        <label>
          Client ID
          <input value={clientID} onChange={(event) => setClientID(event.target.value)} placeholder="灰度测试客户端标识" />
        </label>
      </div>
      <div className="metaControls">
        {(["json", "yml", "xml"] as MetaFormat[]).map((item) => (
          <button
            key={item}
            className={format === item ? "segmented active" : "segmented"}
            onClick={() => void preview(item)}
          >
            {item === "json" ? "update.json" : item === "yml" ? "latest.yml" : "appcast.xml"}
          </button>
        ))}
        <button className="copyButton" title="复制 URL" onClick={() => void navigator.clipboard.writeText(url)}>
          <Copy size={16} />
        </button>
      </div>
      <div className="urlLine">{url}</div>
      <pre className="metadataBox">{loading ? "正在读取..." : text || "选择格式后预览该应用的更新元数据。"}</pre>
    </section>
  );
}

function StatsView({ api }: { api: ApiClient }) {
  const [summary, setSummary] = useState<StatsSummary | null>(null);
  const [updates, setUpdates] = useState<UpdateRequestEvent[]>([]);
  const [downloads, setDownloads] = useState<DownloadEvent[]>([]);
  const [channel, setChannel] = useState("");
  const [platform, setPlatform] = useState("");
  const [arch, setArch] = useState("");
  const [clientID, setClientID] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    void refresh();
  }, []);

  async function refresh() {
    setLoading(true);
    setError("");
    const params = statsParams(channel, platform, arch, clientID, 20);
    try {
      const [nextSummary, nextUpdates, nextDownloads] = await Promise.all([
        api.getStatsSummary(params),
        api.listUpdateRequests(undefined, params),
        api.listDownloads(undefined, params)
      ]);
      setSummary(nextSummary);
      setUpdates(nextUpdates);
      setDownloads(nextDownloads);
    } catch (err) {
      setError(readError(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="statsPage">
      <StatsFilters
        channel={channel}
        platform={platform}
        arch={arch}
        clientID={clientID}
        onChannel={setChannel}
        onPlatform={setPlatform}
        onArch={setArch}
        onClientID={setClientID}
        onApply={refresh}
      />
      {error && <div className="notice error">{error}</div>}
      {loading && <div className="emptyState compact">正在加载统计数据...</div>}
      {summary && <StatsContent summary={summary} updates={updates} downloads={downloads} />}
    </section>
  );
}

function AppStatsTab({ api, app }: { api: ApiClient; app: DesktopApp }) {
  const [summary, setSummary] = useState<StatsSummary | null>(null);
  const [updates, setUpdates] = useState<UpdateRequestEvent[]>([]);
  const [downloads, setDownloads] = useState<DownloadEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    void refresh();
  }, [app.id]);

  async function refresh() {
    setLoading(true);
    setError("");
    const params = statsParams("", "", "", "", 20);
    try {
      const [nextSummary, nextUpdates, nextDownloads] = await Promise.all([
        api.getAppStatsSummary(app.id, params),
        api.listUpdateRequests(app.id, params),
        api.listDownloads(app.id, params)
      ]);
      setSummary(nextSummary);
      setUpdates(nextUpdates);
      setDownloads(nextDownloads);
    } catch (err) {
      setError(readError(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="statsPage">
      <div className="statsToolbar">
        <PanelTitle icon={<BarChart3 size={19} />} title="应用统计" />
        <button className="iconTextButton ghost" onClick={() => void refresh()}>
          <RefreshCw size={17} />
          刷新统计
        </button>
      </div>
      {error && <div className="notice error">{error}</div>}
      {loading && <div className="emptyState compact">正在加载统计数据...</div>}
      {summary && <StatsContent summary={summary} updates={updates} downloads={downloads} appScoped />}
    </section>
  );
}

function StatsFilters({
  channel,
  platform,
  arch,
  clientID,
  onChannel,
  onPlatform,
  onArch,
  onClientID,
  onApply
}: {
  channel: string;
  platform: string;
  arch: string;
  clientID: string;
  onChannel: (value: string) => void;
  onPlatform: (value: string) => void;
  onArch: (value: string) => void;
  onClientID: (value: string) => void;
  onApply: () => void;
}) {
  return (
    <section className="panel statsFilterPanel">
      <PanelTitle icon={<SlidersHorizontal size={19} />} title="筛选" />
      <div className="statsFilterGrid">
        <label>
          渠道
          <select value={channel} onChange={(event) => onChannel(event.target.value)}>
            <option value="">全部渠道</option>
            {channels.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
        <label>
          平台
          <select value={platform} onChange={(event) => onPlatform(event.target.value)}>
            <option value="">全部平台</option>
            {platforms.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
        <label>
          架构
          <select value={arch} onChange={(event) => onArch(event.target.value)}>
            <option value="">全部架构</option>
            {arches.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
        <label>
          Client ID
          <input value={clientID} onChange={(event) => onClientID(event.target.value)} placeholder="按客户端过滤" />
        </label>
        <button className="iconTextButton primary statsApply" onClick={() => void onApply()}>
          <RefreshCw size={17} />
          应用筛选
        </button>
      </div>
    </section>
  );
}

function StatsContent({
  summary,
  updates,
  downloads,
  appScoped
}: {
  summary: StatsSummary;
  updates: UpdateRequestEvent[];
  downloads: DownloadEvent[];
  appScoped?: boolean;
}) {
  return (
    <>
      <div className="statsCards">
        {appScoped && <Metric label="当前应用总下载" value={String(summary.total_downloads)} />}
        <Metric label="今日更新检查" value={String(summary.today_update_requests)} />
        <Metric label="今日下载" value={String(summary.today_downloads)} />
        <Metric label="7 日下载" value={String(summary.downloads_7d)} />
        <Metric label="活跃客户端" value={String(summary.active_clients_7d)} />
        <Metric label="命中灰度" value={String(summary.staging_hits_7d)} />
        <Metric label="强制版本下载" value={String(summary.forced_downloads_7d)} />
      </div>
      <div className="statsGrid">
        <StatsTrend title="更新检查趋势" points={summary.update_trend} />
        <StatsTrend title="下载趋势" points={summary.download_trend} />
        <StatsBreakdownPanel title="平台分布" items={summary.platform_breakdown} />
        <StatsBreakdownPanel title="架构分布" items={summary.arch_breakdown} />
        <StatsBreakdownPanel title="渠道分布" items={summary.channel_breakdown} />
        {appScoped && <StatsBreakdownPanel title="版本下载量" items={summary.version_breakdown} />}
      </div>
      <div className="statsTables">
        <section className="panel">
          <PanelTitle icon={<Server size={19} />} title="最近更新请求" />
          <UpdateRequestTable events={updates} />
        </section>
        <section className="panel">
          <PanelTitle icon={<Download size={19} />} title="最近下载记录" />
          <DownloadEventTable events={downloads} />
        </section>
      </div>
    </>
  );
}

function StatsTrend({ title, points }: { title: string; points: StatsPoint[] }) {
  const max = Math.max(1, ...points.map((point) => point.count));
  return (
    <section className="panel chartPanel">
      <PanelTitle icon={<BarChart3 size={19} />} title={title} />
      <div className="trendBars">
        {points.map((point) => (
          <div className="trendItem" key={point.date}>
            <div className="trendTrack">
              <span style={{ height: `${Math.max(6, (point.count / max) * 100)}%` }} />
            </div>
            <strong>{point.count}</strong>
            <small>{point.date.slice(5)}</small>
          </div>
        ))}
      </div>
    </section>
  );
}

function StatsBreakdownPanel({ title, items }: { title: string; items: StatsBreakdown[] }) {
  const max = Math.max(1, ...items.map((item) => item.count));
  return (
    <section className="panel breakdownPanel">
      <PanelTitle icon={<BarChart3 size={19} />} title={title} />
      {items.length === 0 ? (
        <div className="emptyState compact">暂无数据</div>
      ) : (
        <div className="breakdownList">
          {items.map((item) => (
            <div className="breakdownItem" key={item.key}>
              <div>
                <span>{item.key}</span>
                <strong>{item.count}</strong>
              </div>
              <div className="breakdownTrack"><span style={{ width: `${(item.count / max) * 100}%` }} /></div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function UpdateRequestTable({ events }: { events: UpdateRequestEvent[] }) {
  if (events.length === 0) {
    return <div className="emptyState compact">暂无更新请求记录</div>;
  }
  return (
    <div className="tableWrap">
      <table className="dataTable statsTable">
        <thead>
          <tr>
            <th>时间</th>
            <th>应用</th>
            <th>版本</th>
            <th>目标</th>
            <th>格式</th>
            <th>结果</th>
            <th>Client ID</th>
          </tr>
        </thead>
        <tbody>
          {events.map((event) => (
            <tr key={event.id}>
              <td>{formatDateTime(event.created_at)}</td>
              <td>{event.app_slug}</td>
              <td>{event.version || "-"}</td>
              <td>{event.channel || "-"}/{event.platform || "-"}/{event.arch || "-"}</td>
              <td>{event.format}</td>
              <td>{event.matched ? (event.staged_hit ? "灰度命中" : "命中") : "未命中"}</td>
              <td className="mono">{event.client_id || "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function DownloadEventTable({ events }: { events: DownloadEvent[] }) {
  if (events.length === 0) {
    return <div className="emptyState compact">暂无下载记录</div>;
  }
  return (
    <div className="tableWrap">
      <table className="dataTable statsTable">
        <thead>
          <tr>
            <th>时间</th>
            <th>文件</th>
            <th>版本</th>
            <th>目标</th>
            <th>Client ID</th>
          </tr>
        </thead>
        <tbody>
          {events.map((event) => (
            <tr key={event.id}>
              <td>{formatDateTime(event.created_at)}</td>
              <td>{event.file_name || event.artifact_id}</td>
              <td>{event.version}</td>
              <td>{event.platform}/{event.arch}/{event.file_type}</td>
              <td className="mono">{event.client_id || "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function SettingsTab({
  api,
  app,
  onSaved,
  onArchived,
  onError
}: {
  api: ApiClient;
  app: DesktopApp;
  onSaved: (app: DesktopApp) => void;
  onArchived: () => void;
  onError: (error: string) => void;
}) {
  const [name, setName] = useState(app.name);
  const [description, setDescription] = useState(app.description);
  const [iconURL, setIconURL] = useState(app.icon_url);
  const [channel, setChannel] = useState(app.default_channel);
  const [saving, setSaving] = useState(false);

  async function save(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    onError("");
    try {
      const nextApp = await api.updateApp(app.id, {
        name,
        description,
        icon_url: iconURL,
        default_channel: channel
      });
      onSaved(nextApp);
    } catch (err) {
      onError(readError(err));
    } finally {
      setSaving(false);
    }
  }

  async function archive() {
    if (!window.confirm(`归档应用 ${app.name}？归档后不会出现在 latest 和列表中。`)) return;
    onError("");
    try {
      await api.archiveApp(app.id);
      onArchived();
    } catch (err) {
      onError(readError(err));
    }
  }

  return (
    <section className="panel settingsPanel">
      <PanelTitle icon={<Settings size={19} />} title="应用设置" />
      <form className="stackForm" onSubmit={save}>
        <label>
          应用名称
          <input value={name} onChange={(event) => setName(event.target.value)} />
        </label>
        <label>
          描述
          <textarea value={description} onChange={(event) => setDescription(event.target.value)} />
        </label>
        <label>
          图标 URL
          <input value={iconURL} onChange={(event) => setIconURL(event.target.value)} />
        </label>
        <label>
          默认渠道
          <select value={channel} onChange={(event) => setChannel(event.target.value)}>
            {channels.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
        <div className="rowActions">
          <button className="iconTextButton primary" disabled={saving}>
            <Save size={17} />
            保存设置
          </button>
          <button type="button" className="iconTextButton danger" onClick={() => void archive()}>
            <Archive size={17} />
            归档应用
          </button>
        </div>
      </form>
    </section>
  );
}

function CreateAppDialog({
  api,
  onClose,
  onCreated
}: {
  api: ApiClient;
  onClose: () => void;
  onCreated: (app: DesktopApp) => void;
}) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [iconURL, setIconURL] = useState("");
  const [channel, setChannel] = useState("stable");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const app = await api.createApp({
        name,
        slug,
        description,
        icon_url: iconURL,
        default_channel: channel
      });
      onCreated(app);
    } catch (err) {
      setError(readError(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="dialogBackdrop">
      <form className="dialog" onSubmit={submit}>
        <h2>创建应用</h2>
        <label>
          应用名称
          <input value={name} onChange={(event) => setName(event.target.value)} placeholder="例如 Desktop Client" />
        </label>
        <label>
          Slug
          <input value={slug} onChange={(event) => setSlug(event.target.value)} placeholder="desktop-client" />
        </label>
        <label>
          描述
          <textarea value={description} onChange={(event) => setDescription(event.target.value)} />
        </label>
        <label>
          图标 URL
          <input value={iconURL} onChange={(event) => setIconURL(event.target.value)} />
        </label>
        <label>
          默认渠道
          <select value={channel} onChange={(event) => setChannel(event.target.value)}>
            {channels.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
        {error && <div className="notice error">{error}</div>}
        <div className="dialogActions">
          <button type="button" className="iconTextButton ghost" onClick={onClose}>取消</button>
          <button className="iconTextButton primary" disabled={submitting}>
            <Plus size={18} />
            创建
          </button>
        </div>
      </form>
    </div>
  );
}

function ReleaseForm({
  creating,
  onCreate
}: {
  creating: boolean;
  onCreate: (payload: {
    version: string;
    channel: string;
    changelog: string;
    is_forced: boolean;
    staging_percent: number;
    published_at?: string;
  }) => Promise<void>;
}) {
  const [version, setVersion] = useState("");
  const [channel, setChannel] = useState("stable");
  const [changelog, setChangelog] = useState("");
  const [forced, setForced] = useState(false);
  const [staging, setStaging] = useState(100);

  async function submit(event: FormEvent) {
    event.preventDefault();
    await onCreate({
      version,
      channel,
      changelog,
      is_forced: forced,
      staging_percent: staging
    });
    setVersion("");
    setChangelog("");
  }

  return (
    <form className="stackForm" onSubmit={submit}>
      <div className="twoCols">
        <label>
          版本号
          <input value={version} onChange={(event) => setVersion(event.target.value)} placeholder="1.0.0" />
        </label>
        <label>
          渠道
          <select value={channel} onChange={(event) => setChannel(event.target.value)}>
            {channels.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
      </div>
      <label>
        更新日志
        <textarea value={changelog} onChange={(event) => setChangelog(event.target.value)} />
      </label>
      <label>
        灰度比例：{staging}%
        <input type="range" min="0" max="100" value={staging} onChange={(event) => setStaging(Number(event.target.value))} />
      </label>
      <label className="checkLine">
        <input type="checkbox" checked={forced} onChange={(event) => setForced(event.target.checked)} />
        强制更新
      </label>
      <button className="wideButton primary" disabled={creating}>
        {creating ? "创建中..." : "创建版本"}
      </button>
    </form>
  );
}

function UploadTaskList({ tasks, releases }: { tasks: ArtifactUploadTask[]; releases: Release[] }) {
  if (tasks.length === 0) return null;
  const releaseLabels = new Map(releases.map((release) => [release.id, `${release.version} / ${release.channel}`]));

  return (
    <div className="uploadTaskList" aria-live="polite">
      {tasks.map((task) => (
        <div className={`uploadTask ${task.status}`} key={task.id}>
          <div className="uploadTaskHeader">
            <strong>{task.fileName}</strong>
            <span>{uploadTaskStatus(task)}</span>
          </div>
          <div className="uploadTaskMeta">
            <span>{releaseLabels.get(task.releaseId) ?? task.releaseId}</span>
            <span>{task.target}</span>
            <span>{formatBytes(task.fileSize)}</span>
          </div>
          <div
            className="uploadProgressTrack"
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={task.progress}
          >
            <span style={{ width: `${Math.max(task.progress, task.status === "uploading" || task.status === "processing" ? 2 : 0)}%` }} />
          </div>
          {task.error && <div className="uploadTaskError">{task.error}</div>}
        </div>
      ))}
    </div>
  );
}

function UploadForm({
  releases,
  selectedRelease,
  anyshareEnabled,
  onRelease,
  onUpload
}: {
  releases: Release[];
  selectedRelease: string;
  anyshareEnabled: boolean;
  onRelease: (id: string) => void;
  onUpload: (payload: ArtifactUploadPayload) => void | Promise<void>;
}) {
  const [platform, setPlatform] = useState("windows");
  const [arch, setArch] = useState("x64");
  const [fileType, setFileType] = useState("exe");
  const [sourceType, setSourceType] = useState<ArtifactSource>("managed");
  const [file, setFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!anyshareEnabled && sourceType === "anyshare") {
      setSourceType("managed");
    }
  }, [anyshareEnabled, sourceType]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!file) return;
    setSubmitting(true);
    try {
      await onUpload({ platform, arch, file_type: fileType, source_type: sourceType, file });
      setFile(null);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form className="stackForm" onSubmit={submit}>
      <label>
        版本
        <select value={selectedRelease} onChange={(event) => onRelease(event.target.value)}>
          <option value="">选择版本</option>
          {releases.map((release) => (
            <option value={release.id} key={release.id}>
              {release.version} / {release.channel}
            </option>
          ))}
        </select>
      </label>
      <label>
        存储来源
        <select value={sourceType} onChange={(event) => setSourceType(event.target.value as ArtifactSource)}>
          <option value="managed">默认存储</option>
          {anyshareEnabled && <option value="anyshare">Anyshare 实验</option>}
        </select>
      </label>
      <div className="threeCols">
        <label>
          平台
          <select value={platform} onChange={(event) => setPlatform(event.target.value)}>
            {platforms.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
        <label>
          架构
          <select value={arch} onChange={(event) => setArch(event.target.value)}>
            {arches.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
        <label>
          类型
          <select value={fileType} onChange={(event) => setFileType(event.target.value)}>
            {fileTypes.map((item) => <option key={item}>{item}</option>)}
          </select>
        </label>
      </div>
      <label className="fileDrop">
        <UploadCloud size={22} />
        <span>{file ? file.name : "选择安装包文件"}</span>
        <input type="file" onChange={(event) => setFile(event.target.files?.[0] ?? null)} />
      </label>
      <button className="wideButton primary" disabled={!file || !selectedRelease || submitting}>
        {submitting ? "上传中..." : "上传安装包"}
      </button>
    </form>
  );
}

function TabButton({ active, icon, label, onClick }: { active: boolean; icon: ReactNode; label: string; onClick: () => void }) {
  return (
    <button className={active ? "tabButton active" : "tabButton"} onClick={onClick}>
      {icon}
      {label}
    </button>
  );
}

function PanelTitle({ icon, title }: { icon: ReactNode; title: string }) {
  return (
    <div className="panelTitle">
      {icon}
      <h2>{title}</h2>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

function statsParams(channel: string, platform: string, arch: string, clientID: string, limit: number): URLSearchParams {
  const params = new URLSearchParams();
  if (channel) params.set("channel", channel);
  if (platform) params.set("platform", platform);
  if (arch) params.set("arch", arch);
  if (clientID) params.set("client_id", clientID);
  params.set("limit", String(limit));
  return params;
}

function metadataParams(channel: string, platform: string, arch: string, clientID: string): URLSearchParams {
  const params = new URLSearchParams();
  if (channel) params.set("channel", channel);
  if (platform) params.set("platform", platform);
  if (arch) params.set("arch", arch);
  if (clientID) params.set("client_id", clientID);
  return params;
}

function metadataURL(slug: string, format: MetaFormat, channel: string, platform: string, arch: string, clientID: string): string {
  const suffix = format === "json" ? "update.json" : format === "yml" ? "latest.yml" : "appcast.xml";
  const query = metadataParams(channel, platform, arch, clientID).toString();
  return `/api/latest/${slug}/${suffix}${query ? `?${query}` : ""}`;
}

function formatDateTime(value: string): string {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

function readError(err: unknown): string {
  if (err instanceof ApiError && err.status === 401) {
    return "登录已失效或没有权限";
  }
  if (err instanceof Error) {
    return err.message;
  }
  return "操作失败";
}

function uploadTaskStatus(task: ArtifactUploadTask): string {
  if (task.status === "processing" && task.serverPhase === "received") return "Anyshare 准备中";
  if (task.status === "processing" && task.serverPhase === "anyshare") return `Anyshare ${task.progress}%`;
  if (task.status === "processing" && task.serverPhase === "finalizing") return "Anyshare 收尾中";
  if (task.status === "done") return "完成";
  if (task.status === "error") return "失败";
  if (task.status === "processing") return "服务器处理中";
  if (task.progress > 0) return `${task.progress}%`;
  return task.loaded > 0 ? formatBytes(task.loaded) : "等待上传";
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}
