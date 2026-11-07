import { expect } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksLibraryRow, WorksWorkView } from "../api/gen/model";
import { getGetWorkMockHandler, getSearchWorksMockHandler } from "../api/gen/query/works/works.msw";
import { paletteShortcut } from "../shell/palette-state";
import { addServer } from "./adding";
import { orchard, orchardId } from "./library";

export const label = "Search Works or run a command";
export const duneId = "6f1d3b5a-9c7e-4b2d-8f0a-1c3e5a7b9d2f";
export const dune: WorksWorkView = {
  ...orchard,
  id: duneId,
  key: "9000002",
  label: "Dune: Part Two",
  sortLabel: "Dune: Part Two",
};

export const rowOf = (work: WorksWorkView, year: number): WorksLibraryRow => ({
  id: work.id,
  contentType: work.contentType,
  key: work.key,
  label: work.label,
  sortLabel: work.sortLabel,
  addedAt: work.addedAt,
  refreshedAt: work.refreshedAt,
  goneUpstreamAt: null,
  monitored: "all",
  profileId: work.profile.id,
  rootId: null,
  posterUrl: null,
  availableOn: null,
  bytes: 0,
  files: 0,
  kept: 0,
  status: "missing",
  cells: { missing: 1 },
  failed: 0,
  items: 1,
  itemStatus: { missing: 1 },
  needsYou: 0,
  rating: null,
  values: { year },
});

export const orchards = (count: number): WorksLibraryRow[] =>
  Array.from({ length: count }, (_, index) => ({
    ...rowOf(orchard, 2021),
    id: `${orchardId.slice(0, -2)}${String(index).padStart(2, "0")}`,
    label: `Orchard ${index + 1}`,
    sortLabel: `Orchard ${index + 1}`,
  }));

/** The library's search over rows, matching a row whose label holds the query, and the catalog. */
export function library(rows: WorksLibraryRow[] = [rowOf(orchard, 2021), rowOf(dune, 2024)]) {
  const asked: string[] = [];
  const catalog = addServer();
  const handlers = [
    getSearchWorksMockHandler(({ request }) => {
      const url = new URL(request.url);
      const q = url.searchParams.get("q") ?? "";
      asked.push(`${q} ${url.searchParams.get("limit")}`);
      const found = rows.filter((row) => row.label.toLowerCase().includes(q.toLowerCase()));
      return { workIds: found.map((row) => row.id), rows: found.slice(0, 8) };
    }),
    getGetWorkMockHandler(dune),
    ...catalog.handlers,
  ];
  return { handlers, asked, searched: catalog.searched };
}

export const palette = () => page.getByRole("dialog", { name: label });
export const capsule = () => page.getByRole("banner").getByRole("button", { name: label });
export const group = (name: string) => palette().getByRole("group", { name });
export const optionTexts = (name: string) =>
  group(name)
    .getByRole("option")
    .all()
    .map((option) => option.element().textContent);

const [modifier] = paletteShortcut.keys.split("+");
export const pressShortcut = () => userEvent.keyboard(`{${modifier}>}k{/${modifier}}`);
export const openFromCapsule = async () => {
  await userEvent.click(capsule());
  await expect.element(palette().getByRole("combobox", { name: label })).toHaveFocus();
};

export const groupLabels = () =>
  palette()
    .getByRole("group")
    .all()
    .map((element) => element.element().firstElementChild?.textContent);

export const wait = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
