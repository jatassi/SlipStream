import type { QueryClient, QueryKey } from "@tanstack/react-query";

import { refresh } from "./refresh";

type Row = { readonly id: string };

/** A cached list whose rows a live event names by id. */
export type Listed = { readonly rows: readonly Row[] };

/**
 * The list with fresh's answer for the named ids: a named row it holds is
 * replaced, a named row it lacks is dropped, a new row is appended, and
 * every other field is fresh's.
 */
export function splice<L extends Listed>(list: L, fresh: L, ids: readonly string[]): L {
  const named = new Set(ids);
  const now = new Map(fresh.rows.map((row) => [row.id, row]));
  const known = new Set(list.rows.map((row) => row.id));
  const kept = list.rows.flatMap((row) => {
    if (!named.has(row.id)) {
      return [row];
    }
    const next = now.get(row.id);
    return next === undefined ? [] : [next];
  });
  return { ...fresh, rows: [...kept, ...fresh.rows.filter((row) => !known.has(row.id))] };
}

const without = <L extends Listed>(list: L, gone: ReadonlySet<string>): L => ({
  ...list,
  rows: list.rows.filter((row) => !gone.has(row.id)),
});

export type SpliceSpec<L extends Listed> = {
  readonly queryKey: QueryKey;
  /** Reads the named rows, at most `chunk` of them. */
  readonly read: (ids: readonly string[]) => Promise<L>;
  /** Runs once each flush has landed, with every id it named. */
  readonly landed?: (queryClient: QueryClient, ids: readonly string[]) => void;
};

export type Splicer = {
  /** Reads the named rows again in the next frame's flush. */
  readonly queue: (queryClient: QueryClient, ids: readonly string[]) => void;
  /** Drops the named rows now, and from any answer still in flight. */
  readonly drop: (queryClient: QueryClient, ids: readonly string[]) => void;
};

const chunk = 200;

type Queue = {
  readonly ids: Set<string>;
  readonly gone: Set<string>;
  scheduled: boolean;
  running: boolean;
};

function chunked(ids: readonly string[]): string[][] {
  const chunks: string[][] = [];
  for (let start = 0; start < ids.length; start += chunk) {
    chunks.push(ids.slice(start, start + chunk));
  }
  return chunks;
}

function queueOf(queues: WeakMap<QueryClient, Queue>, queryClient: QueryClient): Queue {
  const known = queues.get(queryClient);
  if (known !== undefined) {
    return known;
  }
  const queue: Queue = { ids: new Set(), gone: new Set(), scheduled: false, running: false };
  queues.set(queryClient, queue);
  return queue;
}

async function flushOnce<L extends Listed>(
  { queryKey, read, landed }: SpliceSpec<L>,
  queryClient: QueryClient,
  queue: Queue,
  ids: readonly string[],
) {
  const state = queryClient.getQueryState<L>(queryKey);
  if (state?.data === undefined) {
    return;
  }
  if (state.fetchStatus === "fetching") {
    await refresh(queryClient, { queryKey, exact: true });
  } else {
    const answers = await Promise.all(
      chunked(ids).map(async (named) => ({ named, fresh: await read(named) })),
    );
    queryClient.setQueryData<L>(queryKey, (list) =>
      list === undefined
        ? list
        : without(
            answers.reduce((next, { named, fresh }) => splice(next, fresh, named), list),
            queue.gone,
          ),
    );
  }
  landed?.(queryClient, ids);
}

async function drain<L extends Listed>(
  spec: SpliceSpec<L>,
  queryClient: QueryClient,
  queue: Queue,
) {
  queue.running = true;
  try {
    while (queue.ids.size > 0) {
      const ids = [...queue.ids];
      queue.ids.clear();
      try {
        // oxlint-disable-next-line no-await-in-loop -- each flush lands before the next reads
        await flushOnce(spec, queryClient, queue, ids);
      } catch {
        // oxlint-disable-next-line no-await-in-loop -- a failed flush leaves the list unknown, so it is read whole
        await refresh(queryClient, { queryKey: spec.queryKey, exact: true });
      }
    }
  } finally {
    queue.running = false;
  }
}

/**
 * Keeps a cached list current from live events that name rows. Ids named
 * in one frame join one flush, which reads them in chunks of `chunk` and
 * splices every answer in at once. One flush runs at a time and ids named
 * meanwhile go to the next, so an older answer never lands over a newer
 * one. A flush while the whole list is loading waits for that load and
 * reads the whole list again.
 */
export function createSplicer<L extends Listed>(spec: SpliceSpec<L>): Splicer {
  const queues = new WeakMap<QueryClient, Queue>();
  return {
    queue: (queryClient, ids) => {
      const queue = queueOf(queues, queryClient);
      for (const id of ids) {
        if (!queue.gone.has(id)) {
          queue.ids.add(id);
        }
      }
      if (queue.scheduled || queue.running || queue.ids.size === 0) {
        return;
      }
      queue.scheduled = true;
      requestAnimationFrame(() => {
        queue.scheduled = false;
        void drain(spec, queryClient, queue);
      });
    },
    drop: (queryClient, ids) => {
      const queue = queueOf(queues, queryClient);
      for (const id of ids) {
        queue.gone.add(id);
        queue.ids.delete(id);
      }
      queryClient.setQueryData<L>(spec.queryKey, (list) =>
        list === undefined ? list : without(list, queue.gone),
      );
    },
  };
}
