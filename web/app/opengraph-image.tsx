import { ImageResponse } from "next/og";

// Dynamic Open Graph / Twitter card (E2): a real PNG generated from the brand
// mark + name + tagline, so shared links render a promo image. App Router serves
// this at /opengraph-image and wires it into the card metadata automatically.
export const alt = "AI Travel — cost-optimal multi-country trip roadmaps";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          padding: "80px",
          background: "linear-gradient(135deg, #0A0E24 0%, #15193a 100%)",
          color: "#F8FAFC",
          fontFamily: "sans-serif",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 28 }}>
          <div
            style={{
              width: 108,
              height: 108,
              borderRadius: 30,
              background: "linear-gradient(135deg, #8B7CF6 0%, #2DD4BF 100%)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <svg width="68" height="68" viewBox="0 0 54 54">
              <path d="M27 12 L38 39 L27 32 L16 39 Z" fill="#0A0E24" />
            </svg>
          </div>
          <div style={{ fontSize: 68, fontWeight: 800, letterSpacing: -1 }}>AI Travel</div>
        </div>
        <div style={{ marginTop: 40, fontSize: 40, fontWeight: 600, color: "#CBD5E1" }}>
          Cost-optimal multi-country trip roadmaps
        </div>
        <div style={{ marginTop: 18, fontSize: 26, color: "#94A3B8" }}>
          Cheapest city sequence · all transport modes · day-by-day plan · live cost
        </div>
      </div>
    ),
    size
  );
}
