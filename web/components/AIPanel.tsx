"use client";

import { useState } from "react";
import {
  AIResult,
  ApiError,
  chat as chatApi,
  itinerary as itineraryApi,
  ItineraryBrief,
} from "../lib/api";
import { useI18n } from "../lib/i18n";
import { Button, Card } from "./ui";
import { SparkIcon } from "./icons";

interface ChatTurn {
  role: "user" | "assistant";
  text: string;
}

// AIPanel wires the itinerary writer and chat to the backend. It handles a 503
// (AI not configured) by switching the whole panel to a friendly "AI is off"
// state, and keeps per-feature loading/error/empty states.
export function AIPanel({ brief }: { brief: ItineraryBrief | null }) {
  const { t } = useI18n();
  const [aiOff, setAiOff] = useState(false);

  const [itin, setItin] = useState<AIResult | null>(null);
  const [itinLoading, setItinLoading] = useState(false);
  const [itinErr, setItinErr] = useState<string | null>(null);

  const [messages, setMessages] = useState<ChatTurn[]>([]);
  const [input, setInput] = useState("");
  const [chatLoading, setChatLoading] = useState(false);
  const [chatErr, setChatErr] = useState<string | null>(null);

  // Returns true when the error was a 503 and the panel switched to AI-off.
  function handledAsOff(e: unknown): boolean {
    if (e instanceof ApiError && e.status === 503) {
      setAiOff(true);
      return true;
    }
    return false;
  }

  async function generate() {
    if (!brief || itinLoading) return;
    setItinLoading(true);
    setItinErr(null);
    try {
      setItin(await itineraryApi(brief));
    } catch (e) {
      if (!handledAsOff(e)) setItinErr(t("ai.error"));
    } finally {
      setItinLoading(false);
    }
  }

  async function send() {
    const msg = input.trim();
    if (!msg || chatLoading) return;
    setMessages((m) => [...m, { role: "user", text: msg }]);
    setInput("");
    setChatLoading(true);
    setChatErr(null);
    try {
      const r = await chatApi(msg);
      setMessages((m) => [...m, { role: "assistant", text: r.text }]);
    } catch (e) {
      if (!handledAsOff(e)) setChatErr(t("ai.error"));
    } finally {
      setChatLoading(false);
    }
  }

  const Spinner = () => (
    <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent opacity-70" />
  );

  if (aiOff) {
    return (
      <Card className="animate-fade-in flex flex-col items-center py-10 text-center">
        <span className="grid h-12 w-12 place-items-center rounded-2xl bg-muted/15 text-muted">
          <SparkIcon width={22} height={22} />
        </span>
        <h2 className="mt-4 font-display text-lg font-bold">{t("ai.off.title")}</h2>
        <p className="mt-1.5 max-w-sm text-sm text-muted">{t("ai.off.desc")}</p>
        <Button variant="ghost" className="mt-5" onClick={() => setAiOff(false)}>
          {t("ai.retry")}
        </Button>
      </Card>
    );
  }

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {/* Itinerary writer */}
      <Card className="animate-fade-up">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <SparkIcon width={18} height={18} className="text-primary" />
            <h2 className="font-display text-lg font-bold">{t("ai.itin.title")}</h2>
          </div>
          <Button onClick={generate} disabled={!brief || itinLoading}>
            {itinLoading ? (
              <>
                <Spinner />
                {t("ai.itin.loading")}
              </>
            ) : (
              <>
                <SparkIcon width={16} height={16} />
                {itin ? t("ai.itin.regen") : t("ai.itin.generate")}
              </>
            )}
          </Button>
        </div>

        <div className="mt-4">
          {itinErr && (
            <p className="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm font-medium text-danger">
              {itinErr}
            </p>
          )}
          {!itin && !itinErr && !itinLoading && (
            <p className="text-sm text-muted">{t("ai.itin.empty")}</p>
          )}
          {itinLoading && !itin && (
            <div className="space-y-2">
              {[0, 1, 2, 3].map((i) => (
                <div key={i} className="h-3 animate-pulse rounded bg-surface-2" style={{ width: `${90 - i * 12}%` }} />
              ))}
            </div>
          )}
          {itin && (
            <>
              <div className="whitespace-pre-wrap text-sm leading-relaxed text-text">{itin.text}</div>
              {itin.violations && itin.violations.length > 0 && (
                <p className="mt-3 rounded-lg bg-surface-2 px-3 py-2 text-xs text-muted">
                  {t("ai.itin.violations")}
                </p>
              )}
            </>
          )}
        </div>
      </Card>

      {/* Chat */}
      <Card className="animate-fade-up flex flex-col">
        <div className="flex items-center gap-2">
          <SparkIcon width={18} height={18} className="text-accent" />
          <h2 className="font-display text-lg font-bold">{t("ai.chat.title")}</h2>
        </div>

        <div className="mt-4 flex-1 space-y-2 overflow-y-auto" style={{ maxHeight: "18rem" }}>
          {messages.length === 0 && !chatErr && (
            <p className="text-sm text-muted">{t("ai.chat.empty")}</p>
          )}
          {messages.map((m, i) => (
            <div key={i} className={m.role === "user" ? "flex justify-end" : "flex justify-start"}>
              <span
                className={
                  "max-w-[85%] whitespace-pre-wrap rounded-2xl px-3.5 py-2 text-sm " +
                  (m.role === "user"
                    ? "bg-primary text-primary-fg"
                    : "border border-border bg-surface-2 text-text")
                }
              >
                {m.text}
              </span>
            </div>
          ))}
          {chatLoading && (
            <div className="flex justify-start">
              <span className="inline-flex items-center gap-2 rounded-2xl border border-border bg-surface-2 px-3.5 py-2 text-sm text-muted">
                <Spinner />
                {t("ai.itin.loading")}
              </span>
            </div>
          )}
          {chatErr && (
            <p className="rounded-xl border border-danger/30 bg-danger/10 px-3 py-2 text-sm font-medium text-danger">
              {chatErr}
            </p>
          )}
        </div>

        <form
          className="mt-4 flex items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            void send();
          }}
        >
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={t("ai.chat.placeholder")}
            className="w-full rounded-xl border border-border bg-surface-2 px-3 py-2.5 text-sm text-text transition-colors hover:border-primary/40 focus:border-primary focus:bg-surface"
          />
          <Button type="submit" disabled={chatLoading || input.trim() === ""}>
            {t("ai.chat.send")}
          </Button>
        </form>
      </Card>
    </div>
  );
}
