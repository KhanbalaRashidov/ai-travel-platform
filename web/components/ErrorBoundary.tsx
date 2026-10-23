"use client";

import React from "react";

interface State {
  hasError: boolean;
  message?: string;
}

// ErrorBoundary catches any render/runtime error in its subtree and shows a
// friendly fallback with a reload action, so a single bad value can never leave
// the user on a blank/crashed page.
export class ErrorBoundary extends React.Component<
  { children: React.ReactNode; fallbackTitle?: string },
  State
> {
  constructor(props: { children: React.ReactNode; fallbackTitle?: string }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(err: unknown): State {
    return { hasError: true, message: err instanceof Error ? err.message : String(err) };
  }

  componentDidCatch(err: unknown) {
    // Surface to the console for diagnosis; never rethrow.
    // eslint-disable-next-line no-console
    console.error("UI error boundary caught:", err);
  }

  reset = () => {
    this.setState({ hasError: false, message: undefined });
    if (typeof window !== "undefined") window.location.reload();
  };

  render() {
    if (!this.state.hasError) return this.props.children;
    return (
      <div className="mx-auto max-w-md px-4 py-20 text-center">
        <div className="card p-8">
          <div className="mx-auto grid h-12 w-12 place-items-center rounded-2xl bg-danger/10 text-danger">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12 9v4M12 17h.01M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z" />
            </svg>
          </div>
          <h2 className="mt-4 font-display text-lg font-bold">
            {this.props.fallbackTitle ?? "Something went wrong"}
          </h2>
          <p className="mt-2 text-sm text-muted">
            The page hit an unexpected error. Reloading usually fixes it.
          </p>
          <button
            onClick={this.reset}
            className="mt-5 inline-flex items-center justify-center rounded-full bg-primary px-5 py-2.5 text-sm font-semibold text-primary-fg transition-colors hover:bg-primary-strong"
          >
            Reload
          </button>
        </div>
      </div>
    );
  }
}
