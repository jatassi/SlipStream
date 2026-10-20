import { expect, test } from "vitest";

import type { DownloadclientsTorrentRow } from "../api/gen/model";
import { homeId, listOf, seedboxId, torrent } from "../test/torrents";
import { etaWords, matches, rateWords, seedWords, stateLine, trackerLine } from "./model";
import { signalsOf, torrentsSentence } from "./summary";

const clock = { now: new Date("2026-10-20T12:00:00Z"), zone: "UTC" };
const place = { node: "seedbox", clock };
const live = { id: "", done: 0, wanted: 0, downBps: 0, upBps: 0, peers: 38, seeds: 4 };

test.each<
  [
    string,
    Partial<DownloadclientsTorrentRow> & Pick<DownloadclientsTorrentRow, "state">,
    boolean,
    string,
  ]
>([
  ["a download with its peers", { state: "downloading" }, true, "38 peers"],
  ["a download before any sample", { state: "downloading" }, false, ""],
  ["a seed with its peers", { state: "seeding" }, true, "38 peers"],
  ["third for a slot", { state: "queued", hold: "slot", queuePosition: 3 }, false, "3rd in line"],
  ["first for a slot", { state: "queued", hold: "slot", queuePosition: 1 }, false, "1st in line"],
  [
    "eleventh for a slot",
    { state: "queued", hold: "slot", queuePosition: 11 },
    false,
    "11th in line",
  ],
  ["waiting to check", { state: "queued", hold: "check_slot" }, false, "Waiting to check"],
  ["not yet reported", { state: "queued" }, false, "Waiting for seedbox"],
  ["a draining Node", { state: "queued", hold: "draining" }, false, "Waiting, seedbox is draining"],
  ["the Controller", { state: "queued", hold: "controller" }, false, "Waiting for the Controller"],
  [
    "the disk guard",
    { state: "queued", hold: "disk_guard" },
    false,
    "Waiting for disk space on seedbox",
  ],
  [
    "a resume inside a tracker's window",
    { state: "paused", hold: "window", until: "2026-10-20T14:32:00Z" },
    false,
    "Resumes at 14:32",
  ],
  [
    "a full disk",
    { state: "paused", hold: "disk_full" },
    false,
    "Disk full, carries on once there is space",
  ],
  ["an Admin's pause", { state: "paused", hold: "wanted" }, false, ""],
  [
    "a pause an offline Node has yet to take",
    { state: "paused", pending: "pause", lastKnown: "2026-10-20T11:52:00Z" },
    false,
    "Pauses when seedbox is back",
  ],
  [
    "a resume an offline Node has yet to take",
    { state: "paused", hold: "wanted", pending: "resume", lastKnown: "2026-10-20T11:52:00Z" },
    false,
    "Resumes when seedbox is back",
  ],
  ["a pause on its way", { state: "paused", pending: "pause" }, false, "Pausing"],
  [
    "a recheck that resumes",
    { state: "checking", until: "2026-10-20T14:32:00Z" },
    false,
    "Rechecks, then resumes at 14:32",
  ],
  ["a check", { state: "checking" }, false, "Checking its files"],
  ["a stall", { state: "stalled" }, false, "No data is arriving"],
  ["a removal", { state: "removing" }, false, "Waiting for seedbox to confirm"],
  ["an engine failure", { state: "failed", error: "file too short" }, false, "file too short"],
  [
    "an unregistered torrent",
    { state: "tracker-error", tracker: { host: "beacon.example", status: "unregistered" } },
    false,
    "Unregistered by beacon.example",
  ],
  [
    "an offline Node's download",
    { state: "downloading", lastKnown: "2026-10-20T11:52:00Z" },
    true,
    "Last known at 11:52",
  ],
])("the line under %s", (_, row, sampled, line) => {
  expect(stateLine(torrent(row), sampled ? live : undefined, place)).toBe(line);
});

test.each([
  [undefined, { host: "No tracker", status: "" }],
  [
    { host: "beacon.example", status: "working" } as const,
    { host: "beacon.example", status: "Working" },
  ],
  [
    { host: "", status: "not_contacted" } as const,
    { host: "Its tracker", status: "Not contacted yet" },
  ],
  [
    {
      host: "beacon.example",
      status: "error",
      message: "Tracker is down",
      retryAt: "2026-10-20T15:10:00Z",
    } as const,
    { host: "beacon.example", status: "Said: Tracker is down, trying again at 15:10" },
  ],
])("a tracker cell reads who and how its announce went", (cell, words) => {
  expect(trackerLine(cell, clock)).toEqual(words);
});

