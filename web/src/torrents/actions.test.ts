import { expect, test } from "vitest";

import type {
  ActInputBodyAction,
  DownloadclientsRefusal,
  DownloadclientsTorrentRow,
} from "../api/gen/model";
import { torrent } from "../test/torrents";
import { menuFor, outcomeWords, wantedOf } from "./actions";

type Shape = Partial<DownloadclientsTorrentRow> & Pick<DownloadclientsTorrentRow, "state">;

const moving = {
  unitId: "u",
  batchId: "b",
  phase: "place",
  toNodeId: "n",
  toNodeName: "den",
  words: "",
} as const;
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
    "a finished torrent moves to another Download Client",
    { state: "seeding", done: 10, size: 10 },
    reject,
    ["pause", "recheck", "move", "remove"],
  ],
  [
    "a moving torrent offers no second Move",
    { state: "seeding", done: 10, size: 10, move: moving },
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
    "a download a Grab holds can be marked failed",
    { state: "downloading", held: true },
    reject,
    ["pause", "recheck", "mark-failed", "remove"],
  ],
  [
    "a stalled download no Grab holds can't be marked failed",
    { state: "stalled" },
    reject,
    ["pause", "recheck", "remove"],
  ],
  [
    "a seed a Grab holds can be marked failed",
    { state: "seeding", held: true },
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
  ["a failed download offers only remove", { state: "failed" }, accept, ["remove"]],
  [
    "a seed that failed offers retry and remove",
    { state: "failed", retryable: true },
    accept,
    ["retry", "remove"],
  ],
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
const context = (action: ActInputBodyAction) => ({
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

test("several moving torrents are counted once, wherever they move", () => {
  expect(
    outcomeWords(
      [
        { id: "a", done: false, refusal: { code: "moving", node: "den" } },
        { id: "b", done: false, refusal: { code: "moving", client: "qBittorrent" } },
        { id: "c", done: false, refusal: { code: "moving", node: "den" } },
      ],
      [orchard, harvest, attic],
      context("pause"),
    ).refused,
  ).toEqual([
    "3 torrents are moving to another Download Client. Cancel their Moves in Activity, then try again.",
  ]);
});

test.each<[string, DownloadclientsRefusal, ActInputBodyAction, string]>([
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
    "A tracker of The Quiet Orchard won't accept another announce until 2:32 PM.",
  ],
  [
    "an offline Node says it waits",
    { code: "offline" },
    "recheck",
    "The Node with The Quiet Orchard is offline. Try again once it reconnects.",
  ],
  [
    "an external client that isn't responding names it and its Node",
    { code: "client_away", client: "Transmission", node: "Media server" },
    "announce",
    "Transmission on Media server isn't responding. Try again once it responds.",
  ],
  [
    "an external client that is off says to turn it on",
    { code: "client_off", client: "Transmission", node: "Media server" },
    "recheck",
    "SlipStream isn't checking Transmission while it's turned off. Turn it on, then try again.",
  ],
  [
    "a torrent its Node no longer holds",
    { code: "not_found" },
    "announce",
    "SlipStream no longer tracks The Quiet Orchard.",
  ],
  [
    "a paused torrent has nothing to announce",
    { code: "not_running" },
    "announce",
    "The Quiet Orchard isn't running, so it has nothing to announce.",
  ],
  [
    "a torrent moving to another Node names the Node",
    { code: "moving", node: "den" },
    "resume",
    "The Quiet Orchard is moving to den. Cancel its Move in Activity, then try again.",
  ],
  [
    "a torrent moving to another client on its Node names the client",
    { code: "moving", client: "qBittorrent" },
    "pause",
    "The Quiet Orchard is moving to qBittorrent. Cancel its Move in Activity, then try again.",
  ],
  [
    "a torrent moving to another Node's client names both",
    { code: "moving", client: "Transmission", node: "den" },
    "remove",
    "The Quiet Orchard is moving to Transmission on den. Cancel its Move in Activity, then try again.",
  ],
  [
    "a failed torrent can only go",
    { code: "failed" },
    "recheck",
    "The Quiet Orchard failed in the torrent client and can't recheck. Remove it instead.",
  ],
  [
    "a torrent no Grab downloads stays",
    { code: "not_held" },
    "mark-failed",
    "No Grab is downloading The Quiet Orchard, so nothing was marked failed.",
  ],
  [
    "a torrent whose Import waits for the Admin points to its Pending import",
    { code: "use_not_this_release" },
    "mark-failed",
    "The Quiet Orchard is waiting in Pending imports. To reject this release, choose Not this Release on its Pending import.",
  ],
  [
    "nothing is marked failed while Reconciling",
    { code: "reconciling" },
    "mark-failed",
    "SlipStream is still matching a restored Backup to what each Node has, so nothing was marked failed. Try again once that finishes.",
  ],
])("%s", (_, refusal, action, sentence) => {
  expect(outcomeWords([{ id: "a", done: false, refusal }], [orchard], context(action))).toEqual({
    done: undefined,
    refused: [sentence],
  });
});

test("a refused Retry says which torrents a Retry checks", () => {
  expect(
    outcomeWords(
      [
        { id: "a", done: true },
        { id: "b", done: false, refusal: { code: "not_retryable" } },
      ],
      [orchard, harvest],
      context("retry"),
    ),
  ).toEqual({
    done: "Checking the files of The Quiet Orchard.",
    refused: [
      "Orchard Harvest can't be retried. Retry checks only a torrent that failed after it finished downloading.",
    ],
  });
});
