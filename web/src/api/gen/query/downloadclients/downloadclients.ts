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
  CreateDownloadClient401,
  CreateDownloadClient403,
  CreateDownloadClient409,
  CreateDownloadClient422,
  CreateDownloadClient500,
  CreateDownloadClient502,
  DeleteDownloadClient401,
  DeleteDownloadClient403,
  DeleteDownloadClient404,
  DeleteDownloadClient409,
  DeleteDownloadClient422,
  DeleteDownloadClient500,
  DeleteDownloadClientParams,
  DownloadclientsActed,
  DownloadclientsClientEdit,
  DownloadclientsClientsList,
  DownloadclientsExternalClient,
  DownloadclientsNewClient,
  DownloadclientsSettings,
  DownloadclientsSettingsPatch,
  DownloadclientsTestReport,
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
  SetDownloadClientThresholds501,
  TestDownloadClient401,
  TestDownloadClient403,
  TestDownloadClient404,
  TestDownloadClient409,
  TestDownloadClient422,
  TestDownloadClient500,
  TestDownloadClient502,
  TestDownloadClientDraft401,
  TestDownloadClientDraft403,
  TestDownloadClientDraft409,
  TestDownloadClientDraft422,
  TestDownloadClientDraft500,
  TestDownloadClientDraft502,
  UpdateDownloadClient401,
  UpdateDownloadClient403,
  UpdateDownloadClient404,
  UpdateDownloadClient409,
  UpdateDownloadClient422,
  UpdateDownloadClient500,
  UpdateDownloadClient502,
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
 * @summary List each downloading Node's built-in and external clients, the kinds of external client, and the thresholds for a bad release
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
 * @summary List each downloading Node's built-in and external clients, the kinds of external client, and the thresholds for a bad release
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
    | SetDownloadClientThresholds501
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
  | SetDownloadClientThresholds501
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
    | SetDownloadClientThresholds501
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
 * @summary Get a Node's built-in torrent client settings and where its new Grabs go
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
 * @summary Get a Node's built-in torrent client settings and where its new Grabs go
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
 * @summary Change a Node's built-in torrent client settings and where its new Grabs go, which the Node applies at once
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
 * @summary Change a Node's built-in torrent client settings and where its new Grabs go, which the Node applies at once
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
export const getCreateDownloadClientUrl = () => {
  return `/api/v1/external-download-clients`;
};

/**
 * @summary Add an external Download Client once its Test passed on its Node
 */
export const createDownloadClient = async (
  downloadclientsNewClient: DownloadclientsNewClient,
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsExternalClient> => {
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
  return apiFetch<DownloadclientsExternalClient>(getCreateDownloadClientUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(downloadclientsNewClient),
  });
};

export const getCreateDownloadClientMutationKey = () => ["createDownloadClient"] as const;

export const getCreateDownloadClientMutationOptions = <
  TError = ErrorType<
    | CreateDownloadClient401
    | CreateDownloadClient403
    | CreateDownloadClient409
    | CreateDownloadClient422
    | CreateDownloadClient500
    | CreateDownloadClient502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof createDownloadClient>>,
    TError,
    CreateDownloadClientMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof createDownloadClient>>,
  TError,
  CreateDownloadClientMutationVariables,
  TContext
