import { ArrowLeft, RefreshCw, Plus } from "lucide-react";
import "./Topbar.css";
import { Button } from "../ui/Button";
import type { View } from "../../lib/types";

export function Topbar({ view, appName, onHome, onRefresh, onCreate }: {
  view: View;
  appName?: string;
  onHome: () => void;
  onRefresh: () => void;
  onCreate?: () => void;
}) {
  const titles: Record<View, string> = { home: "应用发布中心", detail: appName ?? "", stats: "统计面板" };
  return (
    <header className="topbar">
      <div>
        <p className="eyebrow">Distribution Console</p>
        <h1>{titles[view]}</h1>
      </div>
      <div className="topbar-actions">
        {(view === "detail" || view === "stats") && (
          <Button variant="ghost" onClick={onHome}><ArrowLeft size={18} />返回首页</Button>
        )}
        <Button variant="ghost" onClick={onRefresh}><RefreshCw size={18} />刷新</Button>
        {view === "home" && onCreate && (
          <Button variant="primary" onClick={onCreate}><Plus size={18} />创建应用</Button>
        )}
      </div>
    </header>
  );
}
