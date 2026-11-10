import { type QueryClient, useQueries, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import type {
  FilesRenameDraft,
  FilesRenamePlan,
  FilesRenameRow,
  FilesRenameScope,
  FilesWorkLine,
} from "../api/gen/model";
import {
  getPlanWorkRenamesUrl,
  planRenames,
  planWorkRenames,
  renameFiles,
} from "../api/gen/query/files/files";
import { renameCounts } from "../live/library-effects";
import { refresh } from "../live/refresh";
import { useAttempt } from "../targets/attempt";
import {
  type RowsOf,
  type Ticks,
  confirmBody,
  everyTick,
  onlyTicked,
  rowsNeeded,
  tickedIn,
} from "./plan-model";

/** The save flow: the unsaved settings a plan was read under, that plan, and the save that comes first. */
export type Saving = {
  readonly draft: FilesRenameDraft;
  readonly plan: FilesRenamePlan;
  readonly store: () => Promise<void>;
};

/**
 * One look at a plan. A confirm that finds Works changed since the review
 * starts the next round, on the stored settings, with only those Works ticked.
 */
export type Round = {
  /** The sheet's own key and the round's number key its reads, so each round reads the plan afresh. */
  readonly sheet: string;
  readonly n: number;
  readonly draft?: FilesRenameDraft;
  readonly plan?: FilesRenamePlan;
  readonly store?: () => Promise<void>;
  readonly ticks: Ticks;
  readonly changed: number;
};

export const firstRound = (sheet: string, saving: Saving | undefined): Round => ({
  sheet,
  n: 0,
  ...saving,
  ticks: everyTick,
  changed: 0,
});

// A confirm reads the rows of Works left out whole a few at a time, so a large plan doesn't flood the server.
const rowsAtOnce = 6;

const rowsQuery = (round: Round, workId: string) => {
  const body = round.draft === undefined ? {} : { draft: round.draft };
  return {
    queryKey: [getPlanWorkRenamesUrl(workId), body, round.sheet, round.n],
    queryFn: ({ signal }: { signal: AbortSignal }) => planWorkRenames(workId, body, { signal }),
  };
};

async function readRows(queryClient: QueryClient, round: Round, ids: readonly string[]) {
  const read = new Map<string, readonly FilesRenameRow[]>();
  for (let at = 0; at < ids.length; at += rowsAtOnce) {
    // oxlint-disable-next-line no-await-in-loop -- a few at a time, as above
    const plans = await Promise.all(
      ids.slice(at, at + rowsAtOnce).map((id) => queryClient.query(rowsQuery(round, id))),
    );
    for (const plan of plans) {
      read.set(plan.workId, plan.rows);
    }
  }
  return read;
}

export type RowsState = { readonly pending: boolean; readonly error: unknown };

/**
 * The Admin's ticks and the Works they opened. A Work's rows are read when
 * it first opens and kept, so its ticks still count once it closes.
 */
export function useReview(round: Round, lines: readonly FilesWorkLine[], oneWork: boolean) {
  const [ticks, setTicks] = useState(round.ticks);
  const [opened, setOpened] = useState<ReadonlySet<string>>(
    () => new Set(oneWork ? lines.map((line) => line.workId) : []),
  );
  const [read, setRead] = useState(opened);
  const ids = [...read];
  const results = useQueries({ queries: ids.map((id) => rowsQuery(round, id)) });
  const rows = new Map<string, readonly FilesRenameRow[]>();
  const states = new Map<string, RowsState>();
  ids.forEach((id, at) => {
    const result = results[at];
    if (result?.data !== undefined) {
      rows.set(id, result.data.rows);
    }
    states.set(id, { pending: result?.isPending ?? true, error: result?.error });
  });
  return {
    ticks,
    setTicks,
    opened,
    rows: rows as RowsOf,
    stateOf: (workId: string): RowsState => states.get(workId) ?? { pending: true, error: null },
    setOpen: (workId: string, open: boolean) => {
      const next = new Set(opened);
      if (open) {
        next.add(workId);
        setRead(new Set([...read, workId]));
      } else {
        next.delete(workId);
      }
      setOpened(next);
    },
  };
}

/**
 * Confirming: the save first in the save flow, then the rename of each
 * ticked Work. Works that changed since the review start the next round.
 */
export function useConfirm({
  scope,
  round,
  lines,
  ticks,
  rows,
  onRound,
  onClose,
}: {
  scope: FilesRenameScope;
  round: Round;
  lines: readonly FilesWorkLine[];
  ticks: Ticks;
  rows: RowsOf;
  onRound: (round: Round) => void;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const confirming = useAttempt(async () => {
    const known = new Map([
      ...rows,
      ...(await readRows(queryClient, round, rowsNeeded(lines, ticks, rows))),
    ]);
    await round.store?.();
    const { changed } = await renameFiles({ works: confirmBody(lines, ticks, known) });
    void refresh(queryClient, renameCounts);
    const again = new Set(
      lines
        .filter((line) => changed.includes(line.workId) && tickedIn(ticks, line, known) > 0)
        .map((line) => line.workId),
    );
    if (again.size === 0) {
      onClose();
      return;
    }
    const plan = await planRenames({ scope });
    onRound({
      sheet: round.sheet,
      n: round.n + 1,
      plan,
      ticks: onlyTicked(plan.works, again),
      changed: again.size,
    });
  });
  const savingOnly = useAttempt(async () => {
    await round.store?.();
    onClose();
  });
  return { confirming, savingOnly };
}
