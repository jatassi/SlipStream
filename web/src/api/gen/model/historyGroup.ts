import type { HistoryGroupGroup } from "./historyGroupGroup.ts";
import type { HistoryKind } from "./historyKind.ts";

export interface HistoryGroup {
  group: HistoryGroupGroup;
  kinds: HistoryKind[];
  label: string;
}
