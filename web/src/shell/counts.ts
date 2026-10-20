import { useQuery } from "@tanstack/react-query";

import type {
  DownloadclientsAggregate,
  GrabsGrabStage,
  ShellAlerts,
  ShellCounts,
} from "../api/gen/model";
import { getCounts, getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { useAccount } from "../session/session";
import type { AlertTone, CountDot } from "../ui/count";

export type CountKey = {
  [K in keyof ShellCounts]: ShellCounts[K] extends number ? K : never;
}[keyof ShellCounts];

export type FillKey = {
  [K in keyof ShellCounts]: ShellCounts[K] extends DownloadclientsAggregate ? K : never;
}[keyof ShellCounts];

export type AlertKey = keyof ShellAlerts;

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

export const alertOf = (
  counts: ShellCounts | undefined,
  key: AlertKey | undefined,
): AlertTone | undefined => (key === undefined ? undefined : (counts?.alerts[key] ?? undefined));

/** How far a row's fill reaches, from 0 to 100, and whether all of it is paused. */
export type Fill = { percent: number; paused: boolean };

export function fillOf(
  counts: ShellCounts | undefined,
  key: FillKey | undefined,
): Fill | undefined {
  if (counts === undefined || key === undefined) {
    return undefined;
  }
  const { done, wanted, allPaused } = counts[key];
  if (wanted === 0) {
    return undefined;
  }
  return { percent: Math.min(100, (done / wanted) * 100), paused: allPaused };
}
