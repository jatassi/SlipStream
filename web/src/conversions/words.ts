import {
  Alert02Icon,
  Cancel01Icon,
  Copy01Icon,
  Exchange01Icon,
  FileExportIcon,
  FileImportIcon,
  FileSearchIcon,
  PauseIcon,
  Queue01Icon,
  Tick02Icon,
} from "@hugeicons/core-free-icons";

import type {
  ConversionsConversionRow,
  ConversionsConversionRowLane,
  ConversionsConversionRowStage,
  ConversionsCounts,
  ConversionsWait,
  ConversionsWaitCause,
  ConversionsWaitCode,
} from "../api/gen/model";
import { bytes } from "../format";
import { ordinal } from "../torrents/model";
import type { Clock } from "../ui/clock";
import type { StatusPart } from "../ui/status-sentence";
import type { TermsFor } from "../ui/state-marks";
import { whenWords } from "../ui/time-words";

export const stageTerms = {
  queued: {
    word: "Waiting",
    icon: Queue01Icon,
    tone: "quiet",
    definition: "The Conversion is in line and starts once a Node can run it.",
  },
  copying: {
    word: "Copying",
    icon: Copy01Icon,
    tone: "neutral",
    definition: "SlipStream is copying the source file to the Node that converts it.",
  },
  ready: {
    word: "Ready",
    icon: Queue01Icon,
    tone: "neutral",
    definition: "The source file is on the Node, and the Conversion starts when the Node has room.",
  },
  running: { word: "Converting", icon: Exchange01Icon, tone: "live" },
  paused: {
    word: "Paused",
    icon: PauseIcon,
    tone: "quiet",
    definition: "The Conversion stopped while a Media Server re-encodes video on the same Node.",
  },
  checking: {
    word: "Checking",
    icon: FileSearchIcon,
    tone: "neutral",
    definition: "SlipStream is checking that the new file is as long as the source and plays.",
  },
  returning: {
    word: "Copying back",
    icon: FileExportIcon,
    tone: "neutral",
    definition: "SlipStream is copying the new file to the Node that stores the library.",
  },
  landing: {
    word: "Importing",
    icon: FileImportIcon,
    tone: "neutral",
    definition: "SlipStream is importing the new file into the library.",
  },
  done: { word: "Done", icon: Tick02Icon, tone: "met" },
  failed: { word: "Failed", icon: Alert02Icon, tone: "fail" },
  canceled: { word: "Canceled", icon: Cancel01Icon, tone: "quiet" },
} as const satisfies TermsFor<
  ConversionsConversionRowStage,
  "quiet" | "neutral" | "live" | "fail" | "met",
  "running" | "done" | "failed" | "canceled"
>;

export const laneHeadings: { readonly [L in ConversionsConversionRowLane]: string } = {
  moved_up: "Moved up",
  new: "New",
  upgrade: "After an upgrade",
  backfill: "Backfill",
};

const nodeOf = (wait: ConversionsWait) => wait.node?.name ?? "the Node";

const listed = new Intl.ListFormat("en", { type: "conjunction" });

/** "Living room Plex transcodes" or "Living room Plex and Den Jellyfin transcode". */
export const transcodingWords = (servers: readonly string[]): string =>
  `${listed.format(servers)} ${servers.length === 1 ? "transcodes" : "transcode"}`;

export const pausedWords = (servers: readonly string[], node: string): string =>
  `Paused while ${transcodingWords(servers)} on ${node}.`;

/** One Conversion's words for why no Node runs it, one per cause. */
export const causeWords: {
  readonly [C in ConversionsWaitCause]: (node: string, row: ConversionsConversionRow) => string;
} = {
  no_role: () => "No Node has the Converting Role. Give one the Role on the Nodes page.",
  none_online: () => "No Node with the Converting Role is online. It starts when one is back.",
  no_holder: () =>
    "No Node with the Converting Role is ready to convert. It starts when one is ready.",
  lapsed: (node) => `${node} is offline. It starts when ${node} is back.`,
  runs_elsewhere: (node, row) =>
    `${row.target.name} runs only on ${node}, which can't convert this now. It starts when ${node} can, or change Run on for ${row.target.name}.`,
  pinned_elsewhere: (node) =>
    `It's set to run again on ${node}, which can't convert this now. It starts when ${node} can.`,
  failed_here: (node) =>
    `It already failed on ${node}, and no other Node can convert it. It starts when another Node can.`,
  untested: (node) =>
    `${node} hasn't tested this kind of video yet. It starts after ${node}'s test run.`,
  test_failed: (node) =>
    `${node}'s test run for this kind of video failed. Rerun the test run on ${node}.`,
  retesting: (node) => `${node} is rerunning its test runs. It starts once they pass.`,
  proof_lost: (node) =>
    `${node}'s test run failed after its driver changed. Rerun the test run on ${node}.`,
  processor_off: (node) =>
    `Only the processor can convert this, and Use the processor when no GPU can is off on ${node}.`,
};

