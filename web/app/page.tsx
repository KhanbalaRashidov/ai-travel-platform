import Link from "next/link";

export default function Home() {
  return (
    <main className="mx-auto max-w-2xl px-4 py-16">
      <h1 className="text-3xl font-bold">AI Travel Platform</h1>
      <p className="mt-4 text-gray-600">
        Cost-optimal multi-country trip roadmaps — cheapest route, hotels by
        tier, day-by-day plan, and a live total cost. Advisory only.
      </p>
      <Link
        href="/plan"
        className="mt-6 inline-block rounded bg-blue-600 px-5 py-2.5 text-sm font-medium text-white"
      >
        Marşrutu planla →
      </Link>
      <p className="mt-3 text-sm text-gray-400">E5 — wizard + nəticə + canlı total.</p>
    </main>
  );
}
