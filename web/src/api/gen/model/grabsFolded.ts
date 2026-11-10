import type { GrabsFoldGroup } from "./grabsFoldGroup.ts";

export interface GrabsFolded {
  groups: GrabsFoldGroup[];
  /** Such as SlipStream wouldn't grab 14 more Releases. */
  line: string;
}
