import { queryOptions as queryOptionsBuilder, useSuspenseQuery } from "@tanstack/react-query";
import type {
  DataTag,
  QueryClient,
  QueryFunction,
  QueryKey,
  UseSuspenseQueryOptions,
  UseSuspenseQueryResult,
} from "@tanstack/react-query";

import type { GetServing409, GetServing500, ServingWorker } from "../../model";

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

export const getGetServingUrl = () => {
  return `/api/v1/serving`;
};

/**
 * @summary Report this Worker's name, key and Controller, and what it holds and seeds. A Serving Worker responds to this itself
 */
export const getServing = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<ServingWorker> => {
  return apiFetch<ServingWorker>(getGetServingUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetServingQueryKey = () => {
  return [`/api/v1/serving`] as const;
};

export const getGetServingSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getServing>>,
  TError = ErrorType<GetServing409 | GetServing500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getServing>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetServingQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getServing>>> = ({ signal }) =>
    getServing({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getServing>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetServingSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getServing>>>;
export type GetServingSuspenseQueryError = ErrorType<GetServing409 | GetServing500>;

export function useGetServingSuspense<
  TData = Awaited<ReturnType<typeof getServing>>,
  TError = ErrorType<GetServing409 | GetServing500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getServing>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetServingSuspense<
  TData = Awaited<ReturnType<typeof getServing>>,
  TError = ErrorType<GetServing409 | GetServing500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getServing>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetServingSuspense<
  TData = Awaited<ReturnType<typeof getServing>>,
  TError = ErrorType<GetServing409 | GetServing500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getServing>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Report this Worker's name, key and Controller, and what it holds and seeds. A Serving Worker responds to this itself
 */

export function useGetServingSuspense<
  TData = Awaited<ReturnType<typeof getServing>>,
  TError = ErrorType<GetServing409 | GetServing500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getServing>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetServingSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
