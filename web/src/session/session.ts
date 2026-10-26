import { type QueryClient, queryOptions, useSuspenseQuery } from "@tanstack/react-query";

import { isProblem } from "../api/errors";
import type {
  FirstrunFirstRun,
  FirstrunStepID,
  IdentityAccount,
  IdentitySession,
  IdentitySignedIn,
  SystemStatus,
} from "../api/gen/model";
import {
  finishFirstRun,
  getFirstRun,
  getGetFirstRunQueryKey,
  setFirstRunStep,
} from "../api/gen/query/firstrun/firstrun";
import { getGetSessionQueryKey, getSession } from "../api/gen/query/identity/identity";
import { getGetStatusQueryKey, getStatus } from "../api/gen/query/system/system";

export const sessionQuery = queryOptions({
  queryKey: getGetSessionQueryKey(),
  queryFn: async ({ signal }): Promise<IdentitySession | null> => {
    try {
      return await getSession({ signal });
    } catch (error) {
      if (isProblem(error, "signed_out") || isProblem(error, "worker_node")) {
        return null;
      }
      throw error;
    }
  },
});

export const statusQuery = queryOptions({
  queryKey: getGetStatusQueryKey(),
  queryFn: ({ signal }) => getStatus({ signal }),
  staleTime: 0,
});

export const firstRunQuery = queryOptions({
  queryKey: getGetFirstRunQueryKey(),
  queryFn: ({ signal }) => getFirstRun({ signal }),
});

export function useAccount(): IdentityAccount {
  const { data } = useSuspenseQuery(sessionQuery);
  if (data === null) {
    throw new Error(
      "useAccount runs only under routes/_app and routes/portal, whose guards put the Account in the cache",
    );
  }
  return data.account;
}

export function markSignedIn(queryClient: QueryClient, session: IdentitySession): void {
  queryClient.setQueryData(sessionQuery.queryKey, session);
}

export function markSignedOut(queryClient: QueryClient): void {
  queryClient.setQueryData(sessionQuery.queryKey, null);
}

export const refreshSession = (queryClient: QueryClient): Promise<IdentitySession | null> =>
  queryClient.query({ ...sessionQuery, staleTime: 0 });

export function markClaimed(queryClient: QueryClient, signedIn: IdentitySignedIn): void {
  queryClient.setQueryData(
    statusQuery.queryKey,
    (status): SystemStatus | undefined => status && { ...status, claim: "controller" },
  );
  markSignedIn(queryClient, signedIn);
}

/** A step write answers with first setup as it now stands, which the gate and the frame read. */
export function adoptFirstRun(queryClient: QueryClient, view: FirstrunFirstRun): void {
  queryClient.setQueryData(firstRunQuery.queryKey, view);
}

/**
 * Stores a step's outcome and answers true. Once another tab has finished
 * first setup, it reads the finished view instead and answers false, and the
 * page follows the finish into the app.
 */
export async function storeStep(
  queryClient: QueryClient,
  id: FirstrunStepID,
  outcome: "done" | "skipped",
): Promise<boolean> {
  try {
    adoptFirstRun(queryClient, await setFirstRunStep(id, { outcome }));
    return true;
  } catch (error) {
    if (!isProblem(error, "first_run_finished")) {
      throw error;
    }
    await queryClient.query({ ...firstRunQuery, staleTime: 0 });
    return false;
  }
}

/**
 * Finishes first setup, then marks the cached session finished, which the
 * gate reads as static, so the next route enters the app.
 */
export async function finishFirstSetup(queryClient: QueryClient): Promise<void> {
  await finishFirstRun();
  markFirstSetupFinished(queryClient);
}

export function markFirstSetupFinished(queryClient: QueryClient): void {
  queryClient.setQueryData(sessionQuery.queryKey, (session): IdentitySession | null | undefined =>
    session?.firstRun === "open" ? { ...session, firstRun: "finished" } : session,
  );
}
