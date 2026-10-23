"use client";

import { useEffect } from "react";
import {
  MapContainer,
  TileLayer,
  Polyline,
  CircleMarker,
  Tooltip,
  useMap,
} from "react-leaflet";
import type { LatLngExpression, LatLngBoundsExpression } from "leaflet";
import "leaflet/dist/leaflet.css";

export interface MapPoint {
  name: string;
  lat: number;
  lng: number;
}

// Refits the viewport whenever the path changes so the whole route stays framed.
function FitBounds({ bounds }: { bounds: LatLngBoundsExpression }) {
  const map = useMap();
  useEffect(() => {
    map.fitBounds(bounds, { padding: [36, 36], maxZoom: 7 });
  }, [map, bounds]);
  return null;
}

export default function RouteMapInner({
  points = [],
  closed,
}: {
  points?: MapPoint[];
  closed: boolean;
}) {
  const path: LatLngExpression[] = points.map((p) => [p.lat, p.lng]);
  if (closed && points.length > 1) path.push([points[0].lat, points[0].lng]);
  const bounds: LatLngBoundsExpression = points.map((p) => [p.lat, p.lng]) as [
    number,
    number
  ][];
  const center: LatLngExpression = path[0] ?? [41, 44];

  return (
    <MapContainer
      center={center}
      zoom={5}
      scrollWheelZoom={false}
      className="h-72 w-full rounded-xl"
      style={{ zIndex: 0 }}
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />
      <Polyline
        positions={path}
        pathOptions={{
          color: "rgb(99 102 241)",
          weight: 3,
          opacity: 0.9,
          dashArray: "6 8",
        }}
      />
      {points.map((p, i) => (
        <CircleMarker
          key={`${p.name}-${i}`}
          center={[p.lat, p.lng]}
          radius={8}
          pathOptions={{
            color: "#fff",
            weight: 2,
            fillColor: "rgb(79 70 229)",
            fillOpacity: 1,
          }}
        >
          <Tooltip direction="top" offset={[0, -8]}>
            {i + 1}. {p.name}
          </Tooltip>
        </CircleMarker>
      ))}
      {points.length > 1 && <FitBounds bounds={bounds} />}
    </MapContainer>
  );
}