> => {
  const mutationKey = getCreateDownloadClientMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof createDownloadClient>>,
    CreateDownloadClientMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return createDownloadClient(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateDownloadClientMutationResult = NonNullable<
  Awaited<ReturnType<typeof createDownloadClient>>
>;
export type CreateDownloadClientMutationBody = DownloadclientsNewClient;
export type CreateDownloadClientMutationError = ErrorType<
  | CreateDownloadClient401
  | CreateDownloadClient403
  | CreateDownloadClient409
  | CreateDownloadClient422
  | CreateDownloadClient500
  | CreateDownloadClient502
>;
export type CreateDownloadClientMutationVariables = { data: DownloadclientsNewClient };

/**
 * @summary Add an external Download Client once its Test passed on its Node
 */
export const useCreateDownloadClient = <
  TError = ErrorType<
    | CreateDownloadClient401
    | CreateDownloadClient403
    | CreateDownloadClient409
    | CreateDownloadClient422
    | CreateDownloadClient500
    | CreateDownloadClient502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createDownloadClient>>,
      TError,
      CreateDownloadClientMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof createDownloadClient>>,
  TError,
  CreateDownloadClientMutationVariables,
  TContext
> => {
  return useMutation(getCreateDownloadClientMutationOptions(options), queryClient);
};
export const getTestDownloadClientDraftUrl = () => {
  return `/api/v1/external-download-clients/test`;
};

/**
 * @summary Test an external Download Client from its Node without saving it
 */
export const testDownloadClientDraft = async (
  downloadclientsNewClient: DownloadclientsNewClient,
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsTestReport> => {
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
  return apiFetch<DownloadclientsTestReport>(getTestDownloadClientDraftUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(downloadclientsNewClient),
  });
};

export const getTestDownloadClientDraftMutationKey = () => ["testDownloadClientDraft"] as const;

export const getTestDownloadClientDraftMutationOptions = <
  TError = ErrorType<
    | TestDownloadClientDraft401
    | TestDownloadClientDraft403
    | TestDownloadClientDraft409
    | TestDownloadClientDraft422
    | TestDownloadClientDraft500
    | TestDownloadClientDraft502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof testDownloadClientDraft>>,
    TError,
    TestDownloadClientDraftMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof testDownloadClientDraft>>,
  TError,
  TestDownloadClientDraftMutationVariables,
  TContext
> => {
  const mutationKey = getTestDownloadClientDraftMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof testDownloadClientDraft>>,
    TestDownloadClientDraftMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return testDownloadClientDraft(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type TestDownloadClientDraftMutationResult = NonNullable<
  Awaited<ReturnType<typeof testDownloadClientDraft>>
>;
export type TestDownloadClientDraftMutationBody = DownloadclientsNewClient;
export type TestDownloadClientDraftMutationError = ErrorType<
  | TestDownloadClientDraft401
  | TestDownloadClientDraft403
  | TestDownloadClientDraft409
  | TestDownloadClientDraft422
  | TestDownloadClientDraft500
  | TestDownloadClientDraft502
>;
export type TestDownloadClientDraftMutationVariables = { data: DownloadclientsNewClient };

/**
 * @summary Test an external Download Client from its Node without saving it
 */
export const useTestDownloadClientDraft = <
  TError = ErrorType<
    | TestDownloadClientDraft401
    | TestDownloadClientDraft403
    | TestDownloadClientDraft409
    | TestDownloadClientDraft422
    | TestDownloadClientDraft500
    | TestDownloadClientDraft502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof testDownloadClientDraft>>,
      TError,
      TestDownloadClientDraftMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof testDownloadClientDraft>>,
  TError,
  TestDownloadClientDraftMutationVariables,
  TContext
> => {
  return useMutation(getTestDownloadClientDraftMutationOptions(options), queryClient);
};
export const getDeleteDownloadClientUrl = (id: string, params?: DeleteDownloadClientParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/external-download-clients/${id}?${stringifiedParams}`
    : `/api/v1/external-download-clients/${id}`;
};

/**
 * @summary Delete an external Download Client, and with stopTracking stop tracking the torrents it holds
 */
export const deleteDownloadClient = async (
  id: string,
  params?: DeleteDownloadClientParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeleteDownloadClientUrl(id, params), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteDownloadClientMutationKey = () => ["deleteDownloadClient"] as const;

export const getDeleteDownloadClientMutationOptions = <
  TError = ErrorType<
    | DeleteDownloadClient401
    | DeleteDownloadClient403
    | DeleteDownloadClient404
    | DeleteDownloadClient409
    | DeleteDownloadClient422
    | DeleteDownloadClient500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteDownloadClient>>,
    TError,
    DeleteDownloadClientMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteDownloadClient>>,
  TError,
  DeleteDownloadClientMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteDownloadClientMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteDownloadClient>>,
    DeleteDownloadClientMutationVariables
  > = (props) => {
    const { id, params } = props ?? {};

    return deleteDownloadClient(id, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteDownloadClientMutationResult = NonNullable<
  Awaited<ReturnType<typeof deleteDownloadClient>>
>;

export type DeleteDownloadClientMutationError = ErrorType<
  | DeleteDownloadClient401
  | DeleteDownloadClient403
  | DeleteDownloadClient404
  | DeleteDownloadClient409
  | DeleteDownloadClient422
  | DeleteDownloadClient500
>;
export type DeleteDownloadClientMutationVariables = {
  id: string;
  params?: DeleteDownloadClientParams;
};

/**
 * @summary Delete an external Download Client, and with stopTracking stop tracking the torrents it holds
 */
export const useDeleteDownloadClient = <
  TError = ErrorType<
    | DeleteDownloadClient401
    | DeleteDownloadClient403
    | DeleteDownloadClient404
    | DeleteDownloadClient409
    | DeleteDownloadClient422
    | DeleteDownloadClient500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteDownloadClient>>,
      TError,
      DeleteDownloadClientMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteDownloadClient>>,
  TError,
  DeleteDownloadClientMutationVariables,
  TContext
> => {
  return useMutation(getDeleteDownloadClientMutationOptions(options), queryClient);
};
export const getUpdateDownloadClientUrl = (id: string) => {
  return `/api/v1/external-download-clients/${id}`;
};

/**
 * @summary Change an external Download Client, testing it first when the change is to its Node, kind, address, sign-in, daemon, label, folder or Client folders
 */
export const updateDownloadClient = async (
  id: string,
  downloadclientsClientEdit: DownloadclientsClientEdit,
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsExternalClient> => {
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
  return apiFetch<DownloadclientsExternalClient>(getUpdateDownloadClientUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(downloadclientsClientEdit),
  });
};

export const getUpdateDownloadClientMutationKey = () => ["updateDownloadClient"] as const;

export const getUpdateDownloadClientMutationOptions = <
  TError = ErrorType<
    | UpdateDownloadClient401
    | UpdateDownloadClient403
    | UpdateDownloadClient404
    | UpdateDownloadClient409
    | UpdateDownloadClient422
    | UpdateDownloadClient500
    | UpdateDownloadClient502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateDownloadClient>>,
    TError,
    UpdateDownloadClientMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateDownloadClient>>,
  TError,
  UpdateDownloadClientMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateDownloadClientMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateDownloadClient>>,
    UpdateDownloadClientMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return updateDownloadClient(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateDownloadClientMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateDownloadClient>>
>;
export type UpdateDownloadClientMutationBody = DownloadclientsClientEdit;
export type UpdateDownloadClientMutationError = ErrorType<
  | UpdateDownloadClient401
  | UpdateDownloadClient403
  | UpdateDownloadClient404
  | UpdateDownloadClient409
  | UpdateDownloadClient422
  | UpdateDownloadClient500
  | UpdateDownloadClient502
>;
export type UpdateDownloadClientMutationVariables = { id: string; data: DownloadclientsClientEdit };

/**
 * @summary Change an external Download Client, testing it first when the change is to its Node, kind, address, sign-in, daemon, label, folder or Client folders
 */
export const useUpdateDownloadClient = <
  TError = ErrorType<
    | UpdateDownloadClient401
    | UpdateDownloadClient403
    | UpdateDownloadClient404
    | UpdateDownloadClient409
    | UpdateDownloadClient422
    | UpdateDownloadClient500
    | UpdateDownloadClient502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateDownloadClient>>,
      TError,
      UpdateDownloadClientMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateDownloadClient>>,
  TError,
  UpdateDownloadClientMutationVariables,
  TContext
> => {
  return useMutation(getUpdateDownloadClientMutationOptions(options), queryClient);
};
export const getTestDownloadClientUrl = (id: string) => {
  return `/api/v1/external-download-clients/${id}/test`;
};

/**
 * @summary Test a saved external Download Client with unsaved changes, using its saved secrets where the changes leave them
 */
export const testDownloadClient = async (
  id: string,
  downloadclientsClientEdit?: DownloadclientsClientEdit,
  options?: Parameters<typeof apiFetch>[1],
): Promise<DownloadclientsTestReport> => {
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
  return apiFetch<DownloadclientsTestReport>(getTestDownloadClientUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(downloadclientsClientEdit),
  });
};

export const getTestDownloadClientMutationKey = () => ["testDownloadClient"] as const;

export const getTestDownloadClientMutationOptions = <
  TError = ErrorType<
    | TestDownloadClient401
    | TestDownloadClient403
    | TestDownloadClient404
    | TestDownloadClient409
    | TestDownloadClient422
    | TestDownloadClient500
    | TestDownloadClient502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof testDownloadClient>>,
    TError,
    TestDownloadClientMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof testDownloadClient>>,
  TError,
  TestDownloadClientMutationVariables,
  TContext
> => {
  const mutationKey = getTestDownloadClientMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof testDownloadClient>>,
    TestDownloadClientMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return testDownloadClient(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type TestDownloadClientMutationResult = NonNullable<
  Awaited<ReturnType<typeof testDownloadClient>>
>;
export type TestDownloadClientMutationBody = DownloadclientsClientEdit | undefined;
export type TestDownloadClientMutationError = ErrorType<
  | TestDownloadClient401
  | TestDownloadClient403
  | TestDownloadClient404
  | TestDownloadClient409
  | TestDownloadClient422
  | TestDownloadClient500
  | TestDownloadClient502
>;
export type TestDownloadClientMutationVariables = { id: string; data?: DownloadclientsClientEdit };

/**
 * @summary Test a saved external Download Client with unsaved changes, using its saved secrets where the changes leave them
 */
export const useTestDownloadClient = <
  TError = ErrorType<
    | TestDownloadClient401
    | TestDownloadClient403
    | TestDownloadClient404
    | TestDownloadClient409
    | TestDownloadClient422
    | TestDownloadClient500
    | TestDownloadClient502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof testDownloadClient>>,
      TError,
      TestDownloadClientMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof testDownloadClient>>,
  TError,
  TestDownloadClientMutationVariables,
  TContext
> => {
  return useMutation(getTestDownloadClientMutationOptions(options), queryClient);
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
 * @summary List the Torrents every Node's Download Clients hold, with their state and counts
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
 * @summary List the Torrents every Node's Download Clients hold, with their state and counts
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
 * @summary Pause, resume, announce, recheck, mark failed or remove Torrents, and return the result for each
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
 * @summary Pause, resume, announce, recheck, mark failed or remove Torrents, and return the result for each
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
 * @summary Get a Torrent with its announce URLs, files and Grabs, live from its Node while the Node responds
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
 * @summary Get a Torrent with its announce URLs, files and Grabs, live from its Node while the Node responds
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