test("rates, time left and seed words", () => {
  expect(rateWords(0)).toBe("—");
  expect(rateWords(4.1 * 1024 * 1024)).toBe("4.1 MB/s");
  expect(etaWords({ ...live, done: 0, wanted: 600 * 1024 * 1024, downBps: 1024 * 1024 })).toBe(
    "10 min left",
  );
  expect(etaWords({ ...live, downBps: 0, wanted: 10 })).toBe("");
  expect(seedWords(torrent({ state: "seeding", ratio: 1.4249, seedingSeconds: 6 * 86_400 }))).toBe(
    "Ratio 1.42 · 6 days",
  );
});

test("a filter matches the tab, the Node, the Content Type and the Work or release name", () => {
  const row = torrent({
    state: "downloading",
    tabs: ["downloading", "all"],
    nodeId: seedboxId,
    name: "Orchard.Harvest.2019.720p",
  });
  const all = { tab: "all", q: "" } as const;
  expect(matches(row, all, undefined)).toBe(true);
  expect(matches(row, { ...all, tab: "seeding" }, undefined)).toBe(false);
  expect(matches(row, { ...all, node: homeId }, undefined)).toBe(false);
  expect(matches(row, { ...all, node: seedboxId }, undefined)).toBe(true);
  expect(matches(row, { ...all, type: "zines" }, undefined)).toBe(false);
  expect(matches(row, { ...all, q: "harvest.2019" }, undefined)).toBe(true);
  expect(matches(row, { ...all, q: "quiet" }, "The Quiet Orchard")).toBe(true);
  expect(matches(row, { ...all, q: "quiet" }, undefined)).toBe(false);
});

const nodeName = (id: string) => (id === seedboxId ? "seedbox" : "home");
const down = (host: string, at: string) =>
  torrent({
    state: "tracker-error",
    severity: "warning",
    addedAt: at,
    tracker: { host, status: "error", message: "Tracker is down" },
  });

test("one tracker down for many seeds is one warning holding them all", () => {
  const rows = [
    down("beacon.example", "2026-10-12T09:05:00Z"),
    down("beacon.example", "2026-10-12T09:01:00Z"),
    down("beacon.example", "2026-10-12T09:03:00Z"),
  ];
  const [only, ...rest] = signalsOf(rows, nodeName, clock);
  expect(rest).toEqual([]);
  expect(only?.signal).toBe("warn");
  expect(only?.verdict).toBe("beacon.example said: Tracker is down.");
  expect(only?.rows).toHaveLength(3);
});

test("failures lead the warnings, each cause with its oldest torrent first", () => {
  const stalled = torrent({
    state: "stalled",
    severity: "warning",
    addedAt: "2026-10-12T08:00:00Z",
  });
  const failed = torrent({
    state: "failed",
    severity: "failure",
    error: "file too short",
    addedAt: "2026-10-12T10:00:00Z",
  });
  const unregistered = torrent({
    state: "tracker-error",
    severity: "failure",
    addedAt: "2026-10-12T09:00:00Z",
    tracker: { host: "beacon.example", status: "unregistered" },
  });
  const guarded = torrent({
    state: "queued",
    hold: "disk_guard",
    severity: "warning",
    nodeId: seedboxId,
    addedAt: "2026-10-12T07:00:00Z",
  });
  const quiet = torrent({ state: "seeding" });
  const signals = signalsOf([stalled, failed, unregistered, guarded, quiet], nodeName, clock);
  expect(signals.map(({ chip, verdict }) => [chip, verdict])).toEqual([
    ["Unregistered", "Unregistered by beacon.example: the tracker no longer knows the torrent."],
    ["Failed", "The torrent client stopped it: file too short."],
    ["Waiting for disk", "New downloads on seedbox wait for disk space."],
    ["Stalled", "No data has arrived for a while, so it no longer holds a download slot."],
  ]);
});

test("the sentence counts what downloads and seeds, and links each problem to its first signal", () => {
  const rows = [
    torrent({ state: "downloading" }),
    torrent({ state: "seeding" }),
    torrent({ state: "seeding" }),
    torrent({ state: "failed", severity: "failure" }),
    down("beacon.example", "2026-10-12T09:05:00Z"),
    down("beacon.example", "2026-10-12T09:06:00Z"),
  ];
  const signals = signalsOf(rows, nodeName, clock);
  expect(torrentsSentence(listOf(rows).counts, signals)).toEqual([
    { kind: "live", text: "1 downloading" },
    { kind: "plain", text: "2 seeding" },
    { kind: "failure", text: "1 failure", href: "#torrent-signal-0" },
    { kind: "warning", text: "2 warnings", href: "#torrent-signal-1" },
  ]);
});

test("a quiet sentence says nothing is downloading", () => {
  const rows = [torrent({ state: "seeding" })];
  expect(torrentsSentence(listOf(rows).counts, [])).toEqual([
    { kind: "plain", text: "Nothing downloading" },
    { kind: "plain", text: "1 seeding" },
  ]);
});
