import { useQuery } from "@tanstack/react-query";
import { useLocation } from "@tanstack/react-router";

import {
  getGetReconcilingQueryKey,
  getReconciling,
} from "../api/gen/query/reconciling/reconciling";
import { useCounts } from "../shell/counts";
import { useNow } from "../tasks/use-now";
import { type Next, nextOf } from "./words";

/**
 * What the shell's line shows: nothing outside a restore's Reconciling or
 * on the Reconciling pages, which say it all, else the next step once the
 * Reconciling page's read is in.
 */
export function useBanner(): Next | "loading" | undefined {
  const on = (useCounts()?.reconciling ?? null) !== null;
  const now = useNow(60_000);
  const { data } = useQuery({
    queryKey: getGetReconcilingQueryKey(),
    queryFn: ({ signal }) => getReconciling({ signal }),
    enabled: on,
  });
  const onPage = useLocation({ select: ({ pathname }) => pathname.startsWith("/reconciling") });
  if (!on || onPage) {
    return undefined;
  }
  return data === undefined ? "loading" : nextOf(data, { now, zone: undefined });
}
