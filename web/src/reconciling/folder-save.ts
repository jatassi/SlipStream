import { useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";

import { sentenceOf } from "../api/errors";
import type { ReconcilingSavedRoot } from "../api/gen/model";
import { rehomeReconciling } from "../api/gen/query/reconciling/reconciling";
import { type Place, samePlace } from "../roots/use-place";
import { useCounts } from "../shell/counts";
import type { Row } from "./folder-standing";

type Save = { readonly row: Row; readonly forget?: number };

/**
 * Saves rows through the root edit's re-home. Once the step is done and the
 * shell's counts say so, which reconciling.changed brings, it opens the
 * Reconciling page.
 */
export function useSave(placeOf: (row: Row) => Place) {
  const navigate = useNavigate();
  const rehome = useCounts()?.reconciling?.rehome;
  const [answers, setAnswers] = useState<ReadonlyMap<string, ReconcilingSavedRoot>>(new Map());
  const [done, setDone] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  useEffect(() => {
    if (done && rehome !== true) {
      void navigate({ to: "/reconciling", replace: true });
    }
  }, [done, rehome, navigate]);
  const bodyOf = ({ row, forget }: Save) => {
    const place = placeOf(row);
    const moved = !samePlace(place, { nodeId: row.root.nodeId, path: row.root.path });
    return {
      rootId: row.root.id,
      ...(moved && { nodeId: place.nodeId, path: place.path }),
      ...(forget !== undefined && { forget }),
    };
  };
  const save = async (saves: readonly Save[], finish: boolean) => {
    setBusy(true);
    setFailure(undefined);
    try {
      const answer = await rehomeReconciling({
        roots: saves.map((one) => bodyOf(one)),
        done: finish,
      });
      setAnswers(
        (before) => new Map([...before, ...answer.roots.map((r) => [r.rootId, r] as const)]),
      );
      setDone(finish && answer.roots.every((r) => r.refused === null));
    } catch (error) {
      setFailure(sentenceOf(error));
    } finally {
      setBusy(false);
    }
  };
  const forgetAnswer = (rootId: string) => {
    setAnswers((before) => new Map([...before].filter(([id]) => id !== rootId)));
  };
  return { answers, busy, failure, save, forgetAnswer };
}
