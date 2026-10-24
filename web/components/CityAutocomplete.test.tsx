import { useState } from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Place } from "../lib/api";
import { LocaleProvider, useI18n } from "../lib/i18n";
import { Field } from "./ui";

// The autocomplete calls searchPlaces(q, locale); mock it so the test is
// network-free and deterministic. Different localized names per locale prove the
// selection keeps working regardless of language.
const PARIS_EN: Place = { code: "PAR", name: "Paris", country: "France", lat: 48.85, lng: 2.35, kind: "city" };
const PARIS_RU: Place = { ...PARIS_EN, name: "Париж" };

const { searchPlaces } = vi.hoisted(() => ({ searchPlaces: vi.fn() }));
vi.mock("../lib/api", () => ({ searchPlaces }));

// eslint-disable-next-line import/first
import { CityAutocomplete } from "./CityAutocomplete";

searchPlaces.mockImplementation(async (q: string, locale = "en"): Promise<Place[]> => {
  if (!q || q.trim().length < 2) return [];
  return [locale === "ru" ? PARIS_RU : PARIS_EN];
});

// Harness that mirrors the plan page's "From? (origin)" section exactly: a Field
// (which wraps children in a <label>) + a conditional chip, plus a locale toggle.
function OriginHarness() {
  const [origin, setOrigin] = useState<Place | null>(null);
  const { setLocale } = useI18n();
  return (
    <div>
      <button onClick={() => setLocale("ru")}>to-ru</button>
      <Field label="From? (origin)">
        {origin ? (
          <span data-testid="origin-chip">
            {origin.name}
            <button aria-label="remove origin" onClick={() => setOrigin(null)}>
              ×
            </button>
          </span>
        ) : (
          <CityAutocomplete onAdd={(p) => setOrigin(p)} placeholder="From?" isSelected={() => false} />
        )}
      </Field>
    </div>
  );
}

function renderHarness() {
  return render(
    <LocaleProvider>
      <OriginHarness />
    </LocaleProvider>,
  );
}

describe("CityAutocomplete — origin selection", () => {
  beforeEach(() => searchPlaces.mockClear());

  it("selecting a suggestion fills state (the chip appears)", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.type(screen.getByPlaceholderText("From?"), "par");
    const option = await screen.findByRole("button", { name: /Paris/ }, { timeout: 2000 });
    await user.click(option);

    const chip = await screen.findByTestId("origin-chip");
    expect(chip).toHaveTextContent("Paris");
  });

  it("commits the pick on pointer-down, before any later re-render can drop it", async () => {
    // Regression guard: the selection must register on mousedown alone. On the
    // old onClick path, a late debounced response (or blur) re-rendering the list
    // between mousedown and click swallowed the pick, so the origin stayed empty.
    const user = userEvent.setup();
    renderHarness();

    await user.type(screen.getByPlaceholderText("From?"), "par");
    const option = await screen.findByRole("button", { name: /Paris/ }, { timeout: 2000 });
    fireEvent.mouseDown(option); // mousedown only — no click event follows
    expect(await screen.findByTestId("origin-chip")).toHaveTextContent("Paris");
  });

  it("the selection persists when the UI language changes", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.type(screen.getByPlaceholderText("From?"), "par");
    await user.click(await screen.findByRole("button", { name: /Paris/ }, { timeout: 2000 }));
    expect(await screen.findByTestId("origin-chip")).toHaveTextContent("Paris");

    // Switching language must NOT drop the selected origin.
    await user.click(screen.getByText("to-ru"));
    await waitFor(() => expect(screen.getByTestId("origin-chip")).toBeInTheDocument());
    expect(screen.getByTestId("origin-chip")).toHaveTextContent("Paris");
  });
});
