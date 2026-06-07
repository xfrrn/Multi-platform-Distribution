import type { ReactNode } from "react";
import "./Panel.css";

export function Panel({ className = "", children }: { className?: string; children: ReactNode }) {
  return <section className={`panel ${className}`}>{children}</section>;
}

export function PanelTitle({ icon, title }: { icon: ReactNode; title: string }) {
  return (
    <div className="panel-title">
      {icon}
      <h2>{title}</h2>
    </div>
  );
}
