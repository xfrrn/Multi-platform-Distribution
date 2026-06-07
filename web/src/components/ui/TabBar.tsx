import type { ReactNode } from "react";
import "./TabBar.css";

export function TabButton({ active, icon, label, onClick }: {
  active: boolean; icon: ReactNode; label: string; onClick: () => void;
}) {
  return (
    <button className={`tab-btn ${active ? "tab-btn-active" : ""}`} onClick={onClick}>
      {icon}
      {label}
    </button>
  );
}

export function TabBar({ children }: { children: ReactNode }) {
  return <div className="tab-bar">{children}</div>;
}
