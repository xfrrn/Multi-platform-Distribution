import { MonitorDown, FileArchive } from "lucide-react";
import type { DesktopApp, Release, Artifact } from "../../api";
import { absoluteMetadataURL } from "../../lib/utils";
import { Panel, PanelTitle, Metric } from "../ui";
import { ArtifactTable } from "./ArtifactTable";
import "./OverviewTab.css";

export function OverviewTab({ app, publicBaseURL, releases, artifacts }: {
  app: DesktopApp;
  publicBaseURL: string;
  releases: Release[];
  artifacts: Artifact[];
}) {
  const latest = releases[0];
  const platforms = [...new Set(artifacts.map((a) => a.platform))];
  const recent = artifacts.slice(0, 5);
  const url = absoluteMetadataURL(publicBaseURL, app.slug, "update.json");

  return (
    <div className="detail-grid">
      <Panel className="overview-summary">
        <div className="summary-icon">{app.icon_url ? <img src={app.icon_url} alt="" /> : app.name.slice(0, 1)}</div>
        <div>
          <h2>{app.name}</h2>
          <p>{app.description || "暂无应用描述"}</p>
        </div>
        <dl>
          <InfoItem label="Slug" value={app.slug} />
          <InfoItem label="默认渠道" value={app.default_channel} />
          <InfoItem label="接入地址" value={url} />
        </dl>
      </Panel>

      <Panel>
        <PanelTitle icon={<MonitorDown size={19} />} title="发布状态" />
        <div className="metric-grid">
          <Metric label="最新版本" value={latest?.version ?? "暂无"} />
          <Metric label="版本数量" value={String(releases.length)} />
          <Metric label="安装包数量" value={String(artifacts.length)} />
          <Metric label="平台覆盖" value={platforms.length ? platforms.join(", ") : "暂无"} />
        </div>
      </Panel>

      <Panel className="overview-artifacts">
        <PanelTitle icon={<FileArchive size={19} />} title="最近安装包" />
        {recent.length === 0 ? (
          <div className="empty-state compact">暂无上传记录</div>
        ) : (
          <ArtifactTable artifacts={recent} releases={releases} readonly />
        )}
      </Panel>
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
