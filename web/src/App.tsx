import { type JSX, useEffect, useMemo, useState } from "react";
import { ApiClient } from "./api";
import type { DesktopApp, AppStats } from "./lib/types";
import { TOKEN_KEY } from "./lib/constants";
import { readError } from "./lib/utils";
import { useToast } from "./hooks/useToast";
import { Shell, Topbar } from "./components/layout";
import { Toast } from "./components/ui";
import { LoginScreen, HomeView, CreateAppDialog, AppDetail, StatsView } from "./components/features";

type View = "home" | "detail" | "stats";

const api = new ApiClient(null);

export function App(): JSX.Element {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem(TOKEN_KEY));
  const [apps, setApps] = useState<DesktopApp[]>([]);
  const [stats, setStats] = useState<Record<string, AppStats>>({});
  const [selectedApp, setSelectedApp] = useState<DesktopApp | null>(null);
  const [view, setView] = useState<View>("home");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [showCreateApp, setShowCreateApp] = useState(false);
  const { toast, showToast } = useToast();

  useEffect(() => { api.setToken(token); }, [token]);
  useEffect(() => { if (token) { void refreshHome(); } }, [token]);

  async function refreshHome() {
    setLoading(true); setError("");
    try {
      const items = await api.listApps();
      setApps(items);
      const map: Record<string, AppStats> = {};
      await Promise.all(items.map(async (app) => {
        try {
          const releases = await api.listReleases(app.id);
          const latest = releases[0];
          if (!latest) { map[app.id] = { latestVersion: "暂无版本", artifactCount: 0, platforms: [] }; return; }
          const artifacts = await api.listArtifacts(latest.id);
          map[app.id] = { latestVersion: latest.version, artifactCount: artifacts.length, platforms: [...new Set(artifacts.map((a) => a.platform))] };
        } catch { map[app.id] = { latestVersion: "读取失败", artifactCount: 0, platforms: [] }; }
      }));
      setStats(map);
    } catch (err) { setError(readError(err)); }
    finally { setLoading(false); }
  }

  function login(t: string) { localStorage.setItem(TOKEN_KEY, t); setToken(t); }
  function logout() { localStorage.removeItem(TOKEN_KEY); setToken(null); setApps([]); setSelectedApp(null); setView("home"); }

  async function copyText(value: string) {
    try { await navigator.clipboard.writeText(value); showToast("已复制", "success"); }
    catch { showToast("复制失败", "error"); }
  }

  if (!token) return <LoginScreen api={api} onLogin={login} />;

  const filteredApps = apps.filter((a) => {
    const q = query.trim().toLowerCase();
    if (!q) return true;
    return `${a.name} ${a.slug} ${a.description}`.toLowerCase().includes(q);
  });

  return (
    <Shell view={view} onHome={() => { setSelectedApp(null); setView("home"); void refreshHome(); }} onStats={() => { setSelectedApp(null); setView("stats"); }} onLogout={logout}>
      <Topbar view={view} appName={selectedApp?.name} onHome={() => { setSelectedApp(null); setView("home"); void refreshHome(); }} onRefresh={() => { if (view === "home") { void refreshHome(); } }} onCreate={() => setShowCreateApp(true)} />
      {error && <div className="notice error">{error}</div>}
      {view === "home" && <HomeView apps={filteredApps} stats={stats} loading={loading} query={query} onQuery={setQuery} onCreate={() => setShowCreateApp(true)} onOpen={(app) => { setSelectedApp(app); setView("detail"); }} />}
      {view === "detail" && selectedApp && <AppDetail api={api} initialApp={selectedApp} onAppChanged={setSelectedApp} onArchived={() => { setSelectedApp(null); setView("home"); }} onCopy={copyText} />}
      {view === "stats" && <StatsView api={api} />}
      {showCreateApp && <CreateAppDialog api={api} onClose={() => setShowCreateApp(false)} onCreated={(app) => { setApps((prev) => [app, ...prev]); setShowCreateApp(false); setSelectedApp(app); setView("detail"); }} />}
      {toast && <Toast toast={toast} />}
    </Shell>
  );
}
