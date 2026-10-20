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
  ActInputBody,
  ActOnTorrents400,
  ActOnTorrents401,
  ActOnTorrents403,
  ActOnTorrents409,
  ActOnTorrents422,
  ActOnTorrents500,
  DownloadclientsActed,
  DownloadclientsClientsList,
  DownloadclientsSettings,
  DownloadclientsSettingsPatch,
  DownloadclientsThresholds,
  DownloadclientsThresholdsPatch,
  DownloadclientsTorrentDetail,
  DownloadclientsTorrentList,
  GetDownloadClientSettings401,
  GetDownloadClientSettings403,
  GetDownloadClientSettings404,
  GetDownloadClientSettings409,
  GetDownloadClientSettings422,
  GetDownloadClientSettings500,
  GetTorrent401,
  GetTorrent403,
  GetTorrent404,
  GetTorrent409,
  GetTorrent422,
  GetTorrent500,
  ListDownloadClients401,
  ListDownloadClients403,
  ListDownloadClients409,
  ListDownloadClients500,
  ListTorrents401,
  ListTorrents403,
  ListTorrents409,
  ListTorrents422,
  ListTorrents500,
  ListTorrentsParams,
  SetDownloadClientSettings401,
  SetDownloadClientSettings403,
  SetDownloadClientSettings404,
  SetDownloadClientSettings409,
  SetDownloadClientSettings422,
  SetDownloadClientSettings500,
  SetDownloadClientThresholds401,
  SetDownloadClientThresholds403,
  SetDownloadClientThresholds409,
  SetDownloadClientThresholds422,
  SetDownloadClientThresholds500,
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

export const getListDownloadClientsUrl = () => {
  return `/api/v1/download-clients`;
};

/**
 * @summary List each downloading Node's built-in client, and the thresholds for a bad release
 */
