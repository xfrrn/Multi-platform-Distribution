import { Plus } from "lucide-react";
import type { DesktopApp } from "../../api";
import type { AppStats } from "../../lib/types";
import { SearchBox, AppCard, EmptyState } from "../ui";
import "./HomeView.css";

export function HomeView({ apps, stats, loading, query, onQuery, onCreate, onOpen }: {
  apps: DesktopApp[];
  stats: Record<string, AppStats>;
  loading: boolean;
  query: string;
  onQuery: (v: string) => void;
  onCreate: () => void;
  onOpen: (app: DesktopApp) => void;
}) {
  return (
    <section className="home-view">
      <div className="home-tools">
        <SearchBox value={query} onChange={onQuery} placeholder="搜索应用名称、slug 或描述" />
      </div>
      {loading ? (
        <EmptyState>正在加载应用...</EmptyState>
      ) : apps.length === 0 ? (
        <button className="home-create-first" onClick={onCreate}>
          <Plus size={26} />
          <span>创建第一个应用</span>
        </button>
      ) : (
        <div className="app-grid">
          {apps.map((app) => {
            const stat = stats[app.id] ?? { latestVersion: "读取中", artifactCount: 0, platforms: [] };
            return <AppCard key={app.id} app={app} stat={stat} onClick={() => onOpen(app)} />;
          })}
        </div>
      )}
    </section>
  );
}
