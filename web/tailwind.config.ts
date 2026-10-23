import type { Config } from "tailwindcss";

const withAlpha = (v: string) => `rgb(var(${v}) / <alpha-value>)`;

const config: Config = {
  darkMode: "class",
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: withAlpha("--bg"),
        surface: withAlpha("--surface"),
        "surface-2": withAlpha("--surface-2"),
        border: withAlpha("--border"),
        text: withAlpha("--text"),
        muted: withAlpha("--muted"),
        primary: {
          DEFAULT: withAlpha("--primary"),
          strong: withAlpha("--primary-strong"),
          fg: withAlpha("--primary-fg"),
        },
        accent: withAlpha("--accent"),
        ring: withAlpha("--ring"),
        transport: withAlpha("--transport"),
        hotels: withAlpha("--hotels"),
        activities: withAlpha("--activities"),
        success: withAlpha("--success"),
        danger: withAlpha("--danger"),
      },
      fontFamily: {
        sans: ["var(--font-sans)", "system-ui", "sans-serif"],
        display: ["var(--font-display)", "var(--font-sans)", "sans-serif"],
      },
      borderRadius: {
        xl: "1rem",
        "2xl": "1.25rem",
      },
      boxShadow: {
        soft: "0 1px 2px rgb(var(--shadow-color) / 0.04), 0 8px 24px -12px rgb(var(--shadow-color) / 0.12)",
        lift: "0 2px 4px rgb(var(--shadow-color) / 0.06), 0 18px 40px -18px rgb(var(--shadow-color) / 0.25)",
      },
      keyframes: {
        "fade-up": {
          from: { opacity: "0", transform: "translateY(10px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
        "fade-in": {
          from: { opacity: "0" },
          to: { opacity: "1" },
        },
        shimmer: {
          "100%": { transform: "translateX(100%)" },
        },
        "draw-in": {
          from: { strokeDashoffset: "var(--dash)" },
          to: { strokeDashoffset: "0" },
        },
      },
      animation: {
        "fade-up": "fade-up 0.5s cubic-bezier(0.22, 1, 0.36, 1) both",
        "fade-in": "fade-in 0.4s ease both",
        shimmer: "shimmer 1.6s infinite",
        "draw-in": "draw-in 0.9s ease-out both",
      },
    },
  },
  plugins: [],
};

export default config;
