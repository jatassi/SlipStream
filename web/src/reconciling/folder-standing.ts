import { useQueries } from "@tanstack/react-query";

import type { ReconcilingRootRow, ReconcilingSavedRoot, RootsRoot } from "../api/gen/model";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useGetReconcilingSuspense } from "../api/gen/query/reconciling/reconciling";
import { useListRootsSuspense } from "../api/gen/query/roots/roots";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { verdictKey } from "../folders/verdict";
import { type RootVerdict, rootCheck } from "../roots/check";
import type { Place } from "../roots/use-place";
import { joinWords } from "./words";

/** One root the step finds: its Reconciling row and its library root. */
export type Row = { readonly step: ReconcilingRootRow; readonly root: RootsRoot };

/** What the step knows of a row: its verdict at the chosen place, or the last save's answer. */
export type Standing =
  | { readonly kind: "checking" }
  | { readonly kind: "found" }
  | { readonly kind: "refused"; readonly sentence: string; readonly missing: number | null };

function standingOf(
  verdict: RootVerdict | undefined,
  saved: ReconcilingSavedRoot | undefined,
): Standing {
  if (saved !== undefined && saved.refused !== null) {
    return { kind: "refused", sentence: saved.refused, missing: saved.missing };
  }
  if (verdict === undefined) {
    return { kind: "checking" };
  }
  return verdict.usable
    ? { kind: "found" }
    : { kind: "refused", sentence: verdict.sentence, missing: null };
}

export function useNodeNames(): (nodeId: string) => string {
  const { nodes } = useListNodesSuspense().data;
  return (nodeId) => nodes.find(({ id }) => id === nodeId)?.name ?? "";
}

/** Each row's standing: the root edit's check of its chosen folder, unless a save answered since. */
export function useStandings(
  rows: readonly Row[],
  placeOf: (row: Row) => Place,
  saved: ReadonlyMap<string, ReconcilingSavedRoot>,
): ReadonlyMap<string, Standing> {
  const { roots } = useListRootsSuspense().data;
  const { contentTypes } = useListContentTypesSuspense().data;
  const names = useNodeNames();
  const verdicts = useQueries({
    queries: rows.map((row) => {
      const place = placeOf(row);
      const namer = { node: names(place.nodeId), roots, types: contentTypes };
      const check = rootCheck(namer, row.root.id);
      return {
        queryKey: verdictKey(check, place.nodeId, place.path),
        queryFn: ({ signal }: { signal: AbortSignal }) =>
          check.judge({ nodeId: place.nodeId, path: place.path, signal }),
        staleTime: 0,
      };
    }),
  });
  return new Map(
    rows.map((row, index) => [
      row.root.id,
      standingOf(verdicts[index]?.data, saved.get(row.root.id)),
    ]),
  );
}

const folderWords = (count: number) => (count === 1 ? "1 folder" : `${count} folders`);

/** The step's lead: how many folders are where the Backup had them, and which need a path. */
export function leadOf(rows: readonly Row[], standings: ReadonlyMap<string, Standing>): string {
  const kinds = rows.map((row) => standings.get(row.root.id)?.kind);
  const found = kinds.filter((kind) => kind === "found").length;
  const failing = rows.filter((row) => standings.get(row.root.id)?.kind === "refused");
  if (rows.length === 0) {
    return "Every folder is saved.";
  }
  if (found === rows.length) {
    return `${rows.length === 1 ? "The folder is" : `All ${folderWords(rows.length)} are`} at the same path on this machine. Save to start the scans.`;
  }
  const where = `${found} of ${folderWords(rows.length)} ${rows.length === 1 ? "is" : "are"} at the same path on this machine.`;
  if (failing.length === 0) {
    return where;
  }
  const names = joinWords(failing.map((row) => `the ${row.step.label}`));
  return `${where} Give a path for ${names}, or leave ${failing.length === 1 ? "it" : "them"} for later.`;
}

/** The roots the step still finds: not saved and not continued without. */
export function useRows(): Row[] {
  const { roots: steps } = useGetReconcilingSuspense().data;
  const { roots } = useListRootsSuspense().data;
  return steps
    .filter((step) => !step.saved && !step.continued)
    .flatMap((step) => {
      const root = roots.find(({ id }) => id === step.id);
      return root === undefined ? [] : [{ step, root }];
    });
}
