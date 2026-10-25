import { expect, test } from "vitest";

import type { DownloadclientsRefusal, DownloadclientsTorrentRow } from "../api/gen/model";
import { torrent } from "../test/torrents";
import { type TorrentAction, menuFor, outcomeWords, wantedOf } from "./actions";

type Shape = Partial<DownloadclientsTorrentRow> & Pick<DownloadclientsTorrentRow, "state">;

const now = new Date("2026-10-20T12:00:00Z");
const past = "2026-10-20T11:59:00Z";
const later = "2026-10-20T12:30:00Z";
const accept = true;
const reject = false;

test.each<[string, Shape, boolean, string[]]>([
  [
    "a download pauses, rechecks and removes",
    { state: "downloading" },
    reject,
    ["pause", "recheck", "remove"],
  ],
  [
    "a download every tracker accepts now also announces",
    { state: "downloading" },
    accept,
    ["pause", "announce", "recheck", "remove"],
  ],
  [
    "inside a tracker's window it doesn't announce",
    { state: "seeding" },
    reject,
    ["pause", "recheck", "remove"],
  ],
  [
    "a paused torrent resumes",
    { state: "paused", hold: "wanted" },
    accept,
    ["resume", "recheck", "remove"],
  ],
  [
    "a resume waiting out a window still offers pause",
    { state: "paused", hold: "window", until: later },
    accept,
    ["pause", "recheck", "remove"],
  ],
  [
    "a pause its Node hasn't taken yet offers resume",
    { state: "downloading", pending: "pause" },
    accept,
    ["resume", "recheck", "remove"],
  ],
  [
    "a resume its Node hasn't taken yet offers pause and no announce",
    { state: "paused", hold: "wanted", pending: "resume" },
    accept,
    ["pause", "recheck", "remove"],
  ],
  [
    "a stalled download can be marked failed",
    { state: "stalled" },
    reject,
    ["pause", "recheck", "mark-failed", "remove"],
  ],
  ["a check in progress isn't rechecked", { state: "checking" }, reject, ["pause", "remove"]],
  [
    "on an offline Node only pause and remove apply",
    { state: "seeding", lastKnown: past },
    accept,
    ["pause", "remove"],
  ],
  ["a failed torrent offers only remove", { state: "failed" }, accept, ["remove"]],
  ["a removing torrent offers nothing", { state: "removing" }, accept, []],
])("%s", (_, shape, trackersAccept, offered) => {
  expect(menuFor(torrent(shape), trackersAccept)).toEqual(offered);
});

test.each<[Shape, "started" | "paused"]>([
  [{ state: "downloading" }, "started"],
  [{ state: "paused", hold: "wanted" }, "paused"],
  [{ state: "paused", hold: "window" }, "started"],
  [{ state: "downloading", pending: "pause" }, "paused"],
  [{ state: "paused", hold: "wanted", pending: "resume" }, "started"],
])("wanted of %o is %s", (shape, wanted) => {
  expect(wantedOf(torrent(shape))).toBe(wanted);
});

const orchard = { id: "a", name: "The Quiet Orchard" };
const harvest = { id: "b", name: "Orchard Harvest" };
const attic = { id: "c", name: "Attic Light" };
const batman = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";
const context = (action: TorrentAction) => ({
  action,
  work: (id: string) => (id === batman ? "The Batman" : undefined),
  clock: { now, zone: "UTC" },
});

test("one done names the torrent", () => {
  expect(outcomeWords([{ id: "a", done: true }], [orchard], context("pause"))).toEqual({
    done: "Pausing The Quiet Orchard.",
    refused: [],
  });
});

test("several done are counted", () => {
  expect(
    outcomeWords(
      [
        { id: "a", done: true },
        { id: "b", done: true },
      ],
      [orchard, harvest],
      context("remove"),
    ).done,
  ).toBe("Removing 2 torrents.");
});

test("a bulk action names what it did and groups what it refused alike", () => {
  expect(
    outcomeWords(
      [
        { id: "a", done: true },
        { id: "b", done: false, refusal: { code: "removed" } },
        { id: "c", done: false, refusal: { code: "removed" } },
      ],
      [orchard, harvest, attic],
      context("resume"),
    ),
  ).toEqual({
    done: "Resuming The Quiet Orchard.",
    refused: ["2 torrents are already being removed."],
  });
});

test.each<[string, DownloadclientsRefusal, TorrentAction, string]>([
  [
    "an import names the Work it is finishing",
    { code: "importing", workId: batman },
    "remove",
    "SlipStream is finishing the import of The Batman. Try again once it's done.",
  ],
  [
    "a tracker's window says when it accepts",
    { code: "window", acceptsAt: "2026-10-20T14:32:00Z" },
    "announce",
    "A tracker of The Quiet Orchard would reject an announce until 14:32.",
  ],
  [
    "an offline Node says it waits",
    { code: "offline" },
    "recheck",
    "The Node holding The Quiet Orchard is offline. Try again once it reconnects.",
  ],
  [
    "a torrent its Node no longer holds",
    { code: "not_found" },
    "announce",
    "The Quiet Orchard is no longer there.",
  ],
  [
    "a paused torrent has nothing to announce",
    { code: "not_running" },
    "announce",
    "The Quiet Orchard isn't running, so it has nothing to announce.",
  ],
  [
    "a failed torrent can only go",
    { code: "failed" },
    "recheck",
    "The Quiet Orchard failed in the torrent client and can't recheck. Remove it instead.",
  ],
  [
    "a stall that cleared stays",
    { code: "not_stalled" },
    "mark-failed",
    "The Quiet Orchard is moving again, so nothing was marked failed.",
  ],
])("%s", (_, refusal, action, sentence) => {
  expect(outcomeWords([{ id: "a", done: false, refusal }], [orchard], context(action))).toEqual({
    done: undefined,
    refused: [sentence],
  });
});
