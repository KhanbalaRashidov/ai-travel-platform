"use client";

import { useEffect, useRef, useState } from "react";
import { Place, searchPlaces } from "../lib/api";
import { useI18n } from "../lib/i18n";
import { PinIcon } from "./icons";

// CityAutocomplete is a debounced search box over the real places endpoint. It
// has no hardcoded city list — results (with coordinates) come from the API.
export function CityAutocomplete({
  onAdd,
  placeholder,
  isSelected,
}: {
  onAdd: (p: Place) => void;
  placeholder: string;
  isSelected: (name: string) => boolean;
}) {
  const { locale } = useI18n();
  const [q, setQ] = useState("");
  const [sug, setSug] = useState<Place[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);

  // Debounced search.
  useEffect(() => {
    if (q.trim().length < 2) {
      setSug([]);
      return;
    }
    let active = true;
    setLoading(true);
    const id = setTimeout(async () => {
      try {
        const r = await searchPlaces(q, locale);
        if (active) {
          setSug(r ?? []);
          setOpen(true);
        }
      } catch {
        if (active) setSug([]);
      } finally {
        if (active) setLoading(false);
      }
    }, 250);
    return () => {
      active = false;
      clearTimeout(id);
    };
  }, [q, locale]);

  // Close on outside click.
  useEffect(() => {
    function onDoc(e: MouseEvent) {
      if (boxRef.current && !boxRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);

  function pick(p: Place) {
    onAdd(p);
    setQ("");
    setSug([]);
    setOpen(false);
  }

  return (
    <div ref={boxRef} className="relative">
      <div className="flex items-center gap-2 rounded-xl border border-border bg-surface-2 px-3 transition-colors focus-within:border-primary focus-within:bg-surface">
        <PinIcon width={16} height={16} className="shrink-0 text-muted" />
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onFocus={() => sug.length > 0 && setOpen(true)}
          placeholder={placeholder}
          className="w-full bg-transparent py-2.5 text-sm text-text outline-none"
        />
        {loading && (
          <span className="h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-muted border-t-transparent opacity-60" />
        )}
      </div>

      {open && sug.length > 0 && (
        <ul className="absolute z-20 mt-1.5 max-h-72 w-full overflow-auto rounded-xl border border-border bg-surface p-1 shadow-lift">
          {sug.map((p) => {
            const taken = isSelected(p.name);
            return (
              <li key={`${p.code}-${p.name}-${p.lat}`}>
                <button
                  type="button"
                  disabled={taken}
                  // Commit the selection on pointer-DOWN (not click): a late
                  // debounced search response or an input blur can re-render this
                  // list and replace the button between mousedown and click, which
                  // would otherwise swallow the selection. preventDefault keeps the
                  // input focused. (Touch taps also dispatch mousedown.)
                  onMouseDown={(e) => {
                    e.preventDefault();
                    if (!taken) pick(p);
                  }}
                  className="flex w-full items-center justify-between gap-3 rounded-lg px-3 py-2 text-left text-sm transition-colors hover:bg-surface-2 disabled:opacity-40"
                >
                  <span className="min-w-0 truncate">
                    <span className="font-medium">{p.name}</span>
                    <span className="text-muted">
                      {" "}
                      · {p.country}
                      {p.kind === "airport" ? " ✈" : ""}
                    </span>
                  </span>
                  <span className="shrink-0 text-xs font-semibold text-muted">{p.code}</span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
