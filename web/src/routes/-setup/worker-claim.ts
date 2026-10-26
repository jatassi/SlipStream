import { type UseQueryResult, queryOptions } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";

import { isApiError, isProblem } from "../../api/errors";
import type { PairingWorkerClaim } from "../../api/gen/model";
import { getGetWorkerClaimQueryKey, getWorkerClaim } from "../../api/gen/query/pairing/pairing";
import { type SetupCode, forgetSetupCode, setupCodeHeaders } from "../../session/setup-code";
import type { ChoiceId } from "./form-props";

/** How often the page asks whether the Controller has dialed, as Add a Node does. */
export const dialPoll = 2000;

/** This Node's Join, or null when no Join waits for a dial. */
export const workerClaimQuery = (code: SetupCode) =>
  queryOptions({
    queryKey: getGetWorkerClaimQueryKey(),
    queryFn: async ({ signal }): Promise<PairingWorkerClaim | null> => {
      try {
        return await getWorkerClaim({ signal, headers: setupCodeHeaders(code) });
      } catch (error) {
        if (isProblem(error, "no_join_pending")) {
          return null;
        }
        throw error;
      }
    },
    retry: false,
    refetchOnMount: "always",
    refetchInterval: (query) => (query.state.data?.state === "awaiting-dial" ? dialPoll : false),
  });

export type Screen = "choice" | ChoiceId;

/** Where an accepted code opens: a Join this Node is still in, or the choice. */
export function opening(claim: UseQueryResult<PairingWorkerClaim | null>): Screen | undefined {
  if (claim.isPending) {
    return undefined;
  }
  const state = claim.data?.state;
  if (state === "joined") {
    return undefined;
  }
  return state === "awaiting-dial" || state === "failed" ? "join" : "choice";
}

/** A Join that ends while the page reads it: joined, or the Node moved on without this tab. */
export function useJoinOutcome(
  claim: UseQueryResult<PairingWorkerClaim | null>,
  restarted: () => void,
): void {
  const navigate = useNavigate();
  const joined = claim.data?.state === "joined";
  const code = isApiError(claim.error) ? claim.error.code : undefined;
  useEffect(() => {
    if (joined) {
      forgetSetupCode();
      void navigate({ to: "/worker" });
    }
  }, [joined, navigate]);
  useEffect(() => {
    if (code === "setup_code_wrong") {
      restarted();
    } else if (code === "already_claimed") {
      void navigate({ to: "/" });
    }
  }, [code, navigate, restarted]);
}
