import type { ReactNode } from "react";
import "./EmptyState.css";

export function EmptyState({ compact, icon, title, children, action }: {
  compact?: boolean;
  icon?: ReactNode;
  title?: string;
  children?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className={`empty-state ${compact ? "empty-state-compact" : ""}`}>
      {icon && <div className="empty-state-icon">{icon}</div>}
      {title && <p className="empty-state-title">{title}</p>}
      {children && <div className="empty-state-body">{children}</div>}
      {action && <div className="empty-state-action">{action}</div>}
    </div>
  );
}
