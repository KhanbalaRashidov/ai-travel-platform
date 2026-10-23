import type { Metadata } from "next";
import { Inter, Sora } from "next/font/google";
import "./globals.css";
import { LocaleProvider } from "../lib/i18n";

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

export const metadata: Metadata = {
  title: "AI Travel Platform — cost-optimal trip roadmaps",
  description:
    "Cost-optimal multi-country trip roadmaps: cheapest route, hotels by tier, day-by-day plan and a live total. Advisory only.",
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
        <LocaleProvider>{children}</LocaleProvider>
      </body>
    </html>
  );
}
