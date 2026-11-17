import type { GrabsFailure, GrabsFailureStage, GrabsGrab, GrabsGrabStage } from "../api/gen/model";
import type { Stage, TrackProgress } from "../ui/stage-track";

export const stageWord = {
  fetching: "Fetching",
  downloading: "Downloading",
  importing: "Importing",
  imported: "Imported",
  failed: "Failed",
} as const satisfies Record<GrabsGrabStage, string>;

const stopOf = {
  fetching: "downloading",
  downloading: "downloading",
  importing: "import",
} as const satisfies Record<GrabsFailureStage, Stage>;

/** The stop a Grab's track lights: the one it failed at, Transfer while a file of its Import sends, else its stage's. */
export function trackStage(grab: GrabsGrab): Stage {
  if (grab.failure !== undefined) {
    return stopOf[grab.failure.stage];
  }
  if (grab.stage === "imported" || grab.stage === "failed") {
    throw new Error(`Grab ${grab.id} is ${grab.stage}, which lights no single stop`);
  }
  return (grab.import?.sending ?? 0) > 0 ? "transfer" : stopOf[grab.stage];
}

export const skippedStages: readonly Stage[] = ["conversion"];

export type FailedGrab = GrabsGrab & { readonly failure: GrabsFailure };

export const isFailed = (grab: GrabsGrab): grab is FailedGrab => grab.failure !== undefined;

/** A failure an Admin must act on. A cancel and a Manual Grab the Blocklist held are not. */
export const needsYou = (grab: GrabsGrab): grab is FailedGrab => grab.failure?.needsYou === true;

export function progressOf(grab: GrabsGrab): TrackProgress {
  if (isFailed(grab)) {
    return { kind: "failed", stage: trackStage(grab) };
  }
  if (grab.stage === "imported") {
    return { kind: "available" };
  }
  return { kind: "running", stage: trackStage(grab) };
}

/** Drops each failure a later Grab of one of its cells replaced, as the Work's Needs you does. */
export const withoutReplacedFailures = (grabs: readonly GrabsGrab[]): GrabsGrab[] =>
  grabs.filter((grab) => !needsYou(grab) || grab.failure.current);

/** An importing Grab's download can be imported by hand in the Import files sheet. */
export const byHand = (grab: GrabsGrab): boolean =>
  grab.stage === "importing" && grab.import !== null;

export const movesBytes = (grab: GrabsGrab): boolean => grab.stage === "downloading";
