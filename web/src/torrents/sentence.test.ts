import { expect, test } from "vitest";

import type { DownloadclientsTorrentRow } from "../api/gen/model";
import { torrent } from "../test/torrents";
import { inspectorSentence } from "./sentence";

type Shape = Partial<DownloadclientsTorrentRow> & Pick<DownloadclientsTorrentRow, "state">;

const place = { node: "seedbox", clock: { now: new Date("2026-10-20T12:00:00Z"), zone: "UTC" } };
const sample = {
  id: "",
  done: 500_000_000,
  wanted: 1_000_000_000,
  downBps: 4.1 * 1024 * 1024,
  upBps: 0,
  peers: 38,
  seeds: 4,
};

test.each<[string, Shape, typeof sample | undefined, string]>([
  [
    "a download says its peers, time left and speed",
    { state: "downloading" },
    sample,
    "Downloading from 38 peers, 2 min left at 4.1 MB/s.",
  ],
  ["a download before its first sample", { state: "downloading" }, undefined, "Downloading."],
  [
    "a seed says its peers and what it gave back",
    { state: "seeding", ratio: 1.42, seedingSeconds: 6 * 86_400 },
    { ...sample, peers: 3 },
    "Seeding to 3 peers. Ratio 1.42 after 6 days.",
  ],
  [
    "a seed just finished",
    { state: "seeding", seedingSeconds: 10 },
    undefined,
    "Seeding. Ratio 0.00 so far.",
  ],
  [
    "a resume inside a tracker's window says when it starts",
    { state: "paused", hold: "window", until: "2026-10-20T14:32:00Z" },
    undefined,
    "Waiting for its tracker. It resumes at 14:32, when every tracker accepts an announce again.",
  ],
  [
    "the Admin's pause stays",
    { state: "paused", hold: "wanted" },
    undefined,
    "Paused. It stays paused until you resume it.",
  ],
  [
    "a pause its Node hasn't taken",
    { state: "downloading", pending: "pause" },
    sample,
    "Pausing. It stops downloading and seeding once seedbox takes the pause.",
  ],
  [
    "a pause on an offline Node waits for it",
    { state: "downloading", pending: "pause", lastKnown: "2026-10-20T11:00:00Z" },
    undefined,
    "It pauses when seedbox is back online.",
  ],
  [
    "an offline Node's last word",
    { state: "seeding", lastKnown: "2026-10-20T11:00:00Z" },
    undefined,
    "seedbox is offline. When it last reported, at 11:00, this was seeding.",
  ],
  [
    "third in line",
    { state: "queued", hold: "slot", queuePosition: 3 },
    undefined,
    "Queued, 3rd in line on seedbox. It starts once a download slot is free.",
  ],
  [
    "a full disk",
    { state: "paused", hold: "disk_full" },
    undefined,
    "Paused, because seedbox ran out of disk space. It carries on once there is space.",
  ],
  [
    "a recheck that resumes after",
    { state: "checking", until: "2026-10-20T14:32:00Z" },
    undefined,
    "Checking its files. It resumes at 14:32.",
  ],
  [
    "a stall offers the way out",
    { state: "stalled" },
    undefined,
    "Stalled. No data has arrived for a while, so it gave up its download slot. Mark it failed to give up on this release.",
  ],
  [
    "the engine's words for a failure",
    { state: "failed", error: "no space left on device" },
    undefined,
    "The torrent client stopped it, saying no space left on device. Remove it, then search for another release.",
  ],
  [
    "an unregistered torrent",
    { state: "tracker-error", tracker: { host: "beacon.example", status: "unregistered" } },
    undefined,
    "beacon.example no longer knows this torrent, so no peers come from it. Remove it, then search for another release.",
  ],
  [
    "a tracker's own words and its retry",
    {
      state: "tracker-error",
      tracker: {
        host: "beacon.example",
        status: "error",
        message: "too many requests",
        retryAt: "2026-10-20T12:30:00Z",
      },
    },
    undefined,
    'beacon.example said "too many requests", so no peers come from it. It tries again at 12:30.',
  ],
  [
    "a remove waits for its Node",
    { state: "removing" },
    undefined,
    "Removing. It is gone once seedbox confirms.",
  ],
])("%s", (_, shape, live, sentence) => {
  expect(inspectorSentence(torrent(shape), live, place)).toBe(sentence);
});
