import { sinceWords } from "../format";
import type { LiveRow } from "../live/torrents";
import {
  type Place,
  type Row,
  ordinal,
  paceOf,
  seedLength,
  stateWords,
  trackerLine,
} from "./model";

const peers = (count: number) => `${count} ${count === 1 ? "peer" : "peers"}`;

function pendingSentence(row: Row, { node }: Place): string | undefined {
  const away = row.lastKnown !== undefined;
  switch (row.pending) {
    case "pause":
      return away
        ? `It pauses when ${node} is back online.`
        : `Pausing. It stops downloading and seeding once ${node} applies the pause.`;
    case "resume":
      return away
        ? `It resumes when ${node} is back online.`
        : `Resuming. It continues once ${node} applies the resume.`;
    case undefined:
      return undefined;
    default:
      return row.pending satisfies never;
  }
}

function heldSentence(row: Row, { node, clock }: Place): string {
  switch (row.hold) {
    case "window":
      return row.until === undefined
        ? "Waiting for its tracker to accept an announce again."
        : `Waiting for its tracker. It resumes at ${sinceWords(row.until, clock)}, when every tracker accepts an announce again.`;
    case "disk_full":
      return `Paused because ${node} ran out of disk space. It resumes once there is space.`;
    case "slot":
      return row.queuePosition === undefined
        ? `Queued on ${node}. It starts once a download slot is free.`
        : `Queued, ${ordinal(row.queuePosition)} in line on ${node}. It starts once a download slot is free.`;
    case "check_slot":
      return "Waiting to check its files. It checks once the checks ahead of it finish.";
    case "disk_guard":
      return `Waiting for disk space on ${node}. It starts once there is room.`;
    case "draining":
      return `Waiting because ${node} is draining.`;
    case "controller":
      return `Waiting for the Controller to send it to ${node}.`;
    case "wanted":
    case undefined:
      return "Paused. It stays paused until you resume it.";
    default:
      return row.hold satisfies never;
  }
}

function trackerSentence(row: Row, { clock }: Place): string {
  const { host } = trackerLine(row.tracker, clock);
  const tracker = row.tracker;
  if (tracker?.status === "unregistered") {
    return `${host} reports that this torrent is no longer registered, so no peers come from it. Remove it, then search for another release.`;
  }
  const said =
    tracker?.message === undefined
      ? `${host} isn't responding, so no peers come from it.`
      : `${host} responded with "${tracker.message}", so no peers come from it.`;
  return tracker?.retryAt === undefined
    ? said
    : `${said} SlipStream retries at ${sinceWords(tracker.retryAt, clock)}.`;
}

function movingSentence(row: Row, live: LiveRow | undefined): string {
  if (row.state === "seeding") {
    const ratio = `Ratio ${row.ratio.toFixed(2)}`;
    const after =
      row.seedingSeconds < 60
        ? `${ratio} so far.`
        : `${ratio} after ${seedLength(row.seedingSeconds)}.`;
    return live === undefined || live.peers === 0
      ? `Seeding. ${after}`
      : `Seeding to ${peers(live.peers)}. ${after}`;
  }
  if (live === undefined) {
    return "Downloading.";
  }
  const { eta, rate } = paceOf(row.state, live);
  const pace = eta === "" ? "" : `, ${eta} at ${rate}`;
  return `Downloading from ${peers(live.peers)}${pace}.`;
}

/**
 * What the Inspector leads with: what the Torrent is doing and what happens
 * next, in words, so no number needs reading.
 */
export function inspectorSentence(row: Row, live: LiveRow | undefined, place: Place): string {
  const pending = pendingSentence(row, place);
  if (pending !== undefined) {
    return pending;
  }
  if (row.state === "removing") {
    return `Removing. Waiting for ${place.node} to confirm.`;
  }
  if (row.lastKnown !== undefined) {
    return `${place.node} is offline. When it last reported, at ${sinceWords(row.lastKnown, place.clock)}, this was ${stateWords[row.state].toLowerCase()}.`;
  }
  switch (row.state) {
    case "failed":
      return row.error === undefined
        ? "The torrent client stopped it. Remove it, then search for another release."
        : `The torrent client stopped it with the error "${row.error}". Remove it, then search for another release.`;
    case "tracker-error":
      return trackerSentence(row, place);
    case "checking":
      return row.until === undefined
        ? "Checking its files. It continues once the check is done."
        : `Checking its files. It resumes at ${sinceWords(row.until, place.clock)}.`;
    case "queued":
      return row.hold === undefined
        ? `Waiting for ${place.node} to start it.`
        : heldSentence(row, place);
    case "paused":
      return heldSentence(row, place);
    case "stalled":
      return "Stalled. No data has arrived for a while, so it no longer holds a download slot. To stop using this release, mark it failed.";
    case "downloading":
    case "seeding":
      return movingSentence(row, live);
    default:
      return row.state satisfies never;
  }
}
