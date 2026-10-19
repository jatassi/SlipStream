import type { RootsRootVolumeKind } from "./rootsRootVolumeKind.ts";

export interface RootsRootVolume {
  /** The bytes SlipStream may write */
  free: number;
  /** The filesystem id, or the volume serial on Windows */
  id: string;
  kind: RootsRootVolumeKind;
  mount: string;
  /** The volume as its Node names it, such as D: (Media) or /mnt/media */
  name: string;
  /** The roots its Node last found ready on it */
  rootIds: string[];
  /** When its Node read these figures */
  sampledAt: string;
  total: number;
}
