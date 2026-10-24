import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// Vitest for the web app: jsdom DOM + React Testing Library. Scoped to unit and
// component tests (lib/, components/) — it does not touch the Next build.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./vitest.setup.ts"],
    include: ["components/**/*.test.{ts,tsx}", "lib/**/*.test.{ts,tsx}"],
  },
});
