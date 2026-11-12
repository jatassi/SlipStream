import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { GrabsOffer } from "../api/gen/model";
import { filmPlan, lowScore } from "../test/manual";
import { filmReleases, manualServer } from "../test/manual-server";
import { openSheet, sheetNamed } from "../test/manual-sheet";
import { starter } from "../test/targets";

function openAllReleases(releases: readonly GrabsOffer[] = filmReleases) {
  const listed: number[] = [];
  const server = manualServer({ plan: () => filmPlan, releases });
  openSheet(server, {
    handlers: [
      http.get("*/api/v1/works/:workId/manual-search/releases", () => {
        listed.push(1);
        return HttpResponse.json({ releases });
      }),
    ],
  });
  const sheet = sheetNamed("The Quiet Orchard (2021)");
  const all = sheet.getByRole("list", { name: "All Releases" });
  const order = () =>
    all
      .getByRole("listitem")
      .elements()
      .map((item) => item.firstElementChild?.textContent);
  return { server, listed, sheet, all, order };
}

test("All Releases loads once opened and sorts by the chosen column", async () => {
  const { listed, sheet, order } = openAllReleases();
  await expect.element(sheet.getByText(filmPlan.plan.headline)).toBeVisible();
  expect(listed).toHaveLength(0);

  await userEvent.click(sheet.getByRole("button", { name: "All Releases (9)" }));
  await expect
    .poll(order)
    .toEqual([
      "WEBDL-1080p · DD+ 5.1 · FLUX · 4.0 GB",
      "WEBRip-1080p · RARBG · 2.1 GB",
      "Bluray-1080p · SPARKS · 9.8 GB",
      "HDTV-720p · KILLERS · 1.1 GB",
    ]);
  expect(listed).toHaveLength(1);

  // KILLERS's fixture lists 4 GiB, tying FLUX, so rank breaks the tie.
  await userEvent.click(sheet.getByRole("combobox", { name: "Sort by" }));
  await userEvent.click(page.getByRole("option", { name: "Size" }));
  await expect
    .poll(order)
    .toEqual([
      "Bluray-1080p · SPARKS · 9.8 GB",
      "WEBDL-1080p · DD+ 5.1 · FLUX · 4.0 GB",
      "HDTV-720p · KILLERS · 1.1 GB",
      "WEBRip-1080p · RARBG · 2.1 GB",
    ]);
});

test("All Releases grabs one Release on its own with what it gets past, and says why another can't be grabbed", async () => {
  const { server, sheet, all } = openAllReleases();
  await userEvent.click(sheet.getByRole("button", { name: "All Releases (9)" }));
  const killers = all.getByRole("listitem").nth(3);
  await expect.element(killers.getByText("No Target here takes HDTV-720p.")).toBeVisible();
  await expect.element(killers.getByRole("button", { name: "Grab" })).not.toBeInTheDocument();

  const rarbg = all.getByRole("listitem").nth(1);
  await userEvent.click(rarbg.getByRole("button", { name: "Grab" }));
  await expect.element(rarbg.getByText("Grabbed.")).toBeVisible();
  expect(server.grabbed).toEqual([
    [
      {
        releaseId: lowScore.id,
        cells: ["cell-orchard-1080p"],
        lifts: [{ code: "score_below_minimum", targetId: starter.id }],
      },
    ],
  ]);
});

test("a Release's Details list its facts under their labels, the seeders as a number", async () => {
  const { sheet, all } = openAllReleases();
  await userEvent.click(sheet.getByRole("button", { name: "All Releases (9)" }));
  const first = all.getByRole("listitem").first();
  await userEvent.click(first.getByRole("button", { name: "Details" }));
  await expect.element(first).toMatchTextContent(/Seeders40Open in tester/u);
});

test("a Release that isn't better than the file grabs for its Target, forcing it in", async () => {
  const notBetter: GrabsOffer = {
    ...filmReleases[1],
    grab: {
      ...filmReleases[1].grab,
      lifts: [
        {
          code: "not_an_upgrade",
          targetId: starter.id,
          soft: true,
          words:
            "It isn't better than The Quiet Orchard (2021).mkv, the file at 1080p. Grabbing it replaces that file.",
        },
      ],
    },
  };
  const { server, sheet, all } = openAllReleases([notBetter]);
  await userEvent.click(sheet.getByRole("button", { name: "All Releases (9)" }));
  const row = all.getByRole("listitem").first();
  await expect
    .element(
      row.getByText(
        "It isn't better than The Quiet Orchard (2021).mkv, the file at 1080p. Grabbing it replaces that file.",
      ),
    )
    .toBeVisible();
  await userEvent.click(row.getByRole("button", { name: "Grab for 1080p" }));
  await expect.element(row.getByText("Grabbed.")).toBeVisible();
  expect(server.grabbed).toEqual([
    [
      {
        releaseId: lowScore.id,
        cells: ["cell-orchard-1080p"],
        lifts: [{ code: "not_an_upgrade", targetId: starter.id }],
      },
    ],
  ]);
});
