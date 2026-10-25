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
  probing: "Checking",
  transferring: "Transferring",
  importing: "Importing",
  imported: "Imported",
  failed: "Failed",
} as const satisfies Record<GrabsGrabStage, string>;

export const trackStage = {
  fetching: "downloading",
  downloading: "downloading",
  probing: "import",
  transferring: "transfer",
  importing: "import",
} as const satisfies Record<GrabsFailureStage, Stage>;

export const skippedStages: readonly Stage[] = ["conversion"];

export type FailedGrab = GrabsGrab & { readonly failure: GrabsFailure };

export const isFailed = (grab: GrabsGrab): grab is FailedGrab => grab.failure !== undefined;

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

export function withoutRetriedFailures(grabs: readonly GrabsGrab[]): GrabsGrab[] {
  return grabs.filter(
    (grab) =>
      !isFailed(grab) ||
      !grabs.some((later) => later.itemId === grab.itemId && later.createdAt > grab.createdAt),
  );
}

const byteStages = new Set<GrabsGrabStage>(["downloading", "transferring"]);

export const movesBytes = (grab: GrabsGrab): boolean => byteStages.has(grab.stage);

export type Words = { readonly cause: string; readonly recovery?: string };

const searchAgain = "Search again to pick another release.";

export const failureWords: Readonly<Record<GrabsFailureCode, Words>> = {
  indexer_unavailable: {
    cause: "The Indexer didn't respond to the request for the release.",
    recovery: "Search again once the Indexer is back.",
  },
  torrent_unreachable: {
    cause: "SlipStream couldn't download the release's .torrent file from the Indexer.",
    recovery: searchAgain,
  },
  torrent_invalid: {
    cause: "The .torrent file the Indexer sent is damaged.",
    recovery: searchAgain,
  },
  infohash_mismatch: {
    cause: "The .torrent file isn't the release the Indexer listed.",
    recovery: searchAgain,
  },
  torrent_v2_only: {
    cause: "The release only offers a BitTorrent v2 .torrent, which SlipStream can't download.",
    recovery: searchAgain,
  },
  no_download_client: {
    cause: "The Node this Indexer belongs to doesn't download.",
  },
  download_failed: {
    cause: "The download stopped with an error.",
    recovery: searchAgain,
  },
  removed: {
    cause: "An Admin removed the torrent before it finished downloading.",
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
  needs_transfer: {
    cause:
      "The Work's library root is on another Worker, and moving files into a root on another Worker isn't supported yet.",
    recovery: "Tick Downloading on the Node that holds the root, then search again.",
  },
};
