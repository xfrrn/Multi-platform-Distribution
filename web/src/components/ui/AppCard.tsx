import { ChevronRight } from "lucide-react";
import "./AppCard.css";
import type { DesktopApp } from "../../api";
import type { AppStats } from "../../lib/types";
import { Badge } from "./Badge";

export function AppCard({ app, stat, onClick }: {
  app: DesktopApp;
  stat: AppStats;
  onClick: () => void;
}) {
  return (
    <button className="app-card" onClick={onClick}>
      <div className="app-card-icon">{app.icon_url ? <img src={app.icon_url} alt="" /> : app.name.slice(0, 1)}</div>
      <div>
        <h2>{app.name}</h2>
        <p>{app.slug}</p>
      </div>
      <div className="app-card-meta">
        <span>最新：{stat.latestVersion}</span>
        <span>安装包：{stat.artifactCount}</span>
        <span>平台：{stat.platforms.length ? stat.platforms.join(", ") : "未覆盖"}</span>
      </div>
      <Badge tone="success">{app.default_channel}</Badge>
      <ChevronRight className="app-card-arrow" size={19} />
    </button>
  );
}
