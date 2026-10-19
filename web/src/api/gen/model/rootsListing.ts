import type { RootsFolder } from "./rootsFolder.ts";
import type { RootsPlace } from "./rootsPlace.ts";
import type { RootsSegment } from "./rootsSegment.ts";
import type { RootsVolume } from "./rootsVolume.ts";

export interface RootsListing {
  /** Its subfolders, in natural order without case */
  folders: RootsFolder[];
  /** The hidden subfolders left out */
  hidden: number;
  /** The folder above it; empty at a drive, a share or /, and for the places */
  parent: string;
  /** The folder as its Node cleaned it; empty for the Node's places */
  path: string;
  /** For the empty path: the roots on the Node, then its drives or mounts */
  places: RootsPlace[];
  /** Its breadcrumbs as its Node splits it; a drive or a share is one */
  segments: RootsSegment[];
  /** SlipStream runs as a Windows service there, which sees no mapped drive letters */
  service: boolean;
  /** The volume it sits on, when the Node read it */
  volume?: RootsVolume;
}
