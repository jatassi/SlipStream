import {
  Alert02Icon,
  Delete02Icon,
  Download04Icon,
  FileSearchIcon,
  HourglassIcon,
  PauseIcon,
  Queue01Icon,
  Unlink03Icon,
  Upload04Icon,
} from "@hugeicons/core-free-icons";

import type { StatusMark, TermsFor, Tone } from "../ui/state-marks";
import type { Row, State } from "./model";

/** A torrent's states as torrent clients name them. Only the server's severity makes one a signal chip. */
export const torrentTerms = {
  removing: {
    word: "Removing",
    icon: Delete02Icon,
    tone: "quiet",
    definition: "SlipStream is removing the torrent from its torrent client.",
  },
  failed: { word: "Failed", icon: Alert02Icon, tone: "neutral" },
  "tracker-error": {
    word: "Tracker error",
    icon: Unlink03Icon,
    tone: "neutral",
    definition: "None of the torrent's trackers is working, so it can't find new peers.",
  },
  paused: {
    word: "Paused",
    icon: PauseIcon,
    tone: "quiet",
    definition: "The torrent is stopped and doesn't download or upload until it resumes.",
  },
  checking: {
    word: "Checking",
    icon: FileSearchIcon,
    tone: "neutral",
    definition: "The torrent client is checking the files on disk against the torrent's pieces.",
  },
  queued: {
    word: "Queued",
    icon: Queue01Icon,
    tone: "quiet",
    definition: "The torrent is in line and starts once its turn comes.",
  },
  stalled: {
    word: "Stalled",
    icon: HourglassIcon,
    tone: "neutral",
    definition: "The torrent is incomplete, and no peer is sending it data.",
  },
  downloading: { word: "Downloading", icon: Download04Icon, tone: "live" },
  seeding: {
    word: "Seeding",
    icon: Upload04Icon,
    tone: "neutral",
    definition: "The torrent is complete and uploads to other peers.",
  },
} as const satisfies TermsFor<State, Exclude<Tone, "fail" | "warn">, "failed" | "downloading">;

const signals = { none: undefined, warning: "warn", failure: "fail" } as const;

/** The row's state mark, raised to the warn or fail chip when the server says it needs attention. */
export function torrentMark(row: Row): StatusMark {
  const term = torrentTerms[row.state];
  const signal = signals[row.severity];
  return signal === undefined ? term : { ...term, tone: signal };
}
