import type { RootsVolumeKind } from "./rootsVolumeKind.ts";

export interface RootsVolume {
  /** The bytes SlipStream may write */
  free: number;
  /** The filesystem id, or the volume serial on Windows */
  id: string;
  kind: RootsVolumeKind;
  mount: string;
  /** The volume as its Node names it, such as D: (Media) or /mnt/media */
  name: string;
  sampledAt: string;
  total: number;
}
