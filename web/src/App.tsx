import {
  ArrowLeft,
  Boxes,
  CheckCircle2,
  ChevronRight,
  Copy,
  FileUp,
  KeyRound,
  Layers3,
  LogOut,
  MonitorDown,
  PackagePlus,
  Plus,
  RefreshCw,
  Search,
  Server,
  UploadCloud
} from "lucide-react";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { ApiClient, ApiError, Artifact, DesktopApp, Release, UpdateManifest } from "./api";

type View = "home" | "detail";
type MetaFormat = "json" | "yml" | "xml";

const tokenKey = "mpd.adminToken";
const channels = ["stable", "beta", "internal"];
const platforms = ["windows", "macos", "linux"];
const arches = ["x64", "arm64", "universal"];
const fileTypes = ["exe", "dmg", "msi", "zip", "AppImage"];

export function App() {
  const [token, setToken] = useState(() => localStorage.getItem(tokenKey));
  const api = useMemo(() => new ApiClient(token), [token]);
  const [apps, setApps] = useState<DesktopApp[]>([]);
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
      void loadApps(api, setApps, setError, setLoading);
    }
  }, [api, token]);

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

  function goHome() {
    setSelectedApp(null);
    setView("home");
    void loadApps(api, setApps, setError, setLoading);
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
        <button className="sideButton active" title="应用" onClick={goHome}>
          <Layers3 size={21} />
        </button>
        <button className="sideButton" title="更新服务">
          <Server size={21} />
        </button>
        <button className="sideButton" title="下载">
          <MonitorDown size={21} />
        </button>
        <button className="sideButton bottom" title="退出登录" onClick={logout}>
          <LogOut size={21} />
        </button>
      </aside>

      <main className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">Distribution Console</p>
            <h1>{view === "home" ? "应用发布中心" : selectedApp?.name}</h1>
          </div>
          <div className="topActions">
            {view === "detail" && (
              <button className="iconTextButton ghost" onClick={goHome}>
                <ArrowLeft size={18} />
                返回首页
              </button>
            )}
            <button className="iconTextButton" onClick={() => void loadApps(api, setApps, setError, setLoading)}>
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
            loading={loading}
            query={query}
            onQuery={setQuery}
            onCreate={() => setShowCreateApp(true)}
            onOpen={openApp}
          />
        )}

        {view === "detail" && selectedApp && <AppDetail api={api} app={selectedApp} />}
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
  loading,
  query,
  onQuery,
  onCreate,
  onOpen
}: {
  apps: DesktopApp[];
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
          {apps.map((app) => (
            <button className="appCard" key={app.id} onClick={() => onOpen(app)}>
              <div className="appIcon">{app.icon_url ? <img src={app.icon_url} alt="" /> : app.name.slice(0, 1)}</div>
              <div>
                <h2>{app.name}</h2>
                <p>{app.slug}</p>
              </div>
              <span className="channelPill">{app.default_channel}</span>
              <ChevronRight className="cardArrow" size={19} />
            </button>
          ))}
        </div>
      )}
    </section>
  );
}

