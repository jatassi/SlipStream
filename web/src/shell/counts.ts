import type { GrabsGrabStage } from "../api/gen/model";

// The server counts; this is the client's reading of which stages it counts.
export const activityStages = {
  fetching: true,
  downloading: true,
  probing: true,
  transferring: true,
  importing: true,
  imported: false,
  failed: false,
} as const satisfies Record<GrabsGrabStage, boolean>;
