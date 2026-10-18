import { useQuery } from "@tanstack/react-query";

import type { GrabsGrabStage, ShellCounts } from "../api/gen/model";
import { getCounts, getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { useAccount } from "../session/session";
import type { CountDot } from "../ui/count";

export type CountKey = Exclude<keyof ShellCounts, "alerts">;

// The server counts; this is the client's reading of which stages it counts.
export const activityStages = {
  fetching: true,
  downloading: true,
  probing: true,
  transferring: true,
  importing: true,
  imported: false,
  failed: false,
} as const satisfies Record<GrabsGrabStage, boolean>;

export function useCounts(): ShellCounts | undefined {
  const { admin } = useAccount();
  return useQuery({
    queryKey: getGetCountsQueryKey(),
    queryFn: ({ signal }) => getCounts({ signal }),
    enabled: admin,
  }).data;
}

export const countDots = { activity: "live" } as const satisfies Record<CountKey, CountDot>;
