import type { FilesStep } from "./filesStep.ts";

export interface FilesHeldCell {
  /** When the Grab entered its stage */
  from: string;
  /** The Grab in flight for this Item at this Target */
  grabId: string;
  step: FilesStep;
  /**
   * When the Import delay ends, for a download that waits for it; null otherwise
   * @nullable
   */
  until: string | null;
}
