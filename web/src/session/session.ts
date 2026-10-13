import { type QueryClient, queryOptions, useSuspenseQuery } from "@tanstack/react-query";

import { isProblem } from "../api/errors";
import type { IdentityAccount, IdentityClaimed, IdentitySession } from "../api/gen/model";
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

export function useAccount(): IdentityAccount {
  const { data } = useSuspenseQuery(sessionQuery);
  if (data === null) {
    throw new Error(
      "useAccount runs only under routes/_app, whose guard puts the Account in the cache",
    );
  }
  return data.account;
}

export function markSignedIn(queryClient: QueryClient, session: IdentitySession): void {
  queryClient.setQueryData(sessionQuery.queryKey, session);
}

export function markClaimed(queryClient: QueryClient, claimed: IdentityClaimed): void {
  queryClient.setQueryData(statusQuery.queryKey, { claim: "controller" });
  const session: IdentitySession = { account: claimed.account };
  queryClient.setQueryData(sessionQuery.queryKey, session);
}
