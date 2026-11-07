import { expect, test } from "vitest";

import type { DownloadclientsTorrentRow } from "../api/gen/model";
import { torrent } from "../test/torrents";
import { attachedSentence, inspectorSentence } from "./sentence";

type Shape = Partial<DownloadclientsTorrentRow> & Pick<DownloadclientsTorrentRow, "state">;

const transmission = {
  clientId: "7d3f9b1e-5a2c-4e8f-b6d0-1c3e5a7f9b2d",
  clientName: "Transmission",
  clientKind: "transmission",
} as const;

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
    "Waiting for an announce. It resumes at 14:32, when every announce URL accepts one again.",
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
    "Pausing. It stops downloading and seeding once seedbox applies the pause.",
  ],
  [
    "a pause on an offline Node waits for it",
    { state: "downloading", pending: "pause", lastKnown: "2026-10-20T11:00:00Z", lastKnownCause: "node" },
    undefined,
    "It pauses when seedbox reconnects.",
  ],
  [
    "an offline Node's last word",
    { state: "seeding", lastKnown: "2026-10-20T11:00:00Z", lastKnownCause: "node" },
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
    "Paused because seedbox ran out of disk space. It resumes once there is space.",
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
    "Stalled. No data has arrived for a while, so it no longer holds a download slot. To stop using this Release, mark it failed.",
  ],
  [
    "the engine's words for a failure",
    { state: "failed", error: "no space left on device" },
    undefined,
    'The torrent client stopped it with the error "no space left on device". Remove it, then search for another Release.',
  ],
  [
    "an unregistered torrent",
    { state: "tracker-error", tracker: { host: "beacon.example", status: "unregistered" } },
    undefined,
    "beacon.example reports that this torrent is no longer registered, so no peers come from it. Remove it, then search for another Release.",
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
    'beacon.example reported "too many requests", so no peers come from it. The client retries at 12:30.',
  ],
  [
    "a remove waits for its Node",
    { state: "removing" },
    undefined,
    "Removing. Waiting for seedbox to confirm.",
  ],
  [
    "a client that isn't responding names itself and its Node",
    { ...transmission, state: "downloading", lastKnown: "2026-10-20T11:00:00Z", lastKnownCause: "client" },
    undefined,
    "Transmission on seedbox isn't responding. When it last responded, at 11:00, this was downloading.",
  ],
  [
    "an off client isn't checked",
    { ...transmission, state: "seeding", lastKnown: "2026-10-20T11:00:00Z", lastKnownCause: "off" },
    undefined,
    "SlipStream isn't checking Transmission while it's turned off. When it last responded, at 11:00, this was seeding.",
  ],
  [
    "a pause waits for the client to respond",
    {
      ...transmission,
      state: "downloading",
      pending: "pause",
      lastKnown: "2026-10-20T11:00:00Z",
      lastKnownCause: "client",
    },
    undefined,
    "It pauses when Transmission responds again.",
  ],
  [
    "a remove waits for the client to respond",
    { ...transmission, state: "removing", lastKnown: "2026-10-20T11:00:00Z", lastKnownCause: "client" },
    undefined,
    "It's removed once Transmission responds again.",
  ],
  [
    "a stop at the seeding limit is no pause of the Admin's",
    { ...transmission, state: "paused", stoppedAtLimit: true },
    undefined,
    "Transmission stopped it at its seeding limit.",
  ],
  [
    "a finished torrent waits out the Import delay",
    { ...transmission, state: "seeding", importAt: "2026-10-20T14:32:00Z" },
    undefined,
    "Finished. SlipStream imports it at 14:32, when the Import delay for Transmission ends.",
  ],
])("%s", (_, shape, live, sentence) => {
  expect(inspectorSentence(torrent(shape), live, place)).toBe(sentence);
});

test("an attached torrent says a remove leaves it in its client, and an added one says nothing", () => {
  expect(attachedSentence(torrent({ ...transmission, state: "seeding", attached: true }))).toBe(
    "This torrent was in Transmission before SlipStream attached to it. Removing it here stops SlipStream tracking it and leaves it in Transmission.",
  );
  expect(attachedSentence(torrent({ ...transmission, state: "seeding" }))).toBeUndefined();
});
