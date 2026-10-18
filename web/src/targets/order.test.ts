import { expect, test } from "vitest";

import { type Order, accept, drop, keepCutoff, layoutOf, orderOf, placeCutoff } from "./order";

const ids = (rows: Order) =>
  rows.map((row) => (row.kind === "marker" ? "|" : `${row.tiedAbove ? "=" : ""}${row.id}`));

const preset = [
  ["bluray-2160p", "remux-2160p"],
  ["bluray-1080p", "remux-1080p"],
  ["hdtv-1080p", "webrip-1080p", "webdl-1080p"],
  ["hdtv-720p"],
  ["sdtv"],
  ["cam"],
];

const rows = orderOf({
  tiers: [["remux-1080p"], ["webdl-1080p", "webrip-1080p"], ["hdtv-720p"], ["cam"]],
  cutoff: 0,
});

test("a saved Target's rows hold its tiers with the cutoff below its cutoff tier, CAM apart", () => {
  expect(ids(rows)).toEqual(["remux-1080p", "|", "webdl-1080p", "=webrip-1080p", "hdtv-720p"]);
  expect(layoutOf(rows, true)).toEqual({
    tiers: [["remux-1080p"], ["webdl-1080p", "webrip-1080p"], ["hdtv-720p"], ["cam"]],
    cutoff: 0,
  });
});

test("an accepted quality takes the preset's place: tied, above the first it outranks, or last", () => {
  expect(ids(accept(rows, "bluray-1080p", preset))).toEqual([
    "remux-1080p",
    "=bluray-1080p",
    "|",
    "webdl-1080p",
    "=webrip-1080p",
    "hdtv-720p",
  ]);
  expect(ids(accept(rows, "remux-2160p", preset))).toEqual([
    "remux-2160p",
    "remux-1080p",
    "|",
    "webdl-1080p",
    "=webrip-1080p",
    "hdtv-720p",
  ]);
  expect(ids(accept(rows, "sdtv", preset))).toEqual([
    "remux-1080p",
    "|",
    "webdl-1080p",
    "=webrip-1080p",
    "hdtv-720p",
    "sdtv",
  ]);
  expect(ids(accept([...orderOf({ tiers: [], cutoff: 0 })], "sdtv", preset))).toEqual([
    "sdtv",
    "|",
  ]);
});

test("a quality accepted just below the cutoff stays below it, so upgrades stop where they did", () => {
  const top = orderOf({ tiers: [["remux-2160p"], ["hdtv-720p"]], cutoff: 0 });
  const kept = accept(top, "webdl-1080p", preset);
  expect(ids(kept)).toEqual(["remux-2160p", "|", "webdl-1080p", "hdtv-720p"]);
  expect(layoutOf(kept, false).cutoff).toBe(0);
});

test("a dropped quality leaves its tier together, and the cutoff keeps a tier above it", () => {
  expect(ids(drop(rows, "webdl-1080p"))).toEqual(["remux-1080p", "|", "webrip-1080p", "hdtv-720p"]);
  expect(ids(drop(rows, "remux-1080p"))).toEqual([
    "webdl-1080p",
    "=webrip-1080p",
    "|",
    "hdtv-720p",
  ]);
  expect(ids(drop(drop(drop(rows, "remux-1080p"), "webdl-1080p"), "webrip-1080p"))).toEqual([
    "hdtv-720p",
    "|",
  ]);
  expect(ids(keepCutoff(placeCutoff(rows, "bottom")))).toEqual([
    "remux-1080p",
    "webdl-1080p",
    "=webrip-1080p",
    "hdtv-720p",
    "|",
  ]);
  expect(
    layoutOf(
      drop(drop(drop(drop(rows, "remux-1080p"), "webdl-1080p"), "webrip-1080p"), "hdtv-720p"),
      false,
    ),
  ).toEqual({
    tiers: [],
    cutoff: 0,
  });
});
