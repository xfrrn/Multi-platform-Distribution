import type { ReactNode, InputHTMLAttributes } from "react";
import "./Field.css";

type Props = { label: string; children: ReactNode } & InputHTMLAttributes<HTMLInputElement>;

export function Field({ label, children, ...rest }: Props) {
  return (
    <label className="field">
      {label}
      {children}
    </label>
  );
}
