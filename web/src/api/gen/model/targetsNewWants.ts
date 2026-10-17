import type { TargetsEditions } from "./targetsEditions.ts";

export interface TargetsNewWants {
  /** The Work's own Edition list, or null for its Profile's */
  editions: TargetsEditions | null;
  profileId: string;
}
