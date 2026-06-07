import "./Modal.css";
import type { FormEvent, ReactNode } from "react";

export function Modal({ title, children, onClose, wide }: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  wide?: boolean;
}) {
  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className={`modal ${wide ? "modal-wide" : ""}`} onClick={(e) => e.stopPropagation()}>
        <div className="modal-bar">
          <h2>{title}</h2>
          <button className="modal-close" onClick={onClose} aria-label="关闭">&times;</button>
        </div>
        {children}
      </div>
    </div>
  );
}

export function ModalActions({ children }: { children: ReactNode }) {
  return <div className="modal-actions">{children}</div>;
}
