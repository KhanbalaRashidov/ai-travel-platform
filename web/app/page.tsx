"use client";

import Link from "next/link";
import { LanguageSwitcher, useI18n } from "../lib/i18n";

export default function Home() {
  const { t } = useI18n();
  return (
    <main className="mx-auto max-w-2xl px-4 py-16">
      <div className="flex justify-end">
        <LanguageSwitcher />
      </div>
      <h1 className="mt-4 text-3xl font-bold">{t("app.title")}</h1>
      <p className="mt-4 text-gray-600">{t("app.tagline")}</p>
      <Link
        href="/plan"
        className="mt-6 inline-block rounded bg-blue-600 px-5 py-2.5 text-sm font-medium text-white"
      >
        {t("app.cta")}
      </Link>
    </main>
  );
}
