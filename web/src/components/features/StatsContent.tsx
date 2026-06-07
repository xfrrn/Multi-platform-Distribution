import { BarChart3, Server, Download } from "lucide-react";
import type { StatsSummary, StatsPoint, StatsBreakdown, UpdateRequestEvent, DownloadEvent } from "../../api";
import { formatDateTime } from "../../lib/utils";
import { Metric, Panel, PanelTitle, EmptyState } from "../ui";

export function StatsContent({ summary, updates, downloads, appScoped }: {
  summary: StatsSummary;
  updates: UpdateRequestEvent[];
  downloads: DownloadEvent[];
  appScoped?: boolean;
}) {
  return (
    <>
      <div className="stats-cards">
        {appScoped && <Metric label="当前应用总下载" value={String(summary.total_downloads)} />}
        <Metric label="今日更新检查" value={String(summary.today_update_requests)} />
        <Metric label="今日下载" value={String(summary.today_downloads)} />
        <Metric label="7 日下载" value={String(summary.downloads_7d)} />
        <Metric label="活跃客户端" value={String(summary.active_clients_7d)} />
        <Metric label="命中灰度" value={String(summary.staging_hits_7d)} />
        <Metric label="强制版本下载" value={String(summary.forced_downloads_7d)} />
      </div>
      <div className="stats-grid">
        <TrendChart title="更新检查趋势" points={summary.update_trend} />
        <TrendChart title="下载趋势" points={summary.download_trend} />
        <BreakdownChart title="平台分布" items={summary.platform_breakdown} />
        <BreakdownChart title="架构分布" items={summary.arch_breakdown} />
        <BreakdownChart title="渠道分布" items={summary.channel_breakdown} />
        {appScoped && <BreakdownChart title="版本下载量" items={summary.version_breakdown} />}
      </div>
      <div className="stats-tables">
        <Panel>
          <PanelTitle icon={<Server size={19} />} title="最近更新请求" />
          <UpdateTable events={updates} />
        </Panel>
        <Panel>
          <PanelTitle icon={<Download size={19} />} title="最近下载记录" />
          <DownloadTable events={downloads} />
        </Panel>
      </div>
    </>
  );
}

function TrendChart({ title, points }: { title: string; points: StatsPoint[] }) {
  const max = Math.max(1, ...points.map((p) => p.count));
  return (
    <Panel className="chart-panel">
      <PanelTitle icon={<BarChart3 size={19} />} title={title} />
      <div className="trend-bars">
        {points.map((p) => (
          <div className="trend-item" key={p.date}>
            <div className="trend-track"><span style={{ height: `${Math.max(6, (p.count / max) * 100)}%` }} /></div>
            <strong>{p.count}</strong>
            <small>{p.date.slice(5)}</small>
          </div>
        ))}
      </div>
    </Panel>
  );
}

function BreakdownChart({ title, items }: { title: string; items: StatsBreakdown[] }) {
  const max = Math.max(1, ...items.map((i) => i.count));
  return (
    <Panel className="breakdown-panel">
      <PanelTitle icon={<BarChart3 size={19} />} title={title} />
      {items.length === 0 ? <EmptyState compact>暂无数据</EmptyState> : (
        <div className="breakdown-list">
          {items.map((i) => (
            <div className="breakdown-item" key={i.key}>
              <div><span>{i.key}</span><strong>{i.count}</strong></div>
              <div className="breakdown-track"><span style={{ width: `${(i.count / max) * 100}%` }} /></div>
            </div>
          ))}
        </div>
      )}
    </Panel>
  );
}

function UpdateTable({ events }: { events: UpdateRequestEvent[] }) {
  if (events.length === 0) return <EmptyState compact>暂无更新请求记录</EmptyState>;
  return (
    <div className="stats-table-wrap">
      <table className="data-table stats-table">
        <thead><tr><th>时间</th><th>应用</th><th>版本</th><th>目标</th><th>格式</th><th>结果</th><th>Client ID</th></tr></thead>
        <tbody>
          {events.map((e) => (
            <tr key={e.id}>
              <td>{formatDateTime(e.created_at)}</td>
              <td>{e.app_slug}</td>
              <td>{e.version || "-"}</td>
              <td>{e.channel || "-"}/{e.platform || "-"}/{e.arch || "-"}</td>
              <td>{e.format}</td>
              <td>{e.matched ? (e.staged_hit ? "灰度命中" : "命中") : "未命中"}</td>
              <td className="mono">{e.client_id || "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function DownloadTable({ events }: { events: DownloadEvent[] }) {
  if (events.length === 0) return <EmptyState compact>暂无下载记录</EmptyState>;
  return (
    <div className="stats-table-wrap">
      <table className="data-table stats-table">
        <thead><tr><th>时间</th><th>文件</th><th>版本</th><th>目标</th><th>Client ID</th></tr></thead>
        <tbody>
          {events.map((e) => (
            <tr key={e.id}>
              <td>{formatDateTime(e.created_at)}</td>
              <td>{e.file_name || e.artifact_id}</td>
              <td>{e.version}</td>
              <td>{e.platform}/{e.arch}/{e.file_type}</td>
              <td className="mono">{e.client_id || "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
