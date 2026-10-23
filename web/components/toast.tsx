"use client";

import { createContext, useCallback, useContext, useState } from "react";

type Kind = "success" | "error" | "info";
interface Toast {
  id: number;
  msg: string;
  kind: Kind;
}

interface Ctx {
  toast: (msg: string, kind?: Kind) => void;
}

const ToastContext = createContext<Ctx | null>(null);

// ToastProvider renders a small auto-dismissing toast stack (bottom-right) and
// exposes toast() via useToast.
export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<Toast[]>([]);

  const toast = useCallback((msg: string, kind: Kind = "info") => {
    const id = Date.now() + Math.random();
    setItems((x) => [...x, { id, msg, kind }]);
    setTimeout(() => setItems((x) => x.filter((t) => t.id !== id)), 3500);
  }, []);

  const tone = (k: Kind) =>
    k === "success"
      ? "border-success/30 bg-success/10 text-success"
      : k === "error"
        ? "border-danger/30 bg-danger/10 text-danger"
        : "border-border bg-surface text-text";

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col gap-2 print:hidden">
        {items.map((t) => (
          <div
            key={t.id}
            role="status"
            className={`pointer-events-auto animate-fade-up rounded-xl border px-4 py-2.5 text-sm font-medium shadow-lift backdrop-blur ${tone(t.kind)}`}
          >
            {t.msg}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

// useToast returns a safe no-op when used outside a provider.
export function useToast(): Ctx {
  return useContext(ToastContext) ?? { toast: () => {} };
}
