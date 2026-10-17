import { expect, test } from "vitest";

import { type Order, type QualityKey, keepCutoff, layoutOf, orderOf, placeCutoff } from "./order";

const ids = (rows: Order) =>
  rows.map((row) => (row.kind === "marker" ? "|" : `${row.tiedAbove ? "=" : ""}${row.id}`));

const rows = orderOf(
  { tiers: [["remux-1080p"], ["webdl-1080p", "webrip-1080p"], ["hdtv-720p"]], cutoff: 0 },
  ["sdtv", "remux-1080p", "webdl-1080p", "webrip-1080p", "hdtv-720p", "cam"],
);

const all = new Set<QualityKey>(["remux-1080p", "webdl-1080p", "webrip-1080p", "hdtv-720p"]);

test("a saved Target's rows hold its tiers, the cutoff below its cutoff tier, then the rest", () => {
  expect(ids(rows)).toEqual([
    "remux-1080p",
    "|",
    "webdl-1080p",
    "=webrip-1080p",
    "hdtv-720p",
    "sdtv",
  ]);
  expect(layoutOf(rows, all)).toEqual({
    tiers: [["remux-1080p"], ["webdl-1080p", "webrip-1080p"], ["hdtv-720p"]],
    cutoff: 0,
  });
});

test("an unaccepted quality drops out of its tier, and the cutoff counts only accepted tiers", () => {
  const accepted = new Set<QualityKey>(["webrip-1080p", "hdtv-720p", "cam"]);
  const kept = keepCutoff(rows, accepted);
  expect(ids(kept)).toEqual([
    "remux-1080p",
    "webdl-1080p",
    "=webrip-1080p",
    "|",
    "hdtv-720p",
    "sdtv",
  ]);
  expect(layoutOf(kept, accepted)).toEqual({
    tiers: [["webrip-1080p"], ["hdtv-720p"], ["cam"]],
    cutoff: 0,
  });
});

test("the cutoff goes below the first or the last tier holding an accepted quality", () => {
  expect(ids(placeCutoff(rows, all, "bottom"))).toEqual([
    "remux-1080p",
    "webdl-1080p",
    "=webrip-1080p",
    "hdtv-720p",
    "|",
    "sdtv",
  ]);
  expect(ids(placeCutoff(rows, new Set(["webrip-1080p"]), "top"))).toEqual([
    "remux-1080p",
    "webdl-1080p",
    "=webrip-1080p",
    "|",
    "hdtv-720p",
    "sdtv",
  ]);
  expect(layoutOf(placeCutoff(rows, new Set(), "top"), new Set())).toEqual({
    tiers: [],
    cutoff: 0,
  });
});
