import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MetadataSettings } from "../../../api/gen/model";
import { renderApp } from "../../../test/render-app";

const about = { source: "database", applies: "live", pendingRestart: false } as const;

const choice = (key: string, label: string, value: string, options: [string, string][]) => ({
  ...about,
  kind: "choice" as const,
  key,
  label,
  isDefault: value === options[0]?.[0],
  value,
  running: value,
  default: options[0]?.[0] ?? value,
  options: options.map(([option, name]) => ({ value: option, label: name })),
});

const day = 86_400;

const metadataSettings = (region: string): MetadataSettings => ({
  settings: [
    choice("metadata.language", "Metadata language", "en-US", [
      ["en-US", "English (United States)"],
      ["fr-FR", "French (France)"],
      ["de-DE", "German (Germany)"],
      ["ja-JP", "Japanese (Japan)"],
    ]),
    choice("metadata.region", "Metadata region", region, [
      ["US", "United States"],
      ["FR", "France"],
      ["DE", "Germany"],
      ["GB", "United Kingdom"],
    ]),
    {
      ...about,
      kind: "duration",
      key: "metadata.cacheFor",
      label: "Metadata cache keeps responses for",
      isDefault: true,
      value: 3600,
      running: 3600,
      default: 3600,
      min: 0,
      max: day,
      step: 60,
    },
    {
      ...about,
      kind: "count",
      key: "metadata.artworkStore",
      label: "Artwork store size",
      isDefault: true,
      value: 2048,
      running: 2048,
      default: 2048,
      min: 100,
      max: 1_048_576,
      unit: { one: "MB", many: "MB" },
    },
  ],
});

test("Metadata leads with language and region in words and keeps the cache under Details", async () => {
  const sent: unknown[] = [];
  let cleared = 0;
  let region = "US";
  renderApp({
    path: "/settings/metadata",
    handlers: [
      http.get("*/api/v1/metadata/settings", () => HttpResponse.json(metadataSettings(region))),
      http.patch("*/api/v1/metadata/settings", async ({ request }) => {
        sent.push(await request.json());
        region = "GB";
        return HttpResponse.json(metadataSettings(region));
      }),
      http.delete("*/api/v1/metadata/cache", () => {
        cleared += 1;
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  });
  await expect
    .element(
      page.getByText(
        "Titles and overviews are in English (United States), and release dates and certifications come from United States.",
        { exact: false },
      ),
    )
    .toBeVisible();
  await expect.element(page.getByRole("combobox", { name: "Metadata language" })).toBeVisible();
  await expect
    .element(page.getByRole("button", { name: "Clear metadata cache" }))
    .not.toBeInTheDocument();

  await userEvent.click(page.getByRole("combobox", { name: "Metadata region" }));
  await userEvent.click(page.getByRole("option", { name: "United Kingdom" }));
  await expect.poll(() => sent).toEqual([{ values: { "metadata.region": "GB" } }]);
  await expect
    .element(page.getByText("certifications come from United Kingdom.", { exact: false }))
    .toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect
    .element(page.getByRole("textbox", { name: "Metadata cache keeps responses for" }))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Clear metadata cache" }));
  await expect.poll(() => cleared).toBe(1);
  await expect.element(page.getByText("Cleared the metadata cache.")).toBeVisible();
});
