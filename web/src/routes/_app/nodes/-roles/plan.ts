import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../../../../api/errors";
import type { NodesAssignment, NodesOutcome } from "../../../../api/gen/model";
import { planRoles, setRoles } from "../../../../api/gen/query/nodes/nodes";
import { refusedOutcome } from "../../../../nodes/roles";

/** A change of Roles: the basis it was planned on and each Node's new Roles. */
export type Change = { readonly basis: string; readonly set: readonly NodesAssignment[] };

export type Ready = {
  readonly step: "planned" | "applying";
  readonly outcome: NodesOutcome;
  readonly stale: boolean;
  readonly failure?: string;
};

export type Review =
  | { readonly step: "planning" }
  | { readonly step: "unplanned"; readonly failure: string }
  | Ready;

/**
 * What a refusal of a change sent on basis turns the Review into: the fresh
 * Outcome it carries, or the reason it failed.
 */
function refused(was: Ready | undefined, basis: string, error: unknown): Review {
  const outcome = refusedOutcome(error);
  if (outcome !== undefined) {
    return { step: "planned", outcome, stale: outcome.basis !== basis };
  }
  return was === undefined
    ? { step: "unplanned", failure: sentenceOf(error) }
    : { ...was, step: "planned", failure: sentenceOf(error) };
}

async function plan(change: Change, signal: AbortSignal): Promise<Review> {
  try {
    const outcome = await planRoles(
      { basis: change.basis, set: [...change.set], test: true },
      { signal },
    );
    return { step: "planned", outcome, stale: false };
  } catch (error) {
    return refused(undefined, change.basis, error);
  }
}

/**
 * The Review of change: planned with reach tests when it opens, then
 * applied on the basis it was last planned on, so a change someone made
 * meanwhile comes back as a fresh Review rather than being undone.
 */
export function useReview(change: Change, onApplied: (outcome: NodesOutcome) => Promise<void>) {
  const planned = useQuery({
    queryKey: ["planRoles", change],
    queryFn: ({ signal }) => plan(change, signal),
    staleTime: Infinity,
    gcTime: 0,
  });
  const [after, setAfter] = useState<Review>();
  const review: Review = after ?? planned.data ?? { step: "planning" };
  const apply = async ({ outcome, stale }: Ready) => {
    const applying: Ready = { step: "applying", outcome, stale };
    setAfter(applying);
    try {
      await onApplied(await setRoles({ basis: outcome.basis, set: [...change.set] }));
    } catch (error) {
      setAfter(refused(applying, outcome.basis, error));
    }
  };
  return { review, apply };
}