const startWords = (row: ConversionsConversionRow, clock: Clock) =>
  row.start === null ? "" : ` It should start ${whenWords(row.start, "at", clock)}`;

/** Why a Conversion hasn't started, keyed by the wire's codes so a new one needs words. */
export const waitWords: {
  readonly [W in ConversionsWaitCode]: (
    wait: ConversionsWait,
    row: ConversionsConversionRow,
    clock: Clock,
  ) => string;
} = {
  in_line: (wait, row, clock) => {
    const start = startWords(row, clock);
    const place = `${ordinal(wait.place ?? 1)} in line.`;
    return start === "" ? place : `${place}${start} on ${nodeOf(wait)}.`;
  },
  hours: (wait, _row, clock) =>
    `Waiting ${wait.until === undefined ? "" : `${whenWords(wait.until, "until", clock)}, `}when ${nodeOf(wait)} starts Conversions again.`,
  room: (wait) =>
    `Waiting for room on ${nodeOf(wait)}, which has ${bytes(wait.free ?? 0)} free of the ${bytes(wait.need ?? 0)} this needs.`,
  copying: (wait) => `Copying the source to ${nodeOf(wait)}.`,
  away: (wait) => `Waiting for ${nodeOf(wait)}, which is offline.`,
  download: (wait, _row, clock) =>
    `Waiting for a download${wait.until === undefined ? "" : ` ${whenWords(wait.until, "until", clock)}`}. Then SlipStream makes it from ${wait.sourceTarget ?? "its source"}.`,
  testing: (wait) => `Waiting for ${nodeOf(wait)} to finish a test run.`,
  transcode: (wait) => pausedWords(wait.servers ?? [], nodeOf(wait)),
  no_node: (wait, row) =>
    wait.cause === undefined
      ? "No Node can convert this now."
      : causeWords[wait.cause](nodeOf(wait), row),
  no_source: (wait) => `Waits for the ${wait.sourceTarget ?? "source"} file.`,
};

const rowWait = (row: ConversionsConversionRow, clock: Clock): string =>
  row.wait === null ? "" : waitWords[row.wait.code](row.wait, row, clock);

export function waitingLine(row: ConversionsConversionRow, clock: Clock): string {
  const on = row.node === null ? "" : ` on ${row.node.name}`;
  const finish = row.finish === null ? "" : ` Should finish ${whenWords(row.finish, "at", clock)}.`;
  return `${row.target.name} from ${row.source.target}${on} · ${rowWait(row, clock)}${finish}`;
}

export const attentionAnchor = "needs-attention";

/** The tab's opening sentence, its counts in the order the spec reads them. */
export function sentenceParts(counts: ConversionsCounts, clock: Clock): StatusPart[] {
  const { converting, paused, waiting, failed, clearsAt } = counts;
  if (converting + paused + waiting + failed === 0) {
    return [{ kind: "plain", text: "Nothing to convert." }];
  }
  const queue = [
    converting > 0 && `${converting} converting`,
    paused > 0 && `${paused} paused`,
    waiting > 0 && `${waiting} waiting`,
  ].filter((part) => part !== false);
  const clears =
    clearsAt === null ? "" : ` The queue should clear ${whenWords(clearsAt, "at", clock)}.`;
  return [
    ...(failed > 0
      ? [
          {
            kind: "failure",
            text: `${failed} ${failed === 1 ? "Conversion" : "Conversions"} failed and ${failed === 1 ? "needs" : "need"} attention.`,
            href: `#${attentionAnchor}`,
          } as const,
        ]
      : []),
    ...(queue.length > 0
      ? [
          {
            kind: converting > 0 ? "live" : "plain",
            text: `${queue.join(", ")}.${clears}`,
          } as const,
        ]
      : []),
  ];
}
