import type { InvalidateQueryFilters, Query, QueryClient } from "@tanstack/react-query";

export const everyPath = (pattern: RegExp) => ({
  predicate: ({ queryKey: [path] }: { queryKey: readonly unknown[] }) =>
    typeof path === "string" && pattern.test(path),
});

const following = new WeakMap<Query, { again: boolean }>();

/**
 * Invalidates every query filters match without aborting a fetch in flight.
 * A query already fetching is invalidated after that fetch lands, since the
 * fetch may predate the event, and once more if events arrive meanwhile.
 * Live events come in bursts, and macOS WebKit stops sending fetches to the
 * host after a burst of aborted ones.
 */
export async function refresh(
  queryClient: QueryClient,
  filters?: InvalidateQueryFilters,
): Promise<void> {
  const queries = queryClient.getQueryCache().findAll(filters);
  await Promise.all(queries.map((query) => refreshQuery(queryClient, query)));
}

async function refreshQuery(queryClient: QueryClient, query: Query): Promise<void> {
  const inFlight = following.get(query);
  if (inFlight !== undefined) {
    inFlight.again = true;
    return;
  }
  const slot = { again: false };
  following.set(query, slot);
  if (query.state.fetchStatus === "fetching") {
    await Promise.allSettled([query.promise]);
  }
  await queryClient.invalidateQueries(
    { predicate: (candidate) => candidate === query },
    { cancelRefetch: false },
  );
  following.delete(query);
  if (slot.again) {
    await refreshQuery(queryClient, query);
  }
}
