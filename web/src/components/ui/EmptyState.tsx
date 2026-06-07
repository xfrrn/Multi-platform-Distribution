import "./EmptyState.css";

export function EmptyState({ compact, children }: { compact?: boolean; children: React.ReactNode }) {
  return <div className={`empty-state ${compact ? "empty-state-compact" : ""}`}>{children}</div>;
}
