"use client";

import { createContext, useContext, useEffect, useState } from "react";

export type Locale = "az" | "en" | "ru" | "tr";
export const LOCALES: Locale[] = ["az", "en", "ru", "tr"];
export const LOCALE_LABEL: Record<Locale, string> = {
  az: "AZ",
  en: "EN",
  ru: "RU",
  tr: "TR",
};
const DEFAULT_LOCALE: Locale = "az";
const FALLBACK: Locale = "en";

// Default display currency per locale (we only carry EUR/USD/AZN rates).
export const LOCALE_CURRENCY: Record<Locale, string> = {
  az: "AZN",
  en: "EUR",
  ru: "EUR",
  tr: "EUR",
};

type Dict = Record<string, string>;

const MESSAGES: Record<Locale, Dict> = {
  az: {
    "app.title": "AI Travel Platform",
    "app.tagline":
      "Çox-ölkəli ən sərfəli marşrut — ən ucuz yol, tier üzrə otellər, gün-gün plan və canlı total. Yalnız məlumatlandırma.",
    "app.cta": "Marşrutu planla →",
    "app.badge": "Yalnız məlumatlandırma — bron/ödəniş yoxdur",
    "app.heroLead": "Şəhərləri seç — qalanını alqoritm həll edir.",
    "feature.route.t": "Ən sərfəli sıra",
    "feature.route.d": "Sabit başlanğıc/son node ilə TSP marşrutu; gediş-dönüş və açıq-çənə variantlarını müqayisə edir.",
    "feature.cost.t": "Canlı xərc",
    "feature.cost.d": "Nəqliyyat, otel və aktivliklər — rahatlıq dəyişdikcə total anında yenilənir.",
    "feature.smart.t": "Özünü-böyüdən baza",
    "feature.smart.d": "Qiymətlər real mənbələrdən keşlənir və hər axtarışla zənginləşir (mənbə + tarix etiketli).",
    "plan.back": "← Ana səhifə",
    "res.map": "Xəritə",
    "res.planTitle": "Sənin marşrutun",
    "res.emptyTitle": "Marşrut hələ hesablanmayıb",
    "res.emptyDesc": "Şəhərləri seçib “Marşrutu hesabla”ya bas — nəticə burada görünəcək.",
    "plan.title": "Ən sərfəli marşrut",
    "plan.subtitle":
      "Şəhərləri seç, sistem ən ucuz sıranı çıxarır; rahatlıq/nəqliyyat dəyişdikcə total anında yenilənir.",
    "plan.cities": "Şəhərlər",
    "plan.start": "Başlanğıc",
    "plan.comfort": "Rahatlıq",
    "plan.transport": "Nəqliyyat",
    "plan.nights": "Gecə / şəhər",
    "plan.currency": "Valyuta",
    "plan.calc": "Marşrutu hesabla",
    "plan.calculating": "Hesablanır...",
    "tier.budget": "Budget (3★)",
    "tier.standard": "Standart (4★)",
    "tier.vip": "VIP (5★)",
    "mode.cheapest": "Ən ucuz",
    "mode.flight": "Təyyarə",
    "mode.train": "Qatar",
    "mode.bus": "Avtobus",
    "res.route": "Marşrut",
    "res.roundtrip": "gediş-dönüş",
    "res.openjaw": "açıq-çənə",
    "res.alt": "Alternativ",
    "res.liveCost": "Canlı xərc",
    "res.transport": "Nəqliyyat",
    "res.hotels": "Otellər",
    "res.activities": "Aktivliklər",
    "res.total": "Cəmi",
    "res.disclaimer":
      "Qiymətlər indikativdir (demo data). Dəyişikliklər yalnız yenidən qiymətləndirir — marşrut sırası dəyişmir.",
    "err.minCities": "Ən azı 2 şəhər seçin.",
    "err.startInvalid": "Başlanğıc şəhər seçilmiş şəhərlər arasında olmalıdır.",
    "err.modeImpossible":
      "\"{mode}\" bu marşrut üçün mümkün deyil ({from} → {to}). Başqa nəqliyyat seçin.",
    "err.sequence": "marşrut hesablanmadı",
    "err.price": "qiymətləndirmə alınmadı",
  },
  en: {
    "app.title": "AI Travel Platform",
    "app.tagline":
      "Cost-optimal multi-country roadmaps — cheapest route, hotels by tier, day-by-day plan, live total. Advisory only.",
    "app.cta": "Plan a route →",
    "app.badge": "Advisory only — no booking, no payment",
    "app.heroLead": "Pick your cities — the algorithm handles the rest.",
    "feature.route.t": "Cheapest order",
    "feature.route.d": "Fixed start/end TSP routing; compares round-trip and open-jaw variants.",
    "feature.cost.t": "Live cost",
    "feature.cost.d": "Transport, hotels and activities — the total updates instantly as comfort changes.",
    "feature.smart.t": "Self-growing data",
    "feature.smart.d": "Prices are cached from real sources and enrich with every search (source + date tagged).",
    "plan.back": "← Home",
    "res.map": "Map",
    "res.planTitle": "Your roadmap",
    "res.emptyTitle": "No route yet",
    "res.emptyDesc": "Pick cities and hit “Compute route” — your result will appear here.",
    "plan.title": "Cheapest route",
    "plan.subtitle":
      "Pick cities; the engine finds the cheapest order. Changing comfort/transport updates the total instantly.",
    "plan.cities": "Cities",
    "plan.start": "Start",
    "plan.comfort": "Comfort",
    "plan.transport": "Transport",
    "plan.nights": "Nights / city",
    "plan.currency": "Currency",
    "plan.calc": "Compute route",
    "plan.calculating": "Computing...",
    "tier.budget": "Budget (3★)",
    "tier.standard": "Standard (4★)",
    "tier.vip": "VIP (5★)",
    "mode.cheapest": "Cheapest",
    "mode.flight": "Flight",
    "mode.train": "Train",
    "mode.bus": "Bus",
    "res.route": "Route",
    "res.roundtrip": "round-trip",
    "res.openjaw": "open-jaw",
    "res.alt": "Alternative",
    "res.liveCost": "Live cost",
    "res.transport": "Transport",
    "res.hotels": "Hotels",
    "res.activities": "Activities",
    "res.total": "Total",
    "res.disclaimer":
      "Prices are indicative (demo data). Changes only re-price — the route order does not change.",
    "err.minCities": "Select at least 2 cities.",
    "err.startInvalid": "Start city must be among the selected cities.",
    "err.modeImpossible":
      "\"{mode}\" is not possible for this route ({from} → {to}). Choose another transport.",
    "err.sequence": "could not compute the route",
    "err.price": "pricing failed",
  },
  ru: {
    "app.title": "AI Travel Platform",
    "app.tagline":
      "Оптимальные по цене маршруты по нескольким странам — самый дешёвый путь, отели по уровню, план по дням, живой итог. Только информация.",
    "app.cta": "Спланировать маршрут →",
    "app.badge": "Только информация — без брони и оплаты",
    "app.heroLead": "Выберите города — остальное сделает алгоритм.",
    "feature.route.t": "Самый дешёвый порядок",
    "feature.route.d": "Маршрут TSP с фиксированным началом/концом; сравнивает туда-обратно и open-jaw.",
    "feature.cost.t": "Живой расчёт",
    "feature.cost.d": "Транспорт, отели и активности — итог обновляется мгновенно при смене комфорта.",
    "feature.smart.t": "Саморазрастающаяся база",
    "feature.smart.d": "Цены кэшируются из реальных источников и обогащаются с каждым поиском (метка источник+дата).",
    "plan.back": "← На главную",
    "res.map": "Карта",
    "res.planTitle": "Ваш маршрут",
    "res.emptyTitle": "Маршрут ещё не рассчитан",
    "res.emptyDesc": "Выберите города и нажмите «Рассчитать маршрут» — результат появится здесь.",
    "plan.title": "Самый дешёвый маршрут",
    "plan.subtitle":
      "Выберите города; движок находит самый дешёвый порядок. Смена комфорта/транспорта мгновенно обновляет итог.",
    "plan.cities": "Города",
    "plan.start": "Старт",
    "plan.comfort": "Комфорт",
    "plan.transport": "Транспорт",
    "plan.nights": "Ночей / город",
    "plan.currency": "Валюта",
    "plan.calc": "Рассчитать маршрут",
    "plan.calculating": "Расчёт...",
    "tier.budget": "Бюджет (3★)",
    "tier.standard": "Стандарт (4★)",
    "tier.vip": "VIP (5★)",
    "mode.cheapest": "Дешевле всего",
    "mode.flight": "Самолёт",
    "mode.train": "Поезд",
    "mode.bus": "Автобус",
    "res.route": "Маршрут",
    "res.roundtrip": "туда-обратно",
    "res.openjaw": "открытая челюсть",
    "res.alt": "Альтернатива",
    "res.liveCost": "Живой расчёт",
    "res.transport": "Транспорт",
    "res.hotels": "Отели",
    "res.activities": "Активности",
    "res.total": "Итого",
    "res.disclaimer":
      "Цены ориентировочные (демо-данные). Изменения только пересчитывают — порядок маршрута не меняется.",
    "err.minCities": "Выберите минимум 2 города.",
    "err.startInvalid": "Город старта должен быть среди выбранных.",
    "err.modeImpossible":
      "«{mode}» невозможен для этого маршрута ({from} → {to}). Выберите другой транспорт.",
    "err.sequence": "не удалось рассчитать маршрут",
    "err.price": "ошибка расчёта цены",
  },
  tr: {
    "app.title": "AI Travel Platform",
    "app.tagline":
      "Çok ülkeli maliyet-optimal rotalar — en ucuz güzergah, seviyeye göre oteller, gün gün plan, canlı toplam. Yalnızca bilgi.",
    "app.cta": "Rota planla →",
    "app.badge": "Yalnızca bilgi — rezervasyon/ödeme yok",
    "app.heroLead": "Şehirleri seç — gerisini algoritma halleder.",
    "feature.route.t": "En ucuz sıra",
    "feature.route.d": "Sabit başlangıç/bitiş düğümlü TSP rotası; gidiş-dönüş ve open-jaw karşılaştırır.",
    "feature.cost.t": "Canlı maliyet",
    "feature.cost.d": "Ulaşım, otel ve aktiviteler — konfor değişince toplam anında güncellenir.",
    "feature.smart.t": "Kendiliğinden büyüyen veri",
    "feature.smart.d": "Fiyatlar gerçek kaynaklardan önbelleğe alınır ve her aramayla zenginleşir (kaynak + tarih etiketli).",
    "plan.back": "← Ana sayfa",
    "res.map": "Harita",
    "res.planTitle": "Rotanız",
    "res.emptyTitle": "Henüz rota yok",
    "res.emptyDesc": "Şehirleri seçip “Rotayı hesapla”ya bas — sonucun burada görünecek.",
    "plan.title": "En ucuz rota",
    "plan.subtitle":
      "Şehirleri seç; motor en ucuz sırayı bulur. Konfor/ulaşım değişince toplam anında güncellenir.",
    "plan.cities": "Şehirler",
    "plan.start": "Başlangıç",
    "plan.comfort": "Konfor",
    "plan.transport": "Ulaşım",
    "plan.nights": "Gece / şehir",
    "plan.currency": "Para birimi",
    "plan.calc": "Rotayı hesapla",
    "plan.calculating": "Hesaplanıyor...",
    "tier.budget": "Budget (3★)",
    "tier.standard": "Standart (4★)",
    "tier.vip": "VIP (5★)",
    "mode.cheapest": "En ucuz",
    "mode.flight": "Uçak",
    "mode.train": "Tren",
    "mode.bus": "Otobüs",
    "res.route": "Rota",
    "res.roundtrip": "gidiş-dönüş",
    "res.openjaw": "açık çene",
    "res.alt": "Alternatif",
    "res.liveCost": "Canlı maliyet",
    "res.transport": "Ulaşım",
    "res.hotels": "Oteller",
    "res.activities": "Aktiviteler",
    "res.total": "Toplam",
    "res.disclaimer":
      "Fiyatlar göstergeseldir (demo veri). Değişiklikler yalnızca yeniden fiyatlandırır — rota sırası değişmez.",
    "err.minCities": "En az 2 şehir seçin.",
    "err.startInvalid": "Başlangıç şehri seçilenler arasında olmalı.",
    "err.modeImpossible":
      "\"{mode}\" bu rota için mümkün değil ({from} → {to}). Başka ulaşım seçin.",
    "err.sequence": "rota hesaplanamadı",
    "err.price": "fiyatlandırma başarısız",
  },
};

