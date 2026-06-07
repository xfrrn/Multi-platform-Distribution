import type { ReactNode } from "react";
import "./Badge.css";

export function Badge({ tone = "default", children }: { tone?: "default" | "success" | "danger" | "warn"; children: ReactNode }) {
  return <span className={`badge badge-${tone}`}>{children}</span>;
}
