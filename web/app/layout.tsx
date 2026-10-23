import type { Metadata } from "next";
import "./globals.css";
import { LocaleProvider } from "../lib/i18n";

export const metadata: Metadata = {
  title: "AI Travel Platform",
  description: "Cost-optimal multi-country trip roadmaps (advisory only).",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="az">
      <body>
        <LocaleProvider>{children}</LocaleProvider>
      </body>
    </html>
  );
}
