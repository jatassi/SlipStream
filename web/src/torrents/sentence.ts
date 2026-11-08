import { sinceWords } from "../format";
import type { LiveRow } from "../live/torrents";
import {
  type Place,
  type Row,
  type Waits,
  backWords,
  ordinal,
  paceOf,
  seedLength,
  stateWords,
  trackerLine,
  waitsOn,
} from "./model";

const peers = (count: number) => `${count} ${count === 1 ? "peer" : "peers"}`;

const backSentence = ({ who, cause }: Waits) => `${who} ${backWords[cause].sentence}`;

function pendingSentence(row: Row, place: Place): string | undefined {
  const waits = waitsOn(row, place);
  const { node } = place;
  switch (row.pending) {
    case "pause":
      return waits === undefined
        ? `Pausing. It stops downloading and seeding once ${node} applies the pause.`
        : `It pauses when ${backSentence(waits)}.`;
    case "resume":
      return waits === undefined
        ? `Resuming. It continues once ${node} applies the resume.`
        : `It resumes when ${backSentence(waits)}.`;
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
        ? "Waiting until its announce URLs accept an announce again."
        : `Waiting for an announce. It resumes at ${sinceWords(row.until, clock)}, when every announce URL accepts one again.`;
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
    case "client_error":
      return `${row.clientName} stopped this torrent with the error "${row.error}". Fix the cause in ${row.clientName} and the download carries on.`;
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
    return `${host} reports that this torrent is no longer registered, so no peers come from it. Remove it, then search for another Release.`;
  }
  const said =
    tracker?.message === undefined
      ? `${host} isn't responding, so no peers come from it.`
      : `${host} reported "${tracker.message}", so no peers come from it.`;
  return tracker?.retryAt === undefined
    ? said
    : `${said} The client retries at ${sinceWords(tracker.retryAt, clock)}.`;
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

// Why a row is its last report, and what it was doing then.
function lastKnownSentence(row: Row, { cause, who, at }: Waits, place: Place) {
  const when = sinceWords(at, place.clock);
  const was = stateWords[row.state].toLowerCase();
  switch (cause) {
    case "node":
      return `${place.node} is offline. When it last reported, at ${when}, this was ${was}.`;
    case "client":
      return `${who} on ${place.node} isn't responding. When it last responded, at ${when}, this was ${was}.`;
    case "off":
      return `SlipStream isn't checking ${who} while it's turned off. When it last responded, at ${when}, this was ${was}.`;
    default:
      return cause satisfies never;
  }
}

/** What removing a torrent SlipStream attached to does, none for one it added. */
export const attachedSentence = (row: Row): string | undefined =>
  row.attached === true && row.clientName !== undefined
    ? `This torrent was in ${row.clientName} before SlipStream attached to it. Removing it here stops SlipStream tracking it and leaves it in ${row.clientName}.`
    : undefined;

/**
 * What the Inspector leads with: what the Torrent is doing and what happens
 * next, in words, so no number needs reading.
 */
export function inspectorSentence(row: Row, live: LiveRow | undefined, place: Place): string {
  const pending = pendingSentence(row, place);
  if (pending !== undefined) {
    return pending;
  }
  const waits = waitsOn(row, place);
  if (row.state === "removing") {
    return waits === undefined || waits.cause === "node"
      ? `Removing. Waiting for ${place.node} to confirm.`
      : `It's removed once ${backSentence(waits)}.`;
  }
  if (waits !== undefined) {
    return lastKnownSentence(row, waits, place);
  }
  if (row.importAt !== undefined) {
    return `Finished. SlipStream imports it at ${sinceWords(row.importAt, place.clock)}, when the Import delay for ${row.clientName ?? "its client"} ends.`;
  }
  switch (row.state) {
    case "failed":
      return row.error === undefined
        ? "The torrent client stopped it. Remove it, then search for another Release."
        : `The torrent client stopped it with the error "${row.error}". Remove it, then search for another Release.`;
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
      return row.stoppedAtLimit === true
        ? `${row.clientName ?? "Its client"} stopped it at its seeding limit.`
        : heldSentence(row, place);
    case "stalled":
      return "Stalled. No data has arrived for a while, so it no longer holds a download slot. To stop using this Release, mark it failed.";
    case "downloading":
    case "seeding":
      return movingSentence(row, live);
    default:
      return row.state satisfies never;
  }
}
