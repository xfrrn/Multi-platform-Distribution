import { useState, useCallback, useRef } from "react";
import type { Toast } from "../lib/types";

export function useToast() {
  const [toast, setToast] = useState<Toast | null>(null);
  const timer = useRef<number>(undefined);

  const show = useCallback((message: string, tone: Toast["tone"]) => {
    const id = Date.now();
    setToast({ id, message, tone });
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => {
      setToast((cur) => (cur?.id === id ? null : cur));
    }, 2200);
  }, []);

  return { toast, showToast: show };
}
