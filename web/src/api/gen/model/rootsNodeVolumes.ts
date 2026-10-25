import type { RootsRootVolume } from "./rootsRootVolume.ts";

export interface RootsNodeVolumes {
  /** The Node is online, so its figures are current. Otherwise each volume's figures are from its sampledAt */
  live: boolean;
  nodeId: string;
  /** Its volumes that hold a root, by name */
  volumes: RootsRootVolume[];
}
