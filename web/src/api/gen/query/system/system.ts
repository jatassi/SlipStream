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
  GetHealth500,
  GetStatus500,
  GetSystem401,
  GetSystem403,
  GetSystem500,
  SystemHealth,
  SystemReport,
  SystemStatus,
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

export const getGetHealthUrl = () => {
  return `/api/v1/health`;
};

/**
 * @summary Report that the Node is serving
 */
export const getHealth = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<SystemHealth> => {
  return apiFetch<SystemHealth>(getGetHealthUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetHealthQueryKey = () => {
  return [`/api/v1/health`] as const;
};

export const getGetHealthSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getHealth>>,
  TError = ErrorType<GetHealth500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetHealthQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getHealth>>> = ({ signal }) =>
    getHealth({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetHealthSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getHealth>>>;
export type GetHealthSuspenseQueryError = ErrorType<GetHealth500>;

export function useGetHealthSuspense<
  TData = Awaited<ReturnType<typeof getHealth>>,
  TError = ErrorType<GetHealth500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHealthSuspense<
  TData = Awaited<ReturnType<typeof getHealth>>,
  TError = ErrorType<GetHealth500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHealthSuspense<
  TData = Awaited<ReturnType<typeof getHealth>>,
  TError = ErrorType<GetHealth500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Report that the Node is serving
 */

export function useGetHealthSuspense<
  TData = Awaited<ReturnType<typeof getHealth>>,
  TError = ErrorType<GetHealth500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetHealthSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetStatusUrl = () => {
  return `/api/v1/status`;
};

/**
 * @summary Report whether the Node is claimed
 */
export const getStatus = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<SystemStatus> => {
  return apiFetch<SystemStatus>(getGetStatusUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetStatusQueryKey = () => {
  return [`/api/v1/status`] as const;
};

export const getGetStatusSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getStatus>>,
  TError = ErrorType<GetStatus500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetStatusQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getStatus>>> = ({ signal }) =>
    getStatus({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetStatusSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getStatus>>>;
export type GetStatusSuspenseQueryError = ErrorType<GetStatus500>;

export function useGetStatusSuspense<
  TData = Awaited<ReturnType<typeof getStatus>>,
  TError = ErrorType<GetStatus500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetStatusSuspense<
  TData = Awaited<ReturnType<typeof getStatus>>,
  TError = ErrorType<GetStatus500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetStatusSuspense<
  TData = Awaited<ReturnType<typeof getStatus>>,
  TError = ErrorType<GetStatus500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Report whether the Node is claimed
 */

export function useGetStatusSuspense<
  TData = Awaited<ReturnType<typeof getStatus>>,
  TError = ErrorType<GetStatus500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetStatusSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetSystemUrl = () => {
  return `/api/v1/system`;
};

/**
 * @summary Report the build, data dir, process and listeners
 */
export const getSystem = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<SystemReport> => {
  return apiFetch<SystemReport>(getGetSystemUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetSystemQueryKey = () => {
  return [`/api/v1/system`] as const;
};

export const getGetSystemSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSystem>>,
  TError = ErrorType<GetSystem401 | GetSystem403 | GetSystem500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSystemQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSystem>>> = ({ signal }) =>
    getSystem({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSystemSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getSystem>>>;
export type GetSystemSuspenseQueryError = ErrorType<GetSystem401 | GetSystem403 | GetSystem500>;

export function useGetSystemSuspense<
  TData = Awaited<ReturnType<typeof getSystem>>,
  TError = ErrorType<GetSystem401 | GetSystem403 | GetSystem500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSystemSuspense<
  TData = Awaited<ReturnType<typeof getSystem>>,
  TError = ErrorType<GetSystem401 | GetSystem403 | GetSystem500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSystemSuspense<
  TData = Awaited<ReturnType<typeof getSystem>>,
  TError = ErrorType<GetSystem401 | GetSystem403 | GetSystem500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Report the build, data dir, process and listeners
 */

export function useGetSystemSuspense<
  TData = Awaited<ReturnType<typeof getSystem>>,
  TError = ErrorType<GetSystem401 | GetSystem403 | GetSystem500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSystemSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
