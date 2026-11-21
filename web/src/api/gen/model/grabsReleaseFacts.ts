import type { GrabsReleaseFactsQuality } from "./grabsReleaseFactsQuality.ts";

export interface GrabsReleaseFacts {
  /** The first audio codec and channel layout the title names, such as DD+ 5.1; absent when it names neither */
  audio?: string;
  /** The Release group, such as FLUX; absent when the title names none */
  group?: string;
  /** The first HDR format the title names, such as Dolby Vision; absent when it names none */
  hdr?: string;
  /** The quality the title names; absent when it names none */
  quality?: GrabsReleaseFactsQuality;
  /** Such as v2 or v2 REPACK; absent for the original release */
  revision?: string;
}
