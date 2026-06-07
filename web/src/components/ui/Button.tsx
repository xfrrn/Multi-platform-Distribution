import type { ReactNode, ButtonHTMLAttributes } from "react";
import "./Button.css";

type Props = {
  variant?: "primary" | "danger" | "ghost" | "default";
  size?: "sm" | "md";
  children: ReactNode;
} & ButtonHTMLAttributes<HTMLButtonElement>;

export function Button({ variant = "default", size = "md", className = "", children, ...rest }: Props) {
  return (
    <button className={`btn btn-${variant} btn-${size} ${className}`} {...rest}>
      {children}
    </button>
  );
}
