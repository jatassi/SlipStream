import { queryOptions as queryOptionsBuilder, useSuspenseQuery } from "@tanstack/react-query";
import type {
  DataTag,
  QueryClient,
  QueryFunction,
  QueryKey,
  UseSuspenseQueryOptions,
  UseSuspenseQueryResult,
} from "@tanstack/react-query";

import type {
  GetCounts401,
  GetCounts403,
  GetCounts409,
  GetCounts500,
  ShellCounts,
} from "../../model";

import { apiFetch } from "../../../fetch.ts";
import type { ErrorType } from "../../../fetch.ts";

type SecondParameter<T extends (...args: never) => unknown> = Parameters<T>[1];

const withQueryKey = <T extends object, K>(query: T, queryKey: K): T & { queryKey: K } => {
  const result = { queryKey } as T & { queryKey: K };
  for (const key of Object.keys(query)) {
    // The explicit queryKey always wins, matching the previous
    // `{ ...query, queryKey }` spread where it was set last.
    if (key === "queryKey") continue;
    Object.defineProperty(result, key, {
      enumerable: true,
      configurable: true,
      get: () => (query as Record<string, unknown>)[key],
    });
  }
  return result;
};

export const getGetCountsUrl = () => {
  return `/api/v1/counts`;
};

/**
 * @summary Get the counts the shell shows beside its navigation
 */
export const getCounts = async (options?: Parameters<typeof apiFetch>[1]): Promise<ShellCounts> => {
  return apiFetch<ShellCounts>(getGetCountsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetCountsQueryKey = () => {
  return [`/api/v1/counts`] as const;
};

export const getGetCountsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getCounts>>,
  TError = ErrorType<GetCounts401 | GetCounts403 | GetCounts409 | GetCounts500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCounts>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetCountsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getCounts>>> = ({ signal }) =>
    getCounts({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCounts>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetCountsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getCounts>>>;
export type GetCountsSuspenseQueryError = ErrorType<
  GetCounts401 | GetCounts403 | GetCounts409 | GetCounts500
>;

export function useGetCountsSuspense<
  TData = Awaited<ReturnType<typeof getCounts>>,
  TError = ErrorType<GetCounts401 | GetCounts403 | GetCounts409 | GetCounts500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCounts>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetCountsSuspense<
  TData = Awaited<ReturnType<typeof getCounts>>,
  TError = ErrorType<GetCounts401 | GetCounts403 | GetCounts409 | GetCounts500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCounts>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetCountsSuspense<
  TData = Awaited<ReturnType<typeof getCounts>>,
  TError = ErrorType<GetCounts401 | GetCounts403 | GetCounts409 | GetCounts500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCounts>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the counts the shell shows beside its navigation
 */

export function useGetCountsSuspense<
  TData = Awaited<ReturnType<typeof getCounts>>,
  TError = ErrorType<GetCounts401 | GetCounts403 | GetCounts409 | GetCounts500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCounts>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetCountsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
