import type { MetadataRoute } from "next";
import { siteUrl } from "../lib/site";

const base = siteUrl();

export default function sitemap(): MetadataRoute.Sitemap {
  const now = new Date();
  return [
    { url: base, lastModified: now, changeFrequency: "weekly", priority: 1 },
    { url: `${base}/plan`, lastModified: now, changeFrequency: "weekly", priority: 0.8 },
  ];
}
