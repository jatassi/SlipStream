import type { InvalidateQueryFilters, Query, QueryClient } from "@tanstack/react-query";

const following = new WeakMap<Query, { again: boolean }>();

/**
 * Invalidates every query filters match without aborting a fetch in flight.
 * A query already fetching refetches once more after that fetch lands, however
 * many events arrive meanwhile, since the fetch in flight may predate them.
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
  const slot = { again: query.state.fetchStatus === "fetching" };
  following.set(query, slot);
  await queryClient.invalidateQueries(
    { predicate: (candidate) => candidate === query },
    { cancelRefetch: false },
  );
  following.delete(query);
  if (slot.again) {
    await refreshQuery(queryClient, query);
  }
}
