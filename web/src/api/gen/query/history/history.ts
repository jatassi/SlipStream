import {
  queryOptions as queryOptionsBuilder,
  useMutation,
  useSuspenseQuery,
} from "@tanstack/react-query";
import type {
  DataTag,
  MutationFunction,
  QueryClient,
  QueryFunction,
  QueryKey,
  UseMutationOptions,
  UseMutationResult,
  UseSuspenseQueryOptions,
  UseSuspenseQueryResult,
} from "@tanstack/react-query";

import type {
  ClearHistory401,
  ClearHistory403,
  ClearHistory409,
  ClearHistory500,
  GetHistoryEntry401,
  GetHistoryEntry403,
  GetHistoryEntry404,
  GetHistoryEntry409,
  GetHistoryEntry422,
  GetHistoryEntry500,
  GetHistoryEntryParams,
  GetHistorySummary401,
  GetHistorySummary403,
  GetHistorySummary409,
  GetHistorySummary422,
  GetHistorySummary500,
  GetHistorySummaryParams,
  HistoryKindsAnswer,
  HistoryListed,
  HistoryOneEntry,
  HistorySentence,
  ListHistory401,
  ListHistory403,
  ListHistory409,
  ListHistory422,
  ListHistory500,
  ListHistoryKinds401,
  ListHistoryKinds403,
  ListHistoryKinds409,
  ListHistoryKinds500,
  ListHistoryParams,
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

export const getClearHistoryUrl = () => {
  return `/api/v1/history`;
};

/**
 * @summary Delete every History entry
 */
export const clearHistory = async (options?: Parameters<typeof apiFetch>[1]): Promise<void> => {
  return apiFetch<void>(getClearHistoryUrl(), {
    ...options,
    method: "DELETE",
  });
};

export const getClearHistoryMutationKey = () => ["clearHistory"] as const;

export const getClearHistoryMutationOptions = <
  TError = ErrorType<ClearHistory401 | ClearHistory403 | ClearHistory409 | ClearHistory500>,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<Awaited<ReturnType<typeof clearHistory>>, TError, void, TContext>;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof clearHistory>>, TError, void, TContext> => {
  const mutationKey = getClearHistoryMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof clearHistory>>, void> = () => {
    return clearHistory(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ClearHistoryMutationResult = NonNullable<Awaited<ReturnType<typeof clearHistory>>>;

export type ClearHistoryMutationError = ErrorType<
  ClearHistory401 | ClearHistory403 | ClearHistory409 | ClearHistory500
>;

/**
 * @summary Delete every History entry
 */
export const useClearHistory = <
  TError = ErrorType<ClearHistory401 | ClearHistory403 | ClearHistory409 | ClearHistory500>,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<Awaited<ReturnType<typeof clearHistory>>, TError, void, TContext>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof clearHistory>>, TError, void, TContext> => {
  return useMutation(getClearHistoryMutationOptions(options), queryClient);
};
export const getListHistoryUrl = (params?: ListHistoryParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/history?${stringifiedParams}` : `/api/v1/history`;
};

/**
 * @summary List History entries, newest first
 */
export const listHistory = async (
  params?: ListHistoryParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<HistoryListed> => {
  return apiFetch<HistoryListed>(getListHistoryUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListHistoryQueryKey = (params?: ListHistoryParams) => {
  return [`/api/v1/history`, ...(params ? [params] : [])] as const;
};

export const getListHistorySuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listHistory>>,
  TError = ErrorType<
    ListHistory401 | ListHistory403 | ListHistory409 | ListHistory422 | ListHistory500
  >,
>(
  params?: ListHistoryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistory>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListHistoryQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listHistory>>> = ({ signal }) =>
    listHistory(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistory>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListHistorySuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listHistory>>>;
export type ListHistorySuspenseQueryError = ErrorType<
  ListHistory401 | ListHistory403 | ListHistory409 | ListHistory422 | ListHistory500
>;

export function useListHistorySuspense<
  TData = Awaited<ReturnType<typeof listHistory>>,
  TError = ErrorType<
    ListHistory401 | ListHistory403 | ListHistory409 | ListHistory422 | ListHistory500
  >,
>(
  params: undefined | ListHistoryParams,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistory>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListHistorySuspense<
  TData = Awaited<ReturnType<typeof listHistory>>,
  TError = ErrorType<
    ListHistory401 | ListHistory403 | ListHistory409 | ListHistory422 | ListHistory500
  >,
>(
  params?: ListHistoryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistory>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListHistorySuspense<
  TData = Awaited<ReturnType<typeof listHistory>>,
  TError = ErrorType<
    ListHistory401 | ListHistory403 | ListHistory409 | ListHistory422 | ListHistory500
  >,
>(
  params?: ListHistoryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistory>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List History entries, newest first
 */

export function useListHistorySuspense<
  TData = Awaited<ReturnType<typeof listHistory>>,
  TError = ErrorType<
    ListHistory401 | ListHistory403 | ListHistory409 | ListHistory422 | ListHistory500
  >,
>(
  params?: ListHistoryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistory>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListHistorySuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getListHistoryKindsUrl = () => {
  return `/api/v1/history/kinds`;
};

/**
 * @summary List History's groups and kinds
 */
export const listHistoryKinds = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<HistoryKindsAnswer> => {
  return apiFetch<HistoryKindsAnswer>(getListHistoryKindsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListHistoryKindsQueryKey = () => {
  return [`/api/v1/history/kinds`] as const;
};

export const getListHistoryKindsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listHistoryKinds>>,
  TError = ErrorType<
    ListHistoryKinds401 | ListHistoryKinds403 | ListHistoryKinds409 | ListHistoryKinds500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistoryKinds>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListHistoryKindsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listHistoryKinds>>> = ({ signal }) =>
    listHistoryKinds({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistoryKinds>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListHistoryKindsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listHistoryKinds>>
>;
export type ListHistoryKindsSuspenseQueryError = ErrorType<
  ListHistoryKinds401 | ListHistoryKinds403 | ListHistoryKinds409 | ListHistoryKinds500
>;

export function useListHistoryKindsSuspense<
  TData = Awaited<ReturnType<typeof listHistoryKinds>>,
  TError = ErrorType<
    ListHistoryKinds401 | ListHistoryKinds403 | ListHistoryKinds409 | ListHistoryKinds500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistoryKinds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListHistoryKindsSuspense<
  TData = Awaited<ReturnType<typeof listHistoryKinds>>,
  TError = ErrorType<
    ListHistoryKinds401 | ListHistoryKinds403 | ListHistoryKinds409 | ListHistoryKinds500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistoryKinds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListHistoryKindsSuspense<
  TData = Awaited<ReturnType<typeof listHistoryKinds>>,
  TError = ErrorType<
    ListHistoryKinds401 | ListHistoryKinds403 | ListHistoryKinds409 | ListHistoryKinds500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistoryKinds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List History's groups and kinds
 */

export function useListHistoryKindsSuspense<
  TData = Awaited<ReturnType<typeof listHistoryKinds>>,
  TError = ErrorType<
    ListHistoryKinds401 | ListHistoryKinds403 | ListHistoryKinds409 | ListHistoryKinds500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHistoryKinds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListHistoryKindsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetHistorySummaryUrl = (params: GetHistorySummaryParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/history/summary?${stringifiedParams}`
    : `/api/v1/history/summary`;
};

/**
 * @summary Count what happened since an instant
 */
export const getHistorySummary = async (
  params: GetHistorySummaryParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<HistorySentence | null> => {
  return apiFetch<HistorySentence | null>(getGetHistorySummaryUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getGetHistorySummaryQueryKey = (params?: GetHistorySummaryParams) => {
  return [`/api/v1/history/summary`, ...(params ? [params] : [])] as const;
};

export const getGetHistorySummarySuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getHistorySummary>>,
  TError = ErrorType<
    | GetHistorySummary401
    | GetHistorySummary403
    | GetHistorySummary409
    | GetHistorySummary422
    | GetHistorySummary500
  >,
>(
  params: GetHistorySummaryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistorySummary>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetHistorySummaryQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getHistorySummary>>> = ({ signal }) =>
    getHistorySummary(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistorySummary>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetHistorySummarySuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getHistorySummary>>
>;
export type GetHistorySummarySuspenseQueryError = ErrorType<
  | GetHistorySummary401
  | GetHistorySummary403
  | GetHistorySummary409
  | GetHistorySummary422
  | GetHistorySummary500
>;

export function useGetHistorySummarySuspense<
  TData = Awaited<ReturnType<typeof getHistorySummary>>,
  TError = ErrorType<
    | GetHistorySummary401
    | GetHistorySummary403
    | GetHistorySummary409
    | GetHistorySummary422
    | GetHistorySummary500
  >,
>(
  params: GetHistorySummaryParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistorySummary>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHistorySummarySuspense<
  TData = Awaited<ReturnType<typeof getHistorySummary>>,
  TError = ErrorType<
    | GetHistorySummary401
    | GetHistorySummary403
    | GetHistorySummary409
    | GetHistorySummary422
    | GetHistorySummary500
  >,
>(
  params: GetHistorySummaryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistorySummary>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHistorySummarySuspense<
  TData = Awaited<ReturnType<typeof getHistorySummary>>,
  TError = ErrorType<
    | GetHistorySummary401
    | GetHistorySummary403
    | GetHistorySummary409
    | GetHistorySummary422
    | GetHistorySummary500
  >,
>(
  params: GetHistorySummaryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistorySummary>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Count what happened since an instant
 */

export function useGetHistorySummarySuspense<
  TData = Awaited<ReturnType<typeof getHistorySummary>>,
  TError = ErrorType<
    | GetHistorySummary401
    | GetHistorySummary403
    | GetHistorySummary409
    | GetHistorySummary422
    | GetHistorySummary500
  >,
>(
  params: GetHistorySummaryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistorySummary>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetHistorySummarySuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetHistoryEntryUrl = (id: number, params: GetHistoryEntryParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/history/${id}?${stringifiedParams}`
    : `/api/v1/history/${id}`;
};

/**
 * @summary Read one History entry in full
 */
export const getHistoryEntry = async (
  id: number,
  params: GetHistoryEntryParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<HistoryOneEntry> => {
  return apiFetch<HistoryOneEntry>(getGetHistoryEntryUrl(id, params), {
    ...options,
    method: "GET",
  });
};

export const getGetHistoryEntryQueryKey = (id: number, params?: GetHistoryEntryParams) => {
  return [`/api/v1/history/${id}`, ...(params ? [params] : [])] as const;
};

export const getGetHistoryEntrySuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getHistoryEntry>>,
  TError = ErrorType<
    | GetHistoryEntry401
    | GetHistoryEntry403
    | GetHistoryEntry404
    | GetHistoryEntry409
    | GetHistoryEntry422
    | GetHistoryEntry500
  >,
>(
  id: number,
  params: GetHistoryEntryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistoryEntry>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetHistoryEntryQueryKey(id, params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getHistoryEntry>>> = ({ signal }) =>
    getHistoryEntry(id, params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistoryEntry>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetHistoryEntrySuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getHistoryEntry>>
>;
export type GetHistoryEntrySuspenseQueryError = ErrorType<
  | GetHistoryEntry401
  | GetHistoryEntry403
  | GetHistoryEntry404
  | GetHistoryEntry409
  | GetHistoryEntry422
  | GetHistoryEntry500
>;

export function useGetHistoryEntrySuspense<
  TData = Awaited<ReturnType<typeof getHistoryEntry>>,
  TError = ErrorType<
    | GetHistoryEntry401
    | GetHistoryEntry403
    | GetHistoryEntry404
    | GetHistoryEntry409
    | GetHistoryEntry422
    | GetHistoryEntry500
  >,
>(
  id: number,
  params: GetHistoryEntryParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistoryEntry>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHistoryEntrySuspense<
  TData = Awaited<ReturnType<typeof getHistoryEntry>>,
  TError = ErrorType<
    | GetHistoryEntry401
    | GetHistoryEntry403
    | GetHistoryEntry404
    | GetHistoryEntry409
    | GetHistoryEntry422
    | GetHistoryEntry500
  >,
>(
  id: number,
  params: GetHistoryEntryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistoryEntry>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHistoryEntrySuspense<
  TData = Awaited<ReturnType<typeof getHistoryEntry>>,
  TError = ErrorType<
    | GetHistoryEntry401
    | GetHistoryEntry403
    | GetHistoryEntry404
    | GetHistoryEntry409
    | GetHistoryEntry422
    | GetHistoryEntry500
  >,
>(
  id: number,
  params: GetHistoryEntryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistoryEntry>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Read one History entry in full
 */

export function useGetHistoryEntrySuspense<
  TData = Awaited<ReturnType<typeof getHistoryEntry>>,
  TError = ErrorType<
    | GetHistoryEntry401
    | GetHistoryEntry403
    | GetHistoryEntry404
    | GetHistoryEntry409
    | GetHistoryEntry422
    | GetHistoryEntry500
  >,
>(
  id: number,
  params: GetHistoryEntryParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHistoryEntry>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetHistoryEntrySuspenseQueryOptions(id, params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