function AppDetail({ api, app }: { api: ApiClient; app: DesktopApp }) {
  const [releases, setReleases] = useState<Release[]>([]);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [selectedRelease, setSelectedRelease] = useState("");
  const [artifact, setArtifact] = useState<Artifact | null>(null);
  const [metaFormat, setMetaFormat] = useState<MetaFormat>("json");
  const [metaText, setMetaText] = useState("");
  const [metaLoading, setMetaLoading] = useState(false);

  useEffect(() => {
    void loadReleases();
  }, [app.id]);

  async function loadReleases() {
    setError("");
    try {
      const items = await api.listReleases(app.id);
      setReleases(items);
      setSelectedRelease((current) => current || items[0]?.id || "");
    } catch (err) {
      setError(readError(err));
    }
  }

  async function previewMetadata(format: MetaFormat) {
    setMetaFormat(format);
    setMetaLoading(true);
    setError("");
    const params = new URLSearchParams({ channel: app.default_channel });
    try {
      if (format === "json") {
        const manifest = await api.getManifest(app.slug, params);
        setMetaText(JSON.stringify(manifest, null, 2));
      } else {
        const suffix = format === "yml" ? "latest.yml" : "appcast.xml";
        setMetaText(await api.getText(`/api/latest/${app.slug}/${suffix}?${params}`));
      }
    } catch (err) {
      setMetaText("");
      setError(readError(err));
    } finally {
      setMetaLoading(false);
    }
  }

  const selected = releases.find((release) => release.id === selectedRelease);

  return (
    <section className="detailView">
      {error && <div className="notice error">{error}</div>}
      <div className="detailGrid">
        <section className="panel appSummary">
          <div className="summaryIcon">{app.icon_url ? <img src={app.icon_url} alt="" /> : app.name.slice(0, 1)}</div>
          <div>
            <h2>{app.name}</h2>
            <p>{app.description || "暂无应用描述"}</p>
          </div>
          <dl>
            <div>
              <dt>Slug</dt>
              <dd>{app.slug}</dd>
            </div>
            <div>
              <dt>默认渠道</dt>
              <dd>{app.default_channel}</dd>
            </div>
            <div>
              <dt>接入地址</dt>
              <dd>/api/latest/{app.slug}/update.json</dd>
            </div>
          </dl>
        </section>

        <section className="panel releasePanel">
          <PanelTitle icon={<PackagePlus size={19} />} title="创建版本" />
          <ReleaseForm
            creating={creating}
            onCreate={async (payload) => {
              setCreating(true);
              setError("");
              try {
                const release = await api.createRelease(app.id, payload);
                setReleases((items) => [release, ...items]);
                setSelectedRelease(release.id);
              } catch (err) {
                setError(readError(err));
              } finally {
                setCreating(false);
              }
            }}
          />
        </section>

        <section className="panel uploadPanel">
          <PanelTitle icon={<FileUp size={19} />} title="上传安装包" />
          <UploadForm
            releases={releases}
            selectedRelease={selectedRelease}
            onRelease={setSelectedRelease}
            onUpload={async (payload) => {
              if (!selectedRelease) {
                setError("请先创建或选择一个版本");
                return;
              }
              setError("");
              try {
                const result = await api.uploadArtifact(selectedRelease, payload);
                setArtifact(result);
              } catch (err) {
                setError(readError(err));
              }
            }}
          />
          {artifact && (
            <div className="artifactResult">
              <CheckCircle2 size={18} />
              <div>
                <strong>{artifact.platform}/{artifact.arch}</strong>
                <span>{formatBytes(artifact.file_size)} · {artifact.sha512.slice(0, 20)}...</span>
              </div>
            </div>
          )}
        </section>

        <section className="panel listPanel">
          <PanelTitle icon={<Layers3 size={19} />} title="版本列表" />
          {releases.length === 0 ? (
            <div className="emptyState compact">暂无版本</div>
          ) : (
            <div className="releaseList">
              {releases.map((release) => (
                <button
                  className={release.id === selectedRelease ? "releaseItem selected" : "releaseItem"}
                  key={release.id}
                  onClick={() => setSelectedRelease(release.id)}
                >
                  <strong>{release.version}</strong>
                  <span>{release.channel} · 灰度 {release.staging_percent}%</span>
                </button>
              ))}
            </div>
          )}
        </section>

        <section className="panel metadataPanel">
          <PanelTitle icon={<MonitorDown size={19} />} title="元数据预览" />
          <div className="metaControls">
            {(["json", "yml", "xml"] as MetaFormat[]).map((format) => (
              <button
                key={format}
                className={metaFormat === format ? "segmented active" : "segmented"}
                onClick={() => void previewMetadata(format)}
              >
                {format === "json" ? "update.json" : format === "yml" ? "latest.yml" : "appcast.xml"}
              </button>
            ))}
            <button
              className="copyButton"
              onClick={() => void navigator.clipboard.writeText(metadataURL(app.slug, metaFormat, app.default_channel))}
            >
              <Copy size={16} />
            </button>
          </div>
          <pre className="metadataBox">
            {metaLoading
              ? "正在读取..."
              : metaText || (selected ? "选择格式后预览该应用的更新元数据。" : "创建版本并上传安装包后可预览元数据。")}
          </pre>
        </section>
      </div>
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

function UploadForm({
  releases,
  selectedRelease,
  onRelease,
  onUpload
}: {
  releases: Release[];
  selectedRelease: string;
  onRelease: (id: string) => void;
  onUpload: (payload: { platform: string; arch: string; file_type: string; file: File }) => Promise<void>;
}) {
  const [platform, setPlatform] = useState("windows");
  const [arch, setArch] = useState("x64");
  const [fileType, setFileType] = useState("exe");
  const [file, setFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!file) return;
    setSubmitting(true);
    try {
      await onUpload({ platform, arch, file_type: fileType, file });
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

function PanelTitle({ icon, title }: { icon: React.ReactNode; title: string }) {
  return (
    <div className="panelTitle">
      {icon}
      <h2>{title}</h2>
    </div>
  );
}

async function loadApps(
  api: ApiClient,
  setApps: (apps: DesktopApp[]) => void,
  setError: (error: string) => void,
  setLoading: (loading: boolean) => void
) {
  setLoading(true);
  setError("");
  try {
    setApps(await api.listApps());
  } catch (err) {
    setError(readError(err));
  } finally {
    setLoading(false);
  }
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

function metadataURL(slug: string, format: MetaFormat, channel: string): string {
  const suffix = format === "json" ? "update.json" : format === "yml" ? "latest.yml" : "appcast.xml";
  return `/api/latest/${slug}/${suffix}?channel=${channel}`;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}
