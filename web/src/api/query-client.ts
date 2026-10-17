import {
  type DataTag,
  MutationCache,
  QueryCache,
  QueryClient,
  type QueryKey,
  type UseSuspenseQueryOptions,
} from "@tanstack/react-query";

import { UnreachableError, isProblem } from "./errors";

const unreachableRetries = 2;

export function createQueryClient(onSignedOut: () => void): QueryClient {
  const onError = (error: unknown) => {
    if (isProblem(error, "signed_out")) {
      onSignedOut();
    }
  };
  return new QueryClient({
    queryCache: new QueryCache({ onError }),
    mutationCache: new MutationCache({ onError }),
    defaultOptions: {
      queries: {
        staleTime: Infinity,
        refetchOnWindowFocus: false,
        refetchOnReconnect: false,
        retry: (failures, error) =>
          error instanceof UnreachableError && failures < unreachableRetries,
      },
      mutations: { retry: false },
    },
  });
}

type GeneratedQuery<T, E> = UseSuspenseQueryOptions<T, E> & {
  queryKey: DataTag<QueryKey, T, E>;
};

// Orval types its suspense options with TanStack's `queryFn?: … | undefined`,
// which queryClient.query refuses under exactOptionalPropertyTypes.
function readQuery<T, E>(
  queryClient: QueryClient,
  { queryKey, queryFn }: GeneratedQuery<T, E>,
  staleTime: 0 | "static",
): Promise<T> {
  return queryClient.query({
    queryKey,
    ...(queryFn === undefined ? {} : { queryFn }),
    staleTime,
  });
}

export const ensureQuery = <T, E>(queryClient: QueryClient, options: GeneratedQuery<T, E>) =>
  readQuery(queryClient, options, "static");

// Settings change with no live event, so a page that edits them reads them
// again each time it loads.
export const freshQuery = <T, E>(queryClient: QueryClient, options: GeneratedQuery<T, E>) =>
  readQuery(queryClient, options, 0);
