import type { Metadata } from "next";
import { Inter, Sora } from "next/font/google";
import "./globals.css";
import { LocaleProvider } from "../lib/i18n";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { ToastProvider } from "../components/toast";

const inter = Inter({
  subsets: ["latin", "latin-ext", "cyrillic"],
  variable: "--font-sans",
  display: "swap",
});

const sora = Sora({
  subsets: ["latin", "latin-ext"],
  variable: "--font-display",
  display: "swap",
});

const siteURL = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";

export const metadata: Metadata = {
  metadataBase: new URL(siteURL),
  title: "AI Travel Platform — cost-optimal trip roadmaps",
  description:
    "Cost-optimal multi-country trip roadmaps: cheapest route, hotels by tier, day-by-day plan and a live total. Advisory only.",
  openGraph: {
    title: "AI Travel Platform — cost-optimal trip roadmaps",
    description:
      "Find the cheapest multi-country route, hotels by tier, a day-by-day plan and a live total. Advisory only — no booking.",
    type: "website",
    url: siteURL,
  },
  twitter: { card: "summary_large_image", title: "AI Travel Platform" },
};

// Set the theme class before paint so there is no light/dark flash.
const themeBootstrap = `(function(){try{var t=localStorage.getItem('theme');var d=document.documentElement;if(t==='dark'){d.classList.add('dark')}else if(t==='light'){d.classList.add('light')}}catch(e){}})();`;

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="az" className={`${inter.variable} ${sora.variable}`}>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeBootstrap }} />
      </head>
      <body className="font-sans antialiased">
        <LocaleProvider>
          <ErrorBoundary>
            <ToastProvider>{children}</ToastProvider>
          </ErrorBoundary>
        </LocaleProvider>
      </body>
    </html>
  );
}
