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
  DownloadclientsSettings,
  DownloadclientsSettingsPatch,
  GetDownloadClientSettings401,
  GetDownloadClientSettings403,
  GetDownloadClientSettings404,
  GetDownloadClientSettings409,
  GetDownloadClientSettings422,
  GetDownloadClientSettings500,
  SetDownloadClientSettings401,
  SetDownloadClientSettings403,
  SetDownloadClientSettings404,
  SetDownloadClientSettings409,
  SetDownloadClientSettings422,
  SetDownloadClientSettings500,
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
