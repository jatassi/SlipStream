import { page } from "vitest/browser";

import type { GrabsManualSearch, GrabsPlanRow } from "../api/gen/model";
import { openManualSearch } from "../manual/sheet-state";
import { liveServer } from "./live-server";
import { filmPlan, release, runId } from "./manual";
import type { manualServer } from "./manual-server";
import { renderApp } from "./render-app";
import { targetHandlers } from "./targets";

export const workId = "0199f3a1-5c2e-7d4f-8a6b-1c3e5a7c9eaa";

export const routedWords = "Tracker routing sends it to qBittorrent on Den.";

export const rowOf = (
  key: string,
  words: string,
  n: number,
  choices: GrabsPlanRow["choices"] = [],
) => ({
  key,
  pick: {
    cells: [key],
    release: release(
      `${runId}.${n}`,
      `Shelf.S01E0${n}.1080p.WEB-DL-${words}`,
      `WEBDL-1080p · ${words}`,
    ),
    lifts: [],
    edition: null,
    routed: "qBittorrent on Den",
  },
  pinned: false,
  outcome: `It gets you 1080p for E0${n}.`,
  choices,
});

export const swapIn = release(
  `${runId}.9`,
  "Shelf.S01E02.1080p.WEBRip-RARBG",
  "WEBRip-1080p · RARBG",
);

/** A season's plan of two Grabs, E01 by NTb and E02 by FLUX, whose second row has one other choice. */
export const twoGrabs = (pins: readonly string[]): GrabsManualSearch => {
  const swapped = pins.includes("pin-e02-rarbg");
  const second = rowOf("cell-e02", "FLUX", 2, [
    { pin: "pin-e02-rarbg", release: swapIn, cells: ["cell-e02"], lifts: [], edition: null },
  ]);
  return {
    ...filmPlan,
    title: "Season 1 of The Long Shelf",
    queryable: false,
    plan: {
      headline: "2 Grabs get you everything.",
      held: [],
      rows: [
        rowOf("cell-e01", "NTb", 1),
        swapped
          ? {
              ...second,
              pinned: true,
              pick: { ...second.pick, release: swapIn },
              choices: [
                {
                  pin: "pin-e02-flux",
                  release: second.pick.release,
                  cells: ["cell-e02"],
                  lifts: [],
                  edition: null,
                },
              ],
            }
          : second,
      ],
    },
    pins: swapped ? [{ key: "cell-e02", pin: "pin-e02-rarbg" }] : [],
  };
};

export function openSheet(
  server: ReturnType<typeof manualServer>,
  extra: Parameters<typeof renderApp>[0] = {},
) {
  const live = liveServer();
  renderApp({
    path: "/",
    handlers: [
      ...(extra.handlers ?? []),
      ...live.handlers,
      ...server.handlers,
      ...targetHandlers(),
    ],
  });
  openManualSearch({ workId });
  return live;
}

export const sheetNamed = (name: string) => page.getByRole("dialog", { name });
