import type { FilesState } from "./filesState.ts";

export interface FilesItemStatus {
  itemId: string;
  /** Its availability date is today or earlier */
  released: boolean;
  /** Its first state in the order failed, missing, downloading, converting, upgradable, unreleased, met, across its Targets; null for an Item with no Target */
  status: FilesState | null;
}
