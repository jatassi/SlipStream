import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { takeConversions } from "./conversions";
import { everyPath } from "./refresh";
import type { LiveEffects } from "./types";

const everyConversionList = everyPath(/^\/api\/v1\/conversions(\/groups\/[^/]+)?$/u);

export const conversionEffects = {
  "conversions.changed": (data, { refresh }) => {
    refresh(everyConversionList);
    refresh({ queryKey: getGetCountsQueryKey() });
    for (const workId of data.workIds) {
      refresh({ queryKey: getGetWorkCellsQueryKey(workId) });
    }
  },
  "conversions.live": (data, { queryClient }) => {
    takeConversions(queryClient, data);
  },
} satisfies Pick<LiveEffects, "conversions.changed" | "conversions.live">;