interface Ctx {
  locale: Locale;
  setLocale: (l: Locale) => void;
  t: (key: string, vars?: Record<string, string | number>) => string;
}

const LocaleContext = createContext<Ctx | null>(null);

export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(DEFAULT_LOCALE);

  useEffect(() => {
    try {
      const saved = localStorage.getItem("locale") as Locale | null;
      if (saved && LOCALES.includes(saved)) setLocaleState(saved);
    } catch {
      /* localStorage may be unavailable */
    }
  }, []);

  const setLocale = (l: Locale) => {
    setLocaleState(l);
    try {
      localStorage.setItem("locale", l);
    } catch {
      /* ignore */
    }
  };

  const t = (key: string, vars?: Record<string, string | number>) => {
    let s = MESSAGES[locale][key] ?? MESSAGES[FALLBACK][key] ?? key;
    if (vars) {
      for (const [k, v] of Object.entries(vars)) {
        s = s.replace(new RegExp(`\\{${k}\\}`, "g"), String(v));
      }
    }
    return s;
  };

  return (
    <LocaleContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </LocaleContext.Provider>
  );
}

export function useI18n(): Ctx {
  const ctx = useContext(LocaleContext);
  if (!ctx) throw new Error("useI18n must be used within LocaleProvider");
  return ctx;
}

export function LanguageSwitcher() {
  const { locale, setLocale } = useI18n();
  return (
    <div className="flex items-center gap-0.5 rounded-full border border-border bg-surface p-0.5">
      {LOCALES.map((l) => (
        <button
          key={l}
          onClick={() => setLocale(l)}
          aria-pressed={l === locale}
          className={`rounded-full px-2.5 py-1 text-xs font-semibold transition-colors ${
            l === locale
              ? "bg-primary text-primary-fg shadow-soft"
              : "text-muted hover:text-text"
          }`}
        >
          {LOCALE_LABEL[l]}
        </button>
      ))}
    </div>
  );
}