export const listDownloadClients = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsClientsList> => {
  return apiFetch<DownloadclientsClientsList>(getListDownloadClientsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListDownloadClientsQueryKey = () => {
  return [`/api/v1/download-clients`] as const;
};

export const getListDownloadClientsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listDownloadClients>>,
  TError = ErrorType<
    | ListDownloadClients401
    | ListDownloadClients403
    | ListDownloadClients409
    | ListDownloadClients500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listDownloadClients>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListDownloadClientsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listDownloadClients>>> = ({ signal }) =>
    listDownloadClients({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listDownloadClients>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListDownloadClientsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listDownloadClients>>
>;
export type ListDownloadClientsSuspenseQueryError = ErrorType<
  ListDownloadClients401 | ListDownloadClients403 | ListDownloadClients409 | ListDownloadClients500
>;

export function useListDownloadClientsSuspense<
  TData = Awaited<ReturnType<typeof listDownloadClients>>,
  TError = ErrorType<
    | ListDownloadClients401
    | ListDownloadClients403
    | ListDownloadClients409
    | ListDownloadClients500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listDownloadClients>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListDownloadClientsSuspense<
  TData = Awaited<ReturnType<typeof listDownloadClients>>,
  TError = ErrorType<
    | ListDownloadClients401
    | ListDownloadClients403
    | ListDownloadClients409
    | ListDownloadClients500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listDownloadClients>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListDownloadClientsSuspense<
  TData = Awaited<ReturnType<typeof listDownloadClients>>,
  TError = ErrorType<
    | ListDownloadClients401
    | ListDownloadClients403
    | ListDownloadClients409
    | ListDownloadClients500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listDownloadClients>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List each downloading Node's built-in client, and the thresholds for a bad release
 */

export function useListDownloadClientsSuspense<
  TData = Awaited<ReturnType<typeof listDownloadClients>>,
  TError = ErrorType<
    | ListDownloadClients401
    | ListDownloadClients403
    | ListDownloadClients409
    | ListDownloadClients500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listDownloadClients>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListDownloadClientsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetDownloadClientThresholdsUrl = () => {
  return `/api/v1/download-clients/thresholds`;
};

/**
 * @summary Change when a release counts as bad, which every downloading Node applies at once
 */
export const setDownloadClientThresholds = async (
  downloadclientsThresholdsPatch: DownloadclientsThresholdsPatch,
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsThresholds> => {
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
  return apiFetch<DownloadclientsThresholds>(getSetDownloadClientThresholdsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(downloadclientsThresholdsPatch),
  });
};

export const getSetDownloadClientThresholdsMutationKey = () =>
  ["setDownloadClientThresholds"] as const;

export const getSetDownloadClientThresholdsMutationOptions = <
  TError = ErrorType<
    | SetDownloadClientThresholds401
    | SetDownloadClientThresholds403
    | SetDownloadClientThresholds409
    | SetDownloadClientThresholds422
    | SetDownloadClientThresholds500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setDownloadClientThresholds>>,
    TError,
    SetDownloadClientThresholdsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setDownloadClientThresholds>>,
  TError,
  SetDownloadClientThresholdsMutationVariables,
  TContext
> => {
  const mutationKey = getSetDownloadClientThresholdsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setDownloadClientThresholds>>,
    SetDownloadClientThresholdsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return setDownloadClientThresholds(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetDownloadClientThresholdsMutationResult = NonNullable<
  Awaited<ReturnType<typeof setDownloadClientThresholds>>
>;
export type SetDownloadClientThresholdsMutationBody = DownloadclientsThresholdsPatch;
export type SetDownloadClientThresholdsMutationError = ErrorType<
  | SetDownloadClientThresholds401
  | SetDownloadClientThresholds403
  | SetDownloadClientThresholds409
  | SetDownloadClientThresholds422
  | SetDownloadClientThresholds500
>;
export type SetDownloadClientThresholdsMutationVariables = { data: DownloadclientsThresholdsPatch };

/**
 * @summary Change when a release counts as bad, which every downloading Node applies at once
 */
export const useSetDownloadClientThresholds = <
  TError = ErrorType<
    | SetDownloadClientThresholds401
    | SetDownloadClientThresholds403
    | SetDownloadClientThresholds409
    | SetDownloadClientThresholds422
    | SetDownloadClientThresholds500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setDownloadClientThresholds>>,
      TError,
      SetDownloadClientThresholdsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setDownloadClientThresholds>>,
  TError,
  SetDownloadClientThresholdsMutationVariables,
  TContext
> => {
  return useMutation(getSetDownloadClientThresholdsMutationOptions(options), queryClient);
};
export const getGetDownloadClientSettingsUrl = (nodeId: string) => {
  return `/api/v1/download-clients/${nodeId}/settings`;
};

/**
 * @summary Get a Node's built-in torrent client settings
 */
export const getDownloadClientSettings = async (
  nodeId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsSettings> => {
  return apiFetch<DownloadclientsSettings>(getGetDownloadClientSettingsUrl(nodeId), {
    ...options,
    method: "GET",
  });
};

export const getGetDownloadClientSettingsQueryKey = (nodeId: string) => {
  return [`/api/v1/download-clients/${nodeId}/settings`] as const;
};

export const getGetDownloadClientSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getDownloadClientSettings>>,
  TError = ErrorType<
    | GetDownloadClientSettings401
    | GetDownloadClientSettings403
    | GetDownloadClientSettings404
    | GetDownloadClientSettings409
    | GetDownloadClientSettings422
    | GetDownloadClientSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDownloadClientSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetDownloadClientSettingsQueryKey(nodeId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getDownloadClientSettings>>> = ({
    signal,
  }) => getDownloadClientSettings(nodeId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof getDownloadClientSettings>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetDownloadClientSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getDownloadClientSettings>>
>;
export type GetDownloadClientSettingsSuspenseQueryError = ErrorType<
  | GetDownloadClientSettings401
  | GetDownloadClientSettings403
  | GetDownloadClientSettings404
  | GetDownloadClientSettings409
  | GetDownloadClientSettings422
  | GetDownloadClientSettings500
>;

export function useGetDownloadClientSettingsSuspense<
  TData = Awaited<ReturnType<typeof getDownloadClientSettings>>,
  TError = ErrorType<
    | GetDownloadClientSettings401
    | GetDownloadClientSettings403
    | GetDownloadClientSettings404
    | GetDownloadClientSettings409
    | GetDownloadClientSettings422
    | GetDownloadClientSettings500
  >,
>(
  nodeId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDownloadClientSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetDownloadClientSettingsSuspense<
  TData = Awaited<ReturnType<typeof getDownloadClientSettings>>,
  TError = ErrorType<
    | GetDownloadClientSettings401
    | GetDownloadClientSettings403
    | GetDownloadClientSettings404
    | GetDownloadClientSettings409
    | GetDownloadClientSettings422
    | GetDownloadClientSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDownloadClientSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetDownloadClientSettingsSuspense<
  TData = Awaited<ReturnType<typeof getDownloadClientSettings>>,
  TError = ErrorType<
    | GetDownloadClientSettings401
    | GetDownloadClientSettings403
    | GetDownloadClientSettings404
    | GetDownloadClientSettings409
    | GetDownloadClientSettings422
    | GetDownloadClientSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDownloadClientSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Node's built-in torrent client settings
 */

export function useGetDownloadClientSettingsSuspense<
  TData = Awaited<ReturnType<typeof getDownloadClientSettings>>,
  TError = ErrorType<
    | GetDownloadClientSettings401
    | GetDownloadClientSettings403
    | GetDownloadClientSettings404
    | GetDownloadClientSettings409
    | GetDownloadClientSettings422
    | GetDownloadClientSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDownloadClientSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetDownloadClientSettingsSuspenseQueryOptions(nodeId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetDownloadClientSettingsUrl = (nodeId: string) => {
  return `/api/v1/download-clients/${nodeId}/settings`;
};

/**
 * @summary Change a Node's built-in torrent client settings, which the Node applies at once
 */
export const setDownloadClientSettings = async (
  nodeId: string,
  downloadclientsSettingsPatch: DownloadclientsSettingsPatch,
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsSettings> => {
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
  return apiFetch<DownloadclientsSettings>(getSetDownloadClientSettingsUrl(nodeId), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(downloadclientsSettingsPatch),
  });
};

export const getSetDownloadClientSettingsMutationKey = () => ["setDownloadClientSettings"] as const;

export const getSetDownloadClientSettingsMutationOptions = <
  TError = ErrorType<
    | SetDownloadClientSettings401
    | SetDownloadClientSettings403
    | SetDownloadClientSettings404
    | SetDownloadClientSettings409
    | SetDownloadClientSettings422
    | SetDownloadClientSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setDownloadClientSettings>>,
    TError,
    SetDownloadClientSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setDownloadClientSettings>>,
  TError,
  SetDownloadClientSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getSetDownloadClientSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setDownloadClientSettings>>,
    SetDownloadClientSettingsMutationVariables
  > = (props) => {
    const { nodeId, data } = props ?? {};

    return setDownloadClientSettings(nodeId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetDownloadClientSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof setDownloadClientSettings>>
>;
export type SetDownloadClientSettingsMutationBody = DownloadclientsSettingsPatch;
export type SetDownloadClientSettingsMutationError = ErrorType<
  | SetDownloadClientSettings401
  | SetDownloadClientSettings403
  | SetDownloadClientSettings404
  | SetDownloadClientSettings409
  | SetDownloadClientSettings422
  | SetDownloadClientSettings500
>;
export type SetDownloadClientSettingsMutationVariables = {
  nodeId: string;
  data: DownloadclientsSettingsPatch;
};

/**
 * @summary Change a Node's built-in torrent client settings, which the Node applies at once
 */
export const useSetDownloadClientSettings = <
  TError = ErrorType<
    | SetDownloadClientSettings401
    | SetDownloadClientSettings403
    | SetDownloadClientSettings404
    | SetDownloadClientSettings409
    | SetDownloadClientSettings422
    | SetDownloadClientSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setDownloadClientSettings>>,
      TError,
      SetDownloadClientSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setDownloadClientSettings>>,
  TError,
  SetDownloadClientSettingsMutationVariables,
  TContext
> => {
  return useMutation(getSetDownloadClientSettingsMutationOptions(options), queryClient);
};
export const getListTorrentsUrl = (params?: ListTorrentsParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/torrents?${stringifiedParams}`
    : `/api/v1/torrents`;
};

/**
 * @summary List the Torrents every Node's built-in client holds, with their state and counts
 */
export const listTorrents = async (
  params?: ListTorrentsParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsTorrentList> => {
  return apiFetch<DownloadclientsTorrentList>(getListTorrentsUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListTorrentsQueryKey = (params?: ListTorrentsParams) => {
  return [`/api/v1/torrents`, ...(params ? [params] : [])] as const;
};

export const getListTorrentsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listTorrents>>,
  TError = ErrorType<
    ListTorrents401 | ListTorrents403 | ListTorrents409 | ListTorrents422 | ListTorrents500
  >,
>(
  params?: ListTorrentsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTorrents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListTorrentsQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listTorrents>>> = ({ signal }) =>
    listTorrents(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTorrents>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListTorrentsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listTorrents>>>;
export type ListTorrentsSuspenseQueryError = ErrorType<
  ListTorrents401 | ListTorrents403 | ListTorrents409 | ListTorrents422 | ListTorrents500
>;

export function useListTorrentsSuspense<
  TData = Awaited<ReturnType<typeof listTorrents>>,
  TError = ErrorType<
    ListTorrents401 | ListTorrents403 | ListTorrents409 | ListTorrents422 | ListTorrents500
  >,
>(
  params: undefined | ListTorrentsParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTorrents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTorrentsSuspense<
  TData = Awaited<ReturnType<typeof listTorrents>>,
  TError = ErrorType<
    ListTorrents401 | ListTorrents403 | ListTorrents409 | ListTorrents422 | ListTorrents500
  >,
>(
  params?: ListTorrentsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTorrents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTorrentsSuspense<
  TData = Awaited<ReturnType<typeof listTorrents>>,
  TError = ErrorType<
    ListTorrents401 | ListTorrents403 | ListTorrents409 | ListTorrents422 | ListTorrents500
  >,
>(
  params?: ListTorrentsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTorrents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Torrents every Node's built-in client holds, with their state and counts
 */

export function useListTorrentsSuspense<
  TData = Awaited<ReturnType<typeof listTorrents>>,
  TError = ErrorType<
    ListTorrents401 | ListTorrents403 | ListTorrents409 | ListTorrents422 | ListTorrents500
  >,
>(
  params?: ListTorrentsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTorrents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListTorrentsSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getActOnTorrentsUrl = () => {
  return `/api/v1/torrents/actions`;
};

/**
 * @summary Pause, resume, announce, recheck, mark failed or remove Torrents, answering for each
 */
export const actOnTorrents = async (
  actInputBody: ActInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsActed> => {
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
  return apiFetch<DownloadclientsActed>(getActOnTorrentsUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(actInputBody),
  });
};

export const getActOnTorrentsMutationKey = () => ["actOnTorrents"] as const;

export const getActOnTorrentsMutationOptions = <
  TError = ErrorType<
    | ActOnTorrents400
    | ActOnTorrents401
    | ActOnTorrents403
    | ActOnTorrents409
    | ActOnTorrents422
    | ActOnTorrents500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof actOnTorrents>>,
    TError,
    ActOnTorrentsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof actOnTorrents>>,
  TError,
  ActOnTorrentsMutationVariables,
  TContext
> => {
  const mutationKey = getActOnTorrentsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof actOnTorrents>>,
    ActOnTorrentsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return actOnTorrents(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ActOnTorrentsMutationResult = NonNullable<Awaited<ReturnType<typeof actOnTorrents>>>;
export type ActOnTorrentsMutationBody = ActInputBody;
export type ActOnTorrentsMutationError = ErrorType<
  | ActOnTorrents400
  | ActOnTorrents401
  | ActOnTorrents403
  | ActOnTorrents409
  | ActOnTorrents422
  | ActOnTorrents500
>;
export type ActOnTorrentsMutationVariables = { data: ActInputBody };

/**
 * @summary Pause, resume, announce, recheck, mark failed or remove Torrents, answering for each
 */
export const useActOnTorrents = <
  TError = ErrorType<
    | ActOnTorrents400
    | ActOnTorrents401
    | ActOnTorrents403
    | ActOnTorrents409
    | ActOnTorrents422
    | ActOnTorrents500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof actOnTorrents>>,
      TError,
      ActOnTorrentsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof actOnTorrents>>,
  TError,
  ActOnTorrentsMutationVariables,
  TContext
> => {
  return useMutation(getActOnTorrentsMutationOptions(options), queryClient);
};
export const getGetTorrentUrl = (id: string) => {
  return `/api/v1/torrents/${id}`;
};

/**
 * @summary Get a Torrent with its trackers, files and Grabs, live from its Node while it answers
 */
export const getTorrent = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsTorrentDetail> => {
  return apiFetch<DownloadclientsTorrentDetail>(getGetTorrentUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetTorrentQueryKey = (id: string) => {
  return [`/api/v1/torrents/${id}`] as const;
};

export const getGetTorrentSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getTorrent>>,
  TError = ErrorType<
    GetTorrent401 | GetTorrent403 | GetTorrent404 | GetTorrent409 | GetTorrent422 | GetTorrent500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTorrent>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetTorrentQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getTorrent>>> = ({ signal }) =>
    getTorrent(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTorrent>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetTorrentSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getTorrent>>>;
export type GetTorrentSuspenseQueryError = ErrorType<
  GetTorrent401 | GetTorrent403 | GetTorrent404 | GetTorrent409 | GetTorrent422 | GetTorrent500
>;

export function useGetTorrentSuspense<
  TData = Awaited<ReturnType<typeof getTorrent>>,
  TError = ErrorType<
    GetTorrent401 | GetTorrent403 | GetTorrent404 | GetTorrent409 | GetTorrent422 | GetTorrent500
  >,
>(
  id: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTorrent>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTorrentSuspense<
  TData = Awaited<ReturnType<typeof getTorrent>>,
  TError = ErrorType<
    GetTorrent401 | GetTorrent403 | GetTorrent404 | GetTorrent409 | GetTorrent422 | GetTorrent500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTorrent>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTorrentSuspense<
  TData = Awaited<ReturnType<typeof getTorrent>>,
  TError = ErrorType<
    GetTorrent401 | GetTorrent403 | GetTorrent404 | GetTorrent409 | GetTorrent422 | GetTorrent500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTorrent>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Torrent with its trackers, files and Grabs, live from its Node while it answers
 */

export function useGetTorrentSuspense<
  TData = Awaited<ReturnType<typeof getTorrent>>,
  TError = ErrorType<
    GetTorrent401 | GetTorrent403 | GetTorrent404 | GetTorrent409 | GetTorrent422 | GetTorrent500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTorrent>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetTorrentSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
