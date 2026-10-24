"use client";

import type { ReactNode, SelectHTMLAttributes, InputHTMLAttributes } from "react";

function cx(...parts: Array<string | false | undefined>) {
  return parts.filter(Boolean).join(" ");
}

export function Button({
  children,
  variant = "primary",
  className,
  ...rest
}: {
  children: ReactNode;
  variant?: "primary" | "ghost";
} & React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const styles =
    variant === "primary"
      ? "bg-primary text-primary-fg shadow-soft hover:bg-primary-strong active:translate-y-px"
      : "border border-border bg-surface text-text hover:border-primary/40 hover:text-primary";
  return (
    <button
      className={cx(
        "inline-flex items-center justify-center gap-2 rounded-full px-5 py-2.5 text-sm font-semibold transition-all disabled:cursor-not-allowed disabled:opacity-50",
        styles,
        className
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

export function Chip({
  active,
  children,
  ...rest
}: {
  active: boolean;
  children: ReactNode;
} & React.ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      type="button"
      aria-pressed={active}
      className={cx(
        "rounded-full border px-3.5 py-1.5 text-sm font-medium transition-all",
        active
          ? "border-primary bg-primary/10 text-primary shadow-soft"
          : "border-border bg-surface text-muted hover:border-primary/40 hover:text-text"
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

export function Badge({
  children,
  tone = "neutral",
}: {
  children: ReactNode;
  tone?: "neutral" | "primary" | "accent";
}) {
  const tones = {
    neutral: "bg-surface-2 text-muted border-border",
    primary: "bg-primary/10 text-primary border-primary/20",
    accent: "bg-accent/10 text-accent border-accent/20",
  };
  return (
    <span
      className={cx(
        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold",
        tones[tone]
      )}
    >
      {children}
    </span>
  );
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    // min-w-0 so the field can shrink inside a grid/flex row instead of forcing
    // the track wider than its share (which made neighbors overlap).
    <label className="block min-w-0">
      <span className="mb-1.5 block text-xs font-semibold uppercase tracking-wide text-muted">
        {label}
      </span>
      {children}
    </label>
  );
}

// min-w-0 lets the control shrink to its track — crucial for native date inputs,
// whose intrinsic min-width otherwise overflows a narrow grid cell.
const control =
  "w-full min-w-0 min-h-[2.75rem] rounded-xl border border-border bg-surface-2 px-3 py-2.5 text-sm text-text transition-colors hover:border-primary/40 focus:border-primary focus:bg-surface";

export function Select(props: SelectHTMLAttributes<HTMLSelectElement>) {
  const { className, ...rest } = props;
  return <select className={cx(control, "appearance-none pr-8", className)} {...rest} />;
}

export function NumberInput(props: InputHTMLAttributes<HTMLInputElement>) {
  const { className, ...rest } = props;
  return <input className={cx(control, className)} {...rest} />;
}

export function Card({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cx("card p-5", className)}>{children}</div>;
}
