import type { FilesUpgradeNeedKind } from "./filesUpgradeNeedKind.ts";

export interface FilesUpgradeNeed {
  /** any under any cut, else the key of the cut the cells want */
  edition: string;
  /** A run of Items in tree order whose entries match */
  itemIds: string[];
  kind: FilesUpgradeNeedKind;
  /** Every upgradable Target of the Items and Edition */
  upgrades: string[];
}
