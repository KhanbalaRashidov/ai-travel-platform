"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeSanitize from "rehype-sanitize";

// Markdown renders an assistant message as safe, nicely-formatted markdown
// (headings, bold/italic, lists, links, code) instead of raw "**" text (D10).
// rehype-sanitize strips any unsafe HTML; links open in a new tab. POI names
// the model wrote as markdown links stay clickable; plain place names are left
// as text (the roadmap already provides real Google Maps links).
export function Markdown({ children }: { children: string }) {
  return (
    <div className="prose-chat break-words text-sm leading-relaxed [overflow-wrap:anywhere]">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeSanitize]}
        components={{
          a: ({ href, children }) => (
            <a
              href={href}
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline underline-offset-2 hover:opacity-80 [overflow-wrap:anywhere]"
            >
              {children}
            </a>
          ),
          h1: ({ children }) => <h3 className="mt-2 mb-1 text-base font-bold first:mt-0">{children}</h3>,
          h2: ({ children }) => <h4 className="mt-2 mb-1 text-sm font-bold first:mt-0">{children}</h4>,
          h3: ({ children }) => <h4 className="mt-2 mb-1 text-sm font-semibold first:mt-0">{children}</h4>,
          ul: ({ children }) => <ul className="my-1 list-disc space-y-0.5 pl-4">{children}</ul>,
          ol: ({ children }) => <ol className="my-1 list-decimal space-y-0.5 pl-4">{children}</ol>,
          li: ({ children }) => <li className="marker:text-muted">{children}</li>,
          p: ({ children }) => <p className="my-1 first:mt-0 last:mb-0">{children}</p>,
          strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
          code: ({ children }) => (
            <code className="rounded bg-surface px-1 py-0.5 text-[0.8em] text-text [overflow-wrap:anywhere]">{children}</code>
          ),
          // Fenced code never forces the bubble wide: it wraps and, if still too
          // long, scrolls inside its own box.
          pre: ({ children }) => (
            <pre className="my-1 max-w-full overflow-x-auto whitespace-pre-wrap break-words rounded-lg bg-surface p-2 text-xs">
              {children}
            </pre>
          ),
          // GFM tables scroll inside their own container instead of overflowing.
          table: ({ children }) => (
            <div className="my-1 max-w-full overflow-x-auto">
              <table className="w-full text-xs">{children}</table>
            </div>
          ),
          blockquote: ({ children }) => (
            <blockquote className="my-1 border-l-2 border-border pl-2 text-muted">{children}</blockquote>
          ),
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}
