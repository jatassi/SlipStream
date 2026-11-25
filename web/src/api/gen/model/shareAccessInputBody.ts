import type { ShareAccessInputBodyAs } from "./shareAccessInputBodyAs.ts";

export interface ShareAccessInputBody {
  /** friend shares the Libraries for Members; home also adds the Member to the Plex Home */
  as: ShareAccessInputBodyAs;
}
