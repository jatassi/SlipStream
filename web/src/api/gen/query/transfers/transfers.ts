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
  GetTransfer401,
  GetTransfer403,
  GetTransfer404,
  GetTransfer409,
  GetTransfer422,
  GetTransfer500,
  GetTransferSettings401,
  GetTransferSettings403,
  GetTransferSettings404,
  GetTransferSettings409,
  GetTransferSettings422,
  GetTransferSettings500,
  ListTransferSpeeds401,
  ListTransferSpeeds403,
  ListTransferSpeeds409,
  ListTransferSpeeds500,
  ListTransfers401,
  ListTransfers403,
  ListTransfers409,
  ListTransfers422,
  ListTransfers500,
  ListTransfersParams,
  SetTransferSettings401,
  SetTransferSettings403,
  SetTransferSettings404,
  SetTransferSettings409,
  SetTransferSettings422,
  SetTransferSettings500,
  TransfersSettings,
  TransfersSettingsPatch,
  TransfersSpeedList,
  TransfersTransfer,
  TransfersTransferList,
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

export const getListTransfersUrl = (params?: ListTransfersParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/transfers?${stringifiedParams}`
    : `/api/v1/transfers`;
};

/**
 * @summary List the Transfers between Nodes, with each one's state, progress and speed
 */
export const listTransfers = async (
  params?: ListTransfersParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TransfersTransferList> => {
  return apiFetch<TransfersTransferList>(getListTransfersUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListTransfersQueryKey = (params?: ListTransfersParams) => {
  return [`/api/v1/transfers`, ...(params ? [params] : [])] as const;
};

export const getListTransfersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listTransfers>>,
  TError = ErrorType<
    ListTransfers401 | ListTransfers403 | ListTransfers409 | ListTransfers422 | ListTransfers500
  >,
>(
  params?: ListTransfersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransfers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListTransfersQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listTransfers>>> = ({ signal }) =>
    listTransfers(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransfers>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListTransfersSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listTransfers>>
>;
export type ListTransfersSuspenseQueryError = ErrorType<
  ListTransfers401 | ListTransfers403 | ListTransfers409 | ListTransfers422 | ListTransfers500
>;

export function useListTransfersSuspense<
  TData = Awaited<ReturnType<typeof listTransfers>>,
  TError = ErrorType<
    ListTransfers401 | ListTransfers403 | ListTransfers409 | ListTransfers422 | ListTransfers500
  >,
>(
  params: undefined | ListTransfersParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransfers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTransfersSuspense<
  TData = Awaited<ReturnType<typeof listTransfers>>,
  TError = ErrorType<
    ListTransfers401 | ListTransfers403 | ListTransfers409 | ListTransfers422 | ListTransfers500
  >,
>(
  params?: ListTransfersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransfers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTransfersSuspense<
  TData = Awaited<ReturnType<typeof listTransfers>>,
  TError = ErrorType<
    ListTransfers401 | ListTransfers403 | ListTransfers409 | ListTransfers422 | ListTransfers500
  >,
>(
  params?: ListTransfersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransfers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Transfers between Nodes, with each one's state, progress and speed
 */

export function useListTransfersSuspense<
  TData = Awaited<ReturnType<typeof listTransfers>>,
  TError = ErrorType<
    ListTransfers401 | ListTransfers403 | ListTransfers409 | ListTransfers422 | ListTransfers500
  >,
>(
  params?: ListTransfersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransfers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListTransfersSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetTransferSettingsUrl = (nodeId: string) => {
  return `/api/v1/transfers/nodes/${nodeId}/settings`;
};

/**
 * @summary Get a Node's Transfer settings
 */
export const getTransferSettings = async (
  nodeId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TransfersSettings> => {
  return apiFetch<TransfersSettings>(getGetTransferSettingsUrl(nodeId), {
    ...options,
    method: "GET",
  });
};

export const getGetTransferSettingsQueryKey = (nodeId: string) => {
  return [`/api/v1/transfers/nodes/${nodeId}/settings`] as const;
};

export const getGetTransferSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getTransferSettings>>,
  TError = ErrorType<
    | GetTransferSettings401
    | GetTransferSettings403
    | GetTransferSettings404
    | GetTransferSettings409
    | GetTransferSettings422
    | GetTransferSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransferSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetTransferSettingsQueryKey(nodeId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getTransferSettings>>> = ({ signal }) =>
    getTransferSettings(nodeId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransferSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetTransferSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getTransferSettings>>
>;
export type GetTransferSettingsSuspenseQueryError = ErrorType<
  | GetTransferSettings401
  | GetTransferSettings403
  | GetTransferSettings404
  | GetTransferSettings409
  | GetTransferSettings422
  | GetTransferSettings500
>;

export function useGetTransferSettingsSuspense<
  TData = Awaited<ReturnType<typeof getTransferSettings>>,
  TError = ErrorType<
    | GetTransferSettings401
    | GetTransferSettings403
    | GetTransferSettings404
    | GetTransferSettings409
    | GetTransferSettings422
    | GetTransferSettings500
  >,
>(
  nodeId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransferSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTransferSettingsSuspense<
  TData = Awaited<ReturnType<typeof getTransferSettings>>,
  TError = ErrorType<
    | GetTransferSettings401
    | GetTransferSettings403
    | GetTransferSettings404
    | GetTransferSettings409
    | GetTransferSettings422
    | GetTransferSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransferSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTransferSettingsSuspense<
  TData = Awaited<ReturnType<typeof getTransferSettings>>,
  TError = ErrorType<
    | GetTransferSettings401
    | GetTransferSettings403
    | GetTransferSettings404
    | GetTransferSettings409
    | GetTransferSettings422
    | GetTransferSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransferSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Node's Transfer settings
 */

export function useGetTransferSettingsSuspense<
  TData = Awaited<ReturnType<typeof getTransferSettings>>,
  TError = ErrorType<
    | GetTransferSettings401
    | GetTransferSettings403
    | GetTransferSettings404
    | GetTransferSettings409
    | GetTransferSettings422
    | GetTransferSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransferSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetTransferSettingsSuspenseQueryOptions(nodeId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetTransferSettingsUrl = (nodeId: string) => {
  return `/api/v1/transfers/nodes/${nodeId}/settings`;
};

/**
 * @summary Change a Node's Transfer settings, which the Node applies at once
 */
export const setTransferSettings = async (
  nodeId: string,
  transfersSettingsPatch: TransfersSettingsPatch,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TransfersSettings> => {
  const getHeaders = (
    h?: NonNullable<RequestInit["headers"]>,
  ): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(
          h as Iterable<Iterable<string>>,
          (entry) => Array.from(entry) as [string, string],
        ),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
  return apiFetch<TransfersSettings>(getSetTransferSettingsUrl(nodeId), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(transfersSettingsPatch),
  });
};

export const getSetTransferSettingsMutationKey = () => ["setTransferSettings"] as const;

export const getSetTransferSettingsMutationOptions = <
  TError = ErrorType<
    | SetTransferSettings401
    | SetTransferSettings403
    | SetTransferSettings404
    | SetTransferSettings409
    | SetTransferSettings422
    | SetTransferSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setTransferSettings>>,
    TError,
    SetTransferSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setTransferSettings>>,
  TError,
  SetTransferSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getSetTransferSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setTransferSettings>>,
    SetTransferSettingsMutationVariables
  > = (props) => {
    const { nodeId, data } = props ?? {};

    return setTransferSettings(nodeId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetTransferSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof setTransferSettings>>
>;
export type SetTransferSettingsMutationBody = TransfersSettingsPatch;
export type SetTransferSettingsMutationError = ErrorType<
  | SetTransferSettings401
  | SetTransferSettings403
  | SetTransferSettings404
  | SetTransferSettings409
  | SetTransferSettings422
  | SetTransferSettings500
>;
export type SetTransferSettingsMutationVariables = { nodeId: string; data: TransfersSettingsPatch };

/**
 * @summary Change a Node's Transfer settings, which the Node applies at once
 */
export const useSetTransferSettings = <
  TError = ErrorType<
    | SetTransferSettings401
    | SetTransferSettings403
    | SetTransferSettings404
    | SetTransferSettings409
    | SetTransferSettings422
    | SetTransferSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setTransferSettings>>,
      TError,
      SetTransferSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setTransferSettings>>,
  TError,
  SetTransferSettingsMutationVariables,
  TContext
> => {
  return useMutation(getSetTransferSettingsMutationOptions(options), queryClient);
};
export const getListTransferSpeedsUrl = () => {
  return `/api/v1/transfers/speeds`;
};

/**
 * @summary List the recent speed and route of each pair of Nodes a Transfer moved bytes between
 */
export const listTransferSpeeds = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TransfersSpeedList> => {
  return apiFetch<TransfersSpeedList>(getListTransferSpeedsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListTransferSpeedsQueryKey = () => {
  return [`/api/v1/transfers/speeds`] as const;
};

export const getListTransferSpeedsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listTransferSpeeds>>,
  TError = ErrorType<
    ListTransferSpeeds401 | ListTransferSpeeds403 | ListTransferSpeeds409 | ListTransferSpeeds500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransferSpeeds>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListTransferSpeedsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listTransferSpeeds>>> = ({ signal }) =>
    listTransferSpeeds({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransferSpeeds>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListTransferSpeedsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listTransferSpeeds>>
>;
export type ListTransferSpeedsSuspenseQueryError = ErrorType<
  ListTransferSpeeds401 | ListTransferSpeeds403 | ListTransferSpeeds409 | ListTransferSpeeds500
>;

export function useListTransferSpeedsSuspense<
  TData = Awaited<ReturnType<typeof listTransferSpeeds>>,
  TError = ErrorType<
    ListTransferSpeeds401 | ListTransferSpeeds403 | ListTransferSpeeds409 | ListTransferSpeeds500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransferSpeeds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTransferSpeedsSuspense<
  TData = Awaited<ReturnType<typeof listTransferSpeeds>>,
  TError = ErrorType<
    ListTransferSpeeds401 | ListTransferSpeeds403 | ListTransferSpeeds409 | ListTransferSpeeds500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransferSpeeds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTransferSpeedsSuspense<
  TData = Awaited<ReturnType<typeof listTransferSpeeds>>,
  TError = ErrorType<
    ListTransferSpeeds401 | ListTransferSpeeds403 | ListTransferSpeeds409 | ListTransferSpeeds500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransferSpeeds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the recent speed and route of each pair of Nodes a Transfer moved bytes between
 */

export function useListTransferSpeedsSuspense<
  TData = Awaited<ReturnType<typeof listTransferSpeeds>>,
  TError = ErrorType<
    ListTransferSpeeds401 | ListTransferSpeeds403 | ListTransferSpeeds409 | ListTransferSpeeds500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTransferSpeeds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListTransferSpeedsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetTransferUrl = (id: string) => {
  return `/api/v1/transfers/${id}`;
};

/**
 * @summary Get a Transfer with its state, progress, speed and files
 */
export const getTransfer = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TransfersTransfer> => {
  return apiFetch<TransfersTransfer>(getGetTransferUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetTransferQueryKey = (id: string) => {
  return [`/api/v1/transfers/${id}`] as const;
};

export const getGetTransferSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getTransfer>>,
  TError = ErrorType<
    | GetTransfer401
    | GetTransfer403
    | GetTransfer404
    | GetTransfer409
    | GetTransfer422
    | GetTransfer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransfer>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetTransferQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getTransfer>>> = ({ signal }) =>
    getTransfer(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransfer>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetTransferSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getTransfer>>>;
export type GetTransferSuspenseQueryError = ErrorType<
  | GetTransfer401
  | GetTransfer403
  | GetTransfer404
  | GetTransfer409
  | GetTransfer422
  | GetTransfer500
>;

export function useGetTransferSuspense<
  TData = Awaited<ReturnType<typeof getTransfer>>,
  TError = ErrorType<
    | GetTransfer401
    | GetTransfer403
    | GetTransfer404
    | GetTransfer409
    | GetTransfer422
    | GetTransfer500
  >,
>(
  id: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransfer>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTransferSuspense<
  TData = Awaited<ReturnType<typeof getTransfer>>,
  TError = ErrorType<
    | GetTransfer401
    | GetTransfer403
    | GetTransfer404
    | GetTransfer409
    | GetTransfer422
    | GetTransfer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransfer>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTransferSuspense<
  TData = Awaited<ReturnType<typeof getTransfer>>,
  TError = ErrorType<
    | GetTransfer401
    | GetTransfer403
    | GetTransfer404
    | GetTransfer409
    | GetTransfer422
    | GetTransfer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransfer>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Transfer with its state, progress, speed and files
 */

export function useGetTransferSuspense<
  TData = Awaited<ReturnType<typeof getTransfer>>,
  TError = ErrorType<
    | GetTransfer401
    | GetTransfer403
    | GetTransfer404
    | GetTransfer409
    | GetTransfer422
    | GetTransfer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTransfer>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetTransferSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
