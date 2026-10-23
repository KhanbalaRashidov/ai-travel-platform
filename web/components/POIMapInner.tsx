"use client";

import { useEffect } from "react";
import { MapContainer, TileLayer, CircleMarker, Polyline, Tooltip, Popup, useMap } from "react-leaflet";
import type { LatLngExpression, LatLngBoundsExpression } from "leaflet";
import "leaflet/dist/leaflet.css";

export interface POIPoint {
  name: string;
  lat: number;
  lng: number;
  day?: number;
  mapsUrl?: string;
}

function FitBounds({ bounds }: { bounds: LatLngBoundsExpression }) {
  const map = useMap();
  useEffect(() => {
    map.fitBounds(bounds, { padding: [36, 36], maxZoom: 13 });
  }, [map, bounds]);
  return null;
}

// POIMapInner plots every itinerary POI as a pin with its name + a real Google
// Maps link in the popup. No connecting line — these are places, not a route.
export default function POIMapInner({ points = [] }: { points?: POIPoint[] }) {
  const pts = points.filter((p) => p.lat !== 0 || p.lng !== 0);
  const bounds: LatLngBoundsExpression = pts.map((p) => [p.lat, p.lng]) as [number, number][];
  const center: LatLngExpression = pts.length ? [pts[0].lat, pts[0].lng] : [41, 44];
  // Connect the POIs in visit order so the map reads as one roadmap, not dots.
  const path: LatLngExpression[] = pts.map((p) => [p.lat, p.lng]);

  return (
    <MapContainer
      center={center}
      zoom={11}
      scrollWheelZoom={false}
      className="h-72 w-full rounded-xl"
      style={{ zIndex: 0 }}
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />
      {path.length > 1 && (
        <Polyline
          positions={path}
          pathOptions={{ color: "rgb(16 185 129)", weight: 3, opacity: 0.85, dashArray: "6 8" }}
        />
      )}
      {pts.map((p, i) => (
        <CircleMarker
          key={`${p.name}-${i}`}
          center={[p.lat, p.lng]}
          radius={7}
          pathOptions={{ color: "#fff", weight: 2, fillColor: "rgb(16 185 129)", fillOpacity: 1 }}
        >
          <Tooltip direction="top" offset={[0, -6]}>
            {typeof p.day === "number" ? `${p.day}. ` : ""}
            {p.name}
          </Tooltip>
          {p.mapsUrl && (
            <Popup>
              <a href={p.mapsUrl} target="_blank" rel="noopener noreferrer">
                {p.name} ↗
              </a>
            </Popup>
          )}
        </CircleMarker>
      ))}
      {pts.length > 1 && <FitBounds bounds={bounds} />}
    </MapContainer>
  );
}
