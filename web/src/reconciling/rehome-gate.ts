import type { QueryClient } from "@tanstack/react-query";
import { type ParsedLocation, redirect } from "@tanstack/react-router";

import { getGetCountsSuspenseQueryOptions } from "../api/gen/query/shell/shell";
import { ensureQuery } from "../api/query-client";

export const rehomeStep = "/reconciling/folders";

/**
 * After a restore on a new machine, finding the old Controller's folders
 * comes before every other page until it is saved or left for later. It
 * reads the shell's counts, which the banner reads too.
 */
export async function requireRehomed(queryClient: QueryClient, location: ParsedLocation) {
  const { reconciling } = await ensureQuery(queryClient, getGetCountsSuspenseQueryOptions());
  if (reconciling?.rehome === true && location.pathname !== rehomeStep) {
    redirect({ to: rehomeStep, replace: true, throw: true });
  }
}
