import type {
  GrabsFailure,
  GrabsFailureCode,
  GrabsFailureStage,
  GrabsGrab,
  GrabsGrabStage,
} from "../api/gen/model";
import type { Stage, TrackProgress } from "../ui/stage-track";

export const stageWord = {
  fetching: "Fetching",
  downloading: "Downloading",
  importing: "Importing",
  imported: "Imported",
  failed: "Failed",
} as const satisfies Record<GrabsGrabStage, string>;

export const trackStage = {
  fetching: "downloading",
  downloading: "downloading",
  importing: "import",
} as const satisfies Record<GrabsFailureStage, Stage>;

export const skippedStages: readonly Stage[] = ["conversion"];

export type FailedGrab = GrabsGrab & { readonly failure: GrabsFailure };

export const isFailed = (grab: GrabsGrab): grab is FailedGrab => grab.failure !== undefined;

/** A failure an Admin must act on. A cancel and a Manual Grab the Blocklist held are not. */
export const needsYou = (grab: GrabsGrab): grab is FailedGrab => grab.failure?.needsYou === true;

export function progressOf(grab: GrabsGrab): TrackProgress {
  if (isFailed(grab)) {
    return { kind: "failed", stage: trackStage[grab.failure.stage] };
  }
  if (grab.stage === "imported") {
    return { kind: "available" };
  }
  if (grab.stage === "failed") {
    throw new Error(`failed Grab ${grab.id} carries no failure`);
  }
  return { kind: "running", stage: trackStage[grab.stage] };
}

/** Drops each failure a later Grab of one of its cells replaced, as the Work's Needs you does. */
export const withoutReplacedFailures = (grabs: readonly GrabsGrab[]): GrabsGrab[] =>
  grabs.filter((grab) => !needsYou(grab) || grab.failure.current);

export const movesBytes = (grab: GrabsGrab): boolean => grab.stage === "downloading";

export type Words = { readonly cause: string; readonly recovery?: string };

const searchAgain = "Search again to pick another Release.";

const nextRelease =
  "SlipStream grabs the next acceptable Release from the same search, if there is one.";

export const failureWords: Readonly<Record<GrabsFailureCode, Words>> = {
  indexer_unavailable: {
    cause: "The Indexer didn't respond to the request for the Release.",
    recovery: "Search again once the Indexer is back.",
  },
  indexer_removed: {
    cause: "The Indexer was removed before SlipStream fetched the Release's .torrent file.",
    recovery: nextRelease,
  },
  indexer_stopped: {
    cause: "The Indexer stopped before SlipStream fetched the Release's .torrent file.",
    recovery: nextRelease,
  },
  home_offline: {
    cause:
      "The Indexer's home Node was offline past its grace before SlipStream fetched the Release's .torrent file.",
    recovery: nextRelease,
  },
  torrent_unreachable: {
    cause:
      "SlipStream couldn't download the Release's .torrent file or magnet link from the Indexer.",
    recovery: searchAgain,
  },
  magnet_invalid: {
    cause: "The Release's magnet link has no info hash SlipStream can read.",
    recovery: searchAgain,
  },
  no_metadata: {
    cause: "No peer sent the magnet link's metadata in time.",
    recovery: searchAgain,
  },
  no_peer_source: {
    cause: "The magnet link names no announce URL, and DHT is off on the Node that downloads it.",
    recovery:
      "Turn on DHT for public torrents in that Node's torrent client settings, or search again.",
  },
  torrent_invalid: {
    cause:
      "SlipStream can't read the Release's .torrent file, or the metadata peers sent for its magnet link.",
    recovery: searchAgain,
  },
  infohash_mismatch: {
    cause: "The .torrent file isn't the Release the Indexer listed.",
    recovery: searchAgain,
  },
  torrent_v2_only: {
    cause: "The Release only offers a BitTorrent v2 .torrent, which SlipStream can't download.",
    recovery: searchAgain,
  },
  download_failed: {
    cause: "The download stopped with an error.",
    recovery: searchAgain,
  },
  marked_failed: {
    cause: "An Admin marked the download failed. SlipStream added the Release to the Blocklist.",
    recovery: searchAgain,
  },
  removed: {
    cause: "An Admin removed the torrent before SlipStream imported it.",
    recovery: searchAgain,
  },
  client_removed: {
    cause: "The download client removed the torrent before SlipStream imported it.",
    recovery: searchAgain,
  },
  client_deleted: {
    cause: "An Admin deleted the download client before SlipStream imported the torrent.",
    recovery: searchAgain,
  },
  goal_met: {
    cause: "The torrent met its seeding goal and was removed before SlipStream imported it.",
    recovery: searchAgain,
  },
  no_video_file: {
    cause: "The download holds no video file.",
    recovery: searchAgain,
  },
  probe_failed: {
    cause: "SlipStream couldn't read the downloaded video.",
    recovery: searchAgain,
  },
  fills_nothing: {
    cause: "The downloaded video doesn't meet any Target this Work still wants.",
    recovery: searchAgain,
  },
  name_taken: {
    cause: "Another file already has the name this video would take in the Work's folder.",
    recovery: "Move or rename the other file, then search again.",
  },
  name_too_long: {
    cause: "The video's path would be longer than its library root allows.",
    recovery: "Shorten this Content Type's naming templates, then search again.",
  },
  transfer_failed: {
    cause: "The finished download couldn't be copied between Nodes.",
  },
  import_failed: {
    cause: "SlipStream couldn't put the file in the library.",
  },
  node_written_off: {
    cause: "What the Node downloading it held was written off. Its files are kept.",
    recovery: searchAgain,
  },
  root_changed: {
    cause: "The Work moved to another library root after this download was planned.",
    recovery: searchAgain,
  },
  canceled: {
    cause: "An Admin canceled this Grab.",
    recovery: searchAgain,
  },
  blocklisted: {
    cause: "The Release's torrent is on the Blocklist.",
    recovery: nextRelease,
  },
  release_fault: {
    cause:
      "The tracker reported the torrent unregistered, or it never downloaded any data. SlipStream added the Release to the Blocklist.",
    recovery: searchAgain,
  },
  client_error: {
    cause:
      "The download client reported an error on the torrent for longer than the client error setting allows.",
    recovery: nextRelease,
  },
  declined: {
    cause: "You chose not to import this download.",
  },
  unreadable: {
    cause:
      "ffprobe can't read the downloaded video. SlipStream added the Release to the Blocklist.",
    recovery: nextRelease,
  },
};
