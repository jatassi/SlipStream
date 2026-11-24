import { keepPreviousData, skipToken, useQuery } from "@tanstack/react-query";

import { isProblem } from "../api/errors";
import type { MovesMoveAsk, MovesMovePreview, MovesStartedMoves } from "../api/gen/model";
import { previewMoveBack, previewMoves, startMoves } from "../api/gen/query/moves/moves";

/** What a Move Dialog previews: Works to a root, or a finished batch back to the roots it left. */
export type Previewing =
  | { readonly kind: "ask"; readonly ask: MovesMoveAsk }
  | { readonly kind: "back"; readonly batchId: string };

export const changedWords =
  "The Move changed since its preview. Check the new preview, then start it.";

const fetchPreview = (previewing: Previewing, signal: AbortSignal) =>
  previewing.kind === "ask"
    ? previewMoves(previewing.ask, { signal })
    : previewMoveBack(previewing.batchId, { signal });

const expired = (plan: MovesMovePreview, now: number) => Date.parse(plan.expiresAt) <= now;

/**
 * The preview of a Move and its Start. A Start on an expired token previews
 * again first and starts the fresh plan when it moves as many units. A plan
 * that changed under the Admin, by the clock or by a 409 or 410, is shown
 * fresh in place and the Start throws, so the dialog stays open on it.
 */
export function useMovePreview(previewing: Previewing | undefined) {
  const preview = useQuery({
    queryKey: ["moves-preview", previewing],
    queryFn:
      previewing === undefined ? skipToken : ({ signal }) => fetchPreview(previewing, signal),
    gcTime: 0,
    staleTime: Number.POSITIVE_INFINITY,
    retry: false,
    placeholderData: keepPreviousData,
  });

  const fresh = async (): Promise<MovesMovePreview> => {
    const { data, error } = await preview.refetch({ throwOnError: false });
    if (data === undefined) {
      throw error ?? new Error("The preview didn't load. Open it again.");
    }
    return data;
  };

  const start = async (shown: MovesMovePreview): Promise<MovesStartedMoves> => {
    let plan = shown;
    if (expired(plan, Date.now())) {
      plan = await fresh();
      if (plan.movable !== shown.movable) {
        throw new Error(changedWords);
      }
    }
    try {
      return await startMoves({ token: plan.token }, { confirm: plan.movable });
    } catch (error) {
      if (isProblem(error, "move_plan_changed") || isProblem(error, "move_plan_expired")) {
        await fresh();
        throw new Error(changedWords, { cause: error });
      }
      throw error;
    }
  };

  return { preview, start };
}
