"use client";

import Link from "next/link";
import { useI18n } from "../lib/i18n";
import { AppShell } from "../components/AppShell";
import { Badge, Card } from "../components/ui";
import { ArrowRightIcon, RouteIcon, WalletIcon, SparkIcon } from "../components/icons";

export default function Home() {
  const { t } = useI18n();

  const features = [
    { icon: RouteIcon, t: "feature.route.t", d: "feature.route.d", tone: "text-primary bg-primary/10" },
    { icon: WalletIcon, t: "feature.cost.t", d: "feature.cost.d", tone: "text-activities bg-activities/10" },
    { icon: SparkIcon, t: "feature.smart.t", d: "feature.smart.d", tone: "text-hotels bg-hotels/10" },
  ] as const;

  return (
    <AppShell>
      <main className="mx-auto max-w-5xl px-4">
        <section className="py-16 text-center sm:py-24">
          <div className="animate-fade-up">
            <Badge tone="primary">{t("app.badge")}</Badge>
          </div>
          <h1 className="animate-fade-up mt-5 font-display text-4xl font-extrabold tracking-tight sm:text-6xl" style={{ animationDelay: "60ms" }}>
            {t("app.title")}
          </h1>
          <p className="animate-fade-up mx-auto mt-5 max-w-2xl text-lg text-muted" style={{ animationDelay: "120ms" }}>
            {t("app.tagline")}
          </p>
          <p className="animate-fade-up mx-auto mt-2 max-w-xl text-sm text-muted/80" style={{ animationDelay: "160ms" }}>
            {t("app.heroLead")}
          </p>
          <div className="animate-fade-up mt-8 flex justify-center" style={{ animationDelay: "220ms" }}>
            <Link
              href="/plan"
              className="group inline-flex items-center gap-2 rounded-full bg-primary px-7 py-3.5 text-base font-semibold text-primary-fg shadow-lift transition-all hover:bg-primary-strong hover:shadow-soft active:translate-y-px"
            >
              {t("app.cta")}
              <ArrowRightIcon width={18} height={18} className="transition-transform group-hover:translate-x-0.5" />
            </Link>
          </div>
        </section>

        <section className="grid gap-4 pb-16 sm:grid-cols-3">
          {features.map((f, i) => (
            <Card key={f.t} className="animate-fade-up transition-transform hover:-translate-y-1" >
              <div className={`grid h-11 w-11 place-items-center rounded-xl ${f.tone}`} style={{ animationDelay: `${i * 80}ms` }}>
                <f.icon width={20} height={20} />
              </div>
              <h3 className="mt-4 font-display text-lg font-bold">{t(f.t)}</h3>
              <p className="mt-2 text-sm leading-relaxed text-muted">{t(f.d)}</p>
            </Card>
          ))}
        </section>
      </main>
    </AppShell>
  );
}
