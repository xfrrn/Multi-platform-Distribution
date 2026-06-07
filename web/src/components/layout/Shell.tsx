import { Boxes, Layers3, BarChart3, LogOut } from "lucide-react";
import "./Shell.css";
import type { View } from "../../lib/types";

export function Shell({ view, onHome, onStats, onLogout, children }: {
  view: View;
  onHome: () => void;
  onStats: () => void;
  onLogout: () => void;
  children: React.ReactNode;
}) {
  const isActive = (v: View) => view === v || (v === "home" && view === "detail");
  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand-mark"><Boxes size={24} /></div>
        <button className={`side-btn ${isActive("home") ? "side-btn-active" : ""}`} title="应用" onClick={onHome}>
          <Layers3 size={21} />
        </button>
        <button className={`side-btn ${isActive("stats") ? "side-btn-active" : ""}`} title="统计" onClick={onStats}>
          <BarChart3 size={21} />
        </button>
        <button className="side-btn side-btn-bottom" title="退出登录" onClick={onLogout}>
          <LogOut size={21} />
        </button>
      </aside>
      <main className="workspace">{children}</main>
    </div>
  );
}
