"use client";

import dynamic from "next/dynamic";
import type { POIPoint } from "./POIMapInner";

// Leaflet touches `window`, so the POI map loads client-only behind a skeleton.
const Inner = dynamic(() => import("./POIMapInner"), {
  ssr: false,
  loading: () => <div className="h-72 w-full animate-pulse rounded-xl bg-surface-2" />,
});

export type { POIPoint };

export function POIMap({ points }: { points: POIPoint[] }) {
  return <Inner points={points} />;
}
