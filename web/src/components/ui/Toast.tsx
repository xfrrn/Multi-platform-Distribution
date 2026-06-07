import "./Toast.css";
import type { Toast as ToastType } from "../../lib/types";

export function Toast({ toast }: { toast: ToastType }) {
  return (
    <div className={`toast toast-${toast.tone}`} role="status" aria-live="polite">
      {toast.message}
    </div>
  );
}
