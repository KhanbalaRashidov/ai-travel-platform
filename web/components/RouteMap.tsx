"use client";

import dynamic from "next/dynamic";
import type { MapPoint } from "./RouteMapInner";

// Leaflet touches `window`, so the map loads only on the client (ssr: false)
// behind a tasteful skeleton. Swapping OSM tiles for Mapbox later is a one-line
// change in RouteMapInner's <TileLayer url>.
const Inner = dynamic(() => import("./RouteMapInner"), {
  ssr: false,
  loading: () => (
    <div className="h-72 w-full animate-pulse rounded-xl bg-surface-2" />
  ),
});

export type { MapPoint };

export function RouteMap({ points, closed }: { points: MapPoint[]; closed: boolean }) {
  return <Inner points={points} closed={closed} />;
}
