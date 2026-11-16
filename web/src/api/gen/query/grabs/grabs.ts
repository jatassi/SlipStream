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
  CancelGrab401,
  CancelGrab403,
  CancelGrab404,
  CancelGrab409,
  CancelGrab422,
  CancelGrab500,
  CancelInputBody,
  CancelQueuedSearches401,
  CancelQueuedSearches403,
  CancelQueuedSearches409,
  CancelQueuedSearches422,
  CancelQueuedSearches500,
  CancelQueuedSearchesParams,
  GetGrab401,
  GetGrab403,
  GetGrab404,
  GetGrab409,
  GetGrab422,
  GetGrab500,
  GetManualSearch401,
  GetManualSearch403,
  GetManualSearch404,
  GetManualSearch409,
  GetManualSearch422,
  GetManualSearch500,
  GetManualSearchParams,
  GetSearchSettings401,
  GetSearchSettings403,
  GetSearchSettings409,
  GetSearchSettings500,
  GetWorkSearching401,
  GetWorkSearching403,
  GetWorkSearching404,
  GetWorkSearching409,
  GetWorkSearching422,
  GetWorkSearching500,
  GrabFromManualSearch401,
  GrabFromManualSearch403,
  GrabFromManualSearch404,
  GrabFromManualSearch409,
  GrabFromManualSearch422,
  GrabFromManualSearch500,
  GrabManualInputBody,
  GrabWaiting401,
  GrabWaiting403,
  GrabWaiting404,
  GrabWaiting409,
  GrabWaiting422,
  GrabWaiting500,
  GrabWaitingInputBody,
  GrabsBlocklist,
  GrabsGrab,
  GrabsGrabList,
  GrabsGrabbing,
  GrabsIndexerGrabList,
  GrabsManualSearch,
  GrabsManualStarted,
  GrabsOfferList,
  GrabsPickResults,
  GrabsQueuedCanceled,
  GrabsReleaseList,
  GrabsSearchList,
  GrabsSearchQueued,
  GrabsSearchScope,
  GrabsSearchSettings,
  GrabsSeveralBody,
  GrabsSeveralQueued,
  GrabsStartManual,
  GrabsWantedPage,
  GrabsWorkSearching,
  ListBlocklist401,
  ListBlocklist403,
  ListBlocklist409,
  ListBlocklist422,
  ListBlocklist500,
  ListBlocklistParams,
  ListGrabs401,
  ListGrabs403,
  ListGrabs409,
  ListGrabs500,
  ListIndexerGrabs401,
  ListIndexerGrabs403,
  ListIndexerGrabs404,
  ListIndexerGrabs409,
  ListIndexerGrabs422,
  ListIndexerGrabs500,
  ListIndexerGrabsParams,
  ListManualSearchReleases401,
  ListManualSearchReleases403,
  ListManualSearchReleases404,
  ListManualSearchReleases409,
  ListManualSearchReleases422,
  ListManualSearchReleases500,
  ListSearchReleases401,
  ListSearchReleases403,
  ListSearchReleases404,
  ListSearchReleases409,
  ListSearchReleases422,
  ListSearchReleases500,
  ListSearches401,
  ListSearches403,
  ListSearches404,
  ListSearches409,
  ListSearches422,
  ListSearches500,
  ListWanted400,
  ListWanted401,
  ListWanted403,
  ListWanted409,
  ListWanted422,
  ListWanted500,
  ListWantedParams,
  RemoveFromBlocklist401,
  RemoveFromBlocklist403,
  RemoveFromBlocklist409,
  RemoveFromBlocklist422,
  RemoveFromBlocklist500,
  SearchSettingsInputBody,
  SearchSeveralWorks401,
  SearchSeveralWorks403,
  SearchSeveralWorks409,
  SearchSeveralWorks422,
  SearchSeveralWorks500,
  SearchWork401,
  SearchWork403,
  SearchWork404,
  SearchWork409,
  SearchWork422,
  SearchWork500,
  SendGrabTo401,
  SendGrabTo403,
  SendGrabTo404,
  SendGrabTo409,
  SendGrabTo422,
  SendGrabTo500,
  SendToInputBody,
  StartManualSearch401,
  StartManualSearch403,
  StartManualSearch404,
  StartManualSearch409,
  StartManualSearch422,
  StartManualSearch500,
  UnblockInputBody,
  UpdateSearchSettings401,
  UpdateSearchSettings403,
  UpdateSearchSettings409,
  UpdateSearchSettings422,
  UpdateSearchSettings500,
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

export const getListBlocklistUrl = (params?: ListBlocklistParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/blocklist?${stringifiedParams}`
    : `/api/v1/blocklist`;
};

/**
 * @summary List the Blocklist, 50 entries at a time, newest first
 */
export const listBlocklist = async (
  params?: ListBlocklistParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsBlocklist> => {
  return apiFetch<GrabsBlocklist>(getListBlocklistUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListBlocklistQueryKey = (params?: ListBlocklistParams) => {
  return [`/api/v1/blocklist`, ...(params ? [params] : [])] as const;
};

export const getListBlocklistSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listBlocklist>>,
  TError = ErrorType<
    ListBlocklist401 | ListBlocklist403 | ListBlocklist409 | ListBlocklist422 | ListBlocklist500
  >,
>(
  params?: ListBlocklistParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBlocklist>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListBlocklistQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listBlocklist>>> = ({ signal }) =>
    listBlocklist(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBlocklist>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListBlocklistSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listBlocklist>>
>;
export type ListBlocklistSuspenseQueryError = ErrorType<
  ListBlocklist401 | ListBlocklist403 | ListBlocklist409 | ListBlocklist422 | ListBlocklist500
>;

export function useListBlocklistSuspense<
  TData = Awaited<ReturnType<typeof listBlocklist>>,
  TError = ErrorType<
    ListBlocklist401 | ListBlocklist403 | ListBlocklist409 | ListBlocklist422 | ListBlocklist500
  >,
>(
  params: undefined | ListBlocklistParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBlocklist>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListBlocklistSuspense<
  TData = Awaited<ReturnType<typeof listBlocklist>>,
  TError = ErrorType<
    ListBlocklist401 | ListBlocklist403 | ListBlocklist409 | ListBlocklist422 | ListBlocklist500
  >,
>(
  params?: ListBlocklistParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBlocklist>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListBlocklistSuspense<
  TData = Awaited<ReturnType<typeof listBlocklist>>,
  TError = ErrorType<
    ListBlocklist401 | ListBlocklist403 | ListBlocklist409 | ListBlocklist422 | ListBlocklist500
  >,
>(
  params?: ListBlocklistParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBlocklist>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Blocklist, 50 entries at a time, newest first
 */

export function useListBlocklistSuspense<
  TData = Awaited<ReturnType<typeof listBlocklist>>,
  TError = ErrorType<
    ListBlocklist401 | ListBlocklist403 | ListBlocklist409 | ListBlocklist422 | ListBlocklist500
  >,
>(
  params?: ListBlocklistParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBlocklist>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListBlocklistSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getRemoveFromBlocklistUrl = () => {
  return `/api/v1/blocklist/remove`;
};

/**
 * @summary Remove Releases from the Blocklist by info hash. Removing one that isn't listed changes nothing
 */
export const removeFromBlocklist = async (
  unblockInputBody: UnblockInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
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
  return apiFetch<void>(getRemoveFromBlocklistUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(unblockInputBody),
  });
};

export const getRemoveFromBlocklistMutationKey = () => ["removeFromBlocklist"] as const;

export const getRemoveFromBlocklistMutationOptions = <
  TError = ErrorType<
    | RemoveFromBlocklist401
    | RemoveFromBlocklist403
    | RemoveFromBlocklist409
    | RemoveFromBlocklist422
    | RemoveFromBlocklist500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof removeFromBlocklist>>,
    TError,
    RemoveFromBlocklistMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof removeFromBlocklist>>,
  TError,
  RemoveFromBlocklistMutationVariables,
  TContext
> => {
  const mutationKey = getRemoveFromBlocklistMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof removeFromBlocklist>>,
    RemoveFromBlocklistMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return removeFromBlocklist(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RemoveFromBlocklistMutationResult = NonNullable<
  Awaited<ReturnType<typeof removeFromBlocklist>>
>;
export type RemoveFromBlocklistMutationBody = UnblockInputBody;
export type RemoveFromBlocklistMutationError = ErrorType<
  | RemoveFromBlocklist401
  | RemoveFromBlocklist403
  | RemoveFromBlocklist409
  | RemoveFromBlocklist422
  | RemoveFromBlocklist500
>;
export type RemoveFromBlocklistMutationVariables = { data: UnblockInputBody };

/**
 * @summary Remove Releases from the Blocklist by info hash. Removing one that isn't listed changes nothing
 */
export const useRemoveFromBlocklist = <
  TError = ErrorType<
    | RemoveFromBlocklist401
    | RemoveFromBlocklist403
    | RemoveFromBlocklist409
    | RemoveFromBlocklist422
    | RemoveFromBlocklist500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof removeFromBlocklist>>,
      TError,
      RemoveFromBlocklistMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof removeFromBlocklist>>,
  TError,
  RemoveFromBlocklistMutationVariables,
  TContext
> => {
  return useMutation(getRemoveFromBlocklistMutationOptions(options), queryClient);
};
export const getListGrabsUrl = () => {
  return `/api/v1/grabs`;
};

/**
 * @summary List every Grab with its stage and progress
 */
export const listGrabs = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsGrabList> => {
  return apiFetch<GrabsGrabList>(getListGrabsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListGrabsQueryKey = () => {
  return [`/api/v1/grabs`] as const;
};

export const getListGrabsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listGrabs>>,
  TError = ErrorType<ListGrabs401 | ListGrabs403 | ListGrabs409 | ListGrabs500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listGrabs>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListGrabsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listGrabs>>> = ({ signal }) =>
    listGrabs({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listGrabs>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListGrabsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listGrabs>>>;
export type ListGrabsSuspenseQueryError = ErrorType<
  ListGrabs401 | ListGrabs403 | ListGrabs409 | ListGrabs500
>;

export function useListGrabsSuspense<
  TData = Awaited<ReturnType<typeof listGrabs>>,
  TError = ErrorType<ListGrabs401 | ListGrabs403 | ListGrabs409 | ListGrabs500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listGrabs>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListGrabsSuspense<
  TData = Awaited<ReturnType<typeof listGrabs>>,
  TError = ErrorType<ListGrabs401 | ListGrabs403 | ListGrabs409 | ListGrabs500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listGrabs>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListGrabsSuspense<
  TData = Awaited<ReturnType<typeof listGrabs>>,
  TError = ErrorType<ListGrabs401 | ListGrabs403 | ListGrabs409 | ListGrabs500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listGrabs>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List every Grab with its stage and progress
 */

export function useListGrabsSuspense<
  TData = Awaited<ReturnType<typeof listGrabs>>,
  TError = ErrorType<ListGrabs401 | ListGrabs403 | ListGrabs409 | ListGrabs500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listGrabs>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListGrabsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetGrabUrl = (grabId: string) => {
  return `/api/v1/grabs/${grabId}`;
};

/**
 * @summary Get a Grab with its stage and progress
 */
export const getGrab = async (
  grabId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsGrab> => {
  return apiFetch<GrabsGrab>(getGetGrabUrl(grabId), {
    ...options,
    method: "GET",
  });
};

export const getGetGrabQueryKey = (grabId: string) => {
  return [`/api/v1/grabs/${grabId}`] as const;
};

export const getGetGrabSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getGrab>>,
  TError = ErrorType<GetGrab401 | GetGrab403 | GetGrab404 | GetGrab409 | GetGrab422 | GetGrab500>,
>(
  grabId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getGrab>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetGrabQueryKey(grabId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getGrab>>> = ({ signal }) =>
    getGrab(grabId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getGrab>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetGrabSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getGrab>>>;
export type GetGrabSuspenseQueryError = ErrorType<
  GetGrab401 | GetGrab403 | GetGrab404 | GetGrab409 | GetGrab422 | GetGrab500
>;

export function useGetGrabSuspense<
  TData = Awaited<ReturnType<typeof getGrab>>,
  TError = ErrorType<GetGrab401 | GetGrab403 | GetGrab404 | GetGrab409 | GetGrab422 | GetGrab500>,
>(
  grabId: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getGrab>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetGrabSuspense<
  TData = Awaited<ReturnType<typeof getGrab>>,
  TError = ErrorType<GetGrab401 | GetGrab403 | GetGrab404 | GetGrab409 | GetGrab422 | GetGrab500>,
>(
  grabId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getGrab>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetGrabSuspense<
  TData = Awaited<ReturnType<typeof getGrab>>,
  TError = ErrorType<GetGrab401 | GetGrab403 | GetGrab404 | GetGrab409 | GetGrab422 | GetGrab500>,
>(
  grabId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getGrab>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Grab with its stage and progress
 */

export function useGetGrabSuspense<
  TData = Awaited<ReturnType<typeof getGrab>>,
  TError = ErrorType<GetGrab401 | GetGrab403 | GetGrab404 | GetGrab409 | GetGrab422 | GetGrab500>,
>(
  grabId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getGrab>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetGrabSuspenseQueryOptions(grabId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCancelGrabUrl = (grabId: string) => {
  return `/api/v1/grabs/${grabId}/cancel`;
};

/**
 * @summary Cancel a Grab that is fetching, downloading or importing. Canceling a canceled Grab changes nothing
 */
export const cancelGrab = async (
  grabId: string,
  cancelInputBody: CancelInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
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
  return apiFetch<void>(getCancelGrabUrl(grabId), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(cancelInputBody),
  });
};

export const getCancelGrabMutationKey = () => ["cancelGrab"] as const;

export const getCancelGrabMutationOptions = <
  TError = ErrorType<
    CancelGrab401 | CancelGrab403 | CancelGrab404 | CancelGrab409 | CancelGrab422 | CancelGrab500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof cancelGrab>>,
    TError,
    CancelGrabMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof cancelGrab>>,
  TError,
  CancelGrabMutationVariables,
  TContext
> => {
  const mutationKey = getCancelGrabMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof cancelGrab>>,
    CancelGrabMutationVariables
  > = (props) => {
    const { grabId, data } = props ?? {};

    return cancelGrab(grabId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CancelGrabMutationResult = NonNullable<Awaited<ReturnType<typeof cancelGrab>>>;
export type CancelGrabMutationBody = CancelInputBody;
export type CancelGrabMutationError = ErrorType<
  CancelGrab401 | CancelGrab403 | CancelGrab404 | CancelGrab409 | CancelGrab422 | CancelGrab500
>;
export type CancelGrabMutationVariables = { grabId: string; data: CancelInputBody };

/**
 * @summary Cancel a Grab that is fetching, downloading or importing. Canceling a canceled Grab changes nothing
 */
export const useCancelGrab = <
  TError = ErrorType<
    CancelGrab401 | CancelGrab403 | CancelGrab404 | CancelGrab409 | CancelGrab422 | CancelGrab500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof cancelGrab>>,
      TError,
      CancelGrabMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof cancelGrab>>,
  TError,
  CancelGrabMutationVariables,
  TContext
> => {
  return useMutation(getCancelGrabMutationOptions(options), queryClient);
};
export const getSendGrabToUrl = (grabId: string) => {
  return `/api/v1/grabs/${grabId}/send-to`;
};

/**
 * @summary Send a Grab to another Download Client: a downloading Grab's torrent, or a Grab whose fetch waits for its client's Node, which then fetches for the new client
 */
export const sendGrabTo = async (
  grabId: string,
  sendToInputBody: SendToInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
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
  return apiFetch<void>(getSendGrabToUrl(grabId), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(sendToInputBody),
  });
};

export const getSendGrabToMutationKey = () => ["sendGrabTo"] as const;

export const getSendGrabToMutationOptions = <
  TError = ErrorType<
    SendGrabTo401 | SendGrabTo403 | SendGrabTo404 | SendGrabTo409 | SendGrabTo422 | SendGrabTo500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof sendGrabTo>>,
    TError,
    SendGrabToMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof sendGrabTo>>,
  TError,
  SendGrabToMutationVariables,
  TContext
> => {
  const mutationKey = getSendGrabToMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof sendGrabTo>>,
    SendGrabToMutationVariables
  > = (props) => {
    const { grabId, data } = props ?? {};

    return sendGrabTo(grabId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SendGrabToMutationResult = NonNullable<Awaited<ReturnType<typeof sendGrabTo>>>;
export type SendGrabToMutationBody = SendToInputBody;
export type SendGrabToMutationError = ErrorType<
  SendGrabTo401 | SendGrabTo403 | SendGrabTo404 | SendGrabTo409 | SendGrabTo422 | SendGrabTo500
>;
export type SendGrabToMutationVariables = { grabId: string; data: SendToInputBody };

/**
 * @summary Send a Grab to another Download Client: a downloading Grab's torrent, or a Grab whose fetch waits for its client's Node, which then fetches for the new client
 */
export const useSendGrabTo = <
  TError = ErrorType<
    SendGrabTo401 | SendGrabTo403 | SendGrabTo404 | SendGrabTo409 | SendGrabTo422 | SendGrabTo500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof sendGrabTo>>,
      TError,
      SendGrabToMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof sendGrabTo>>,
  TError,
  SendGrabToMutationVariables,
  TContext
> => {
  return useMutation(getSendGrabToMutationOptions(options), queryClient);
};
export const getListIndexerGrabsUrl = (id: string, params?: ListIndexerGrabsParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/indexers/${id}/grabs?${stringifiedParams}`
    : `/api/v1/indexers/${id}/grabs`;
};

/**
 * @summary List the Grabs taken from an Indexer, 50 at a time
 */
export const listIndexerGrabs = async (
  id: string,
  params?: ListIndexerGrabsParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsIndexerGrabList> => {
  return apiFetch<GrabsIndexerGrabList>(getListIndexerGrabsUrl(id, params), {
    ...options,
    method: "GET",
  });
};

export const getListIndexerGrabsQueryKey = (id: string, params?: ListIndexerGrabsParams) => {
  return [`/api/v1/indexers/${id}/grabs`, ...(params ? [params] : [])] as const;
};

export const getListIndexerGrabsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listIndexerGrabs>>,
  TError = ErrorType<
    | ListIndexerGrabs401
    | ListIndexerGrabs403
    | ListIndexerGrabs404
    | ListIndexerGrabs409
    | ListIndexerGrabs422
    | ListIndexerGrabs500
  >,
>(
  id: string,
  params?: ListIndexerGrabsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerGrabs>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListIndexerGrabsQueryKey(id, params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listIndexerGrabs>>> = ({ signal }) =>
    listIndexerGrabs(id, params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerGrabs>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListIndexerGrabsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listIndexerGrabs>>
>;
export type ListIndexerGrabsSuspenseQueryError = ErrorType<
  | ListIndexerGrabs401
  | ListIndexerGrabs403
  | ListIndexerGrabs404
  | ListIndexerGrabs409
  | ListIndexerGrabs422
  | ListIndexerGrabs500
>;

export function useListIndexerGrabsSuspense<
  TData = Awaited<ReturnType<typeof listIndexerGrabs>>,
  TError = ErrorType<
    | ListIndexerGrabs401
    | ListIndexerGrabs403
    | ListIndexerGrabs404
    | ListIndexerGrabs409
    | ListIndexerGrabs422
    | ListIndexerGrabs500
  >,
>(
  id: string,
  params: undefined | ListIndexerGrabsParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerGrabs>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListIndexerGrabsSuspense<
  TData = Awaited<ReturnType<typeof listIndexerGrabs>>,
  TError = ErrorType<
    | ListIndexerGrabs401
    | ListIndexerGrabs403
    | ListIndexerGrabs404
    | ListIndexerGrabs409
    | ListIndexerGrabs422
    | ListIndexerGrabs500
  >,
>(
  id: string,
  params?: ListIndexerGrabsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerGrabs>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListIndexerGrabsSuspense<
  TData = Awaited<ReturnType<typeof listIndexerGrabs>>,
  TError = ErrorType<
    | ListIndexerGrabs401
    | ListIndexerGrabs403
    | ListIndexerGrabs404
    | ListIndexerGrabs409
    | ListIndexerGrabs422
    | ListIndexerGrabs500
  >,
>(
  id: string,
  params?: ListIndexerGrabsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerGrabs>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Grabs taken from an Indexer, 50 at a time
 */

export function useListIndexerGrabsSuspense<
  TData = Awaited<ReturnType<typeof listIndexerGrabs>>,
  TError = ErrorType<
    | ListIndexerGrabs401
    | ListIndexerGrabs403
    | ListIndexerGrabs404
    | ListIndexerGrabs409
    | ListIndexerGrabs422
    | ListIndexerGrabs500
  >,
>(
  id: string,
  params?: ListIndexerGrabsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerGrabs>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListIndexerGrabsSuspenseQueryOptions(id, params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCancelQueuedSearchesUrl = (params: CancelQueuedSearchesParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/search-queue?${stringifiedParams}`
    : `/api/v1/search-queue`;
};

/**
 * @summary Cancel every search queued in a lane
 */
export const cancelQueuedSearches = async (
  params: CancelQueuedSearchesParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsQueuedCanceled> => {
  return apiFetch<GrabsQueuedCanceled>(getCancelQueuedSearchesUrl(params), {
    ...options,
    method: "DELETE",
  });
};

export const getCancelQueuedSearchesMutationKey = () => ["cancelQueuedSearches"] as const;

export const getCancelQueuedSearchesMutationOptions = <
  TError = ErrorType<
    | CancelQueuedSearches401
    | CancelQueuedSearches403
    | CancelQueuedSearches409
    | CancelQueuedSearches422
    | CancelQueuedSearches500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof cancelQueuedSearches>>,
    TError,
    CancelQueuedSearchesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof cancelQueuedSearches>>,
  TError,
  CancelQueuedSearchesMutationVariables,
  TContext
> => {
  const mutationKey = getCancelQueuedSearchesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof cancelQueuedSearches>>,
    CancelQueuedSearchesMutationVariables
  > = (props) => {
    const { params } = props ?? {};

    return cancelQueuedSearches(params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CancelQueuedSearchesMutationResult = NonNullable<
  Awaited<ReturnType<typeof cancelQueuedSearches>>
>;

export type CancelQueuedSearchesMutationError = ErrorType<
  | CancelQueuedSearches401
  | CancelQueuedSearches403
  | CancelQueuedSearches409
  | CancelQueuedSearches422
  | CancelQueuedSearches500
>;
export type CancelQueuedSearchesMutationVariables = { params: CancelQueuedSearchesParams };

/**
 * @summary Cancel every search queued in a lane
 */
export const useCancelQueuedSearches = <
  TError = ErrorType<
    | CancelQueuedSearches401
    | CancelQueuedSearches403
    | CancelQueuedSearches409
    | CancelQueuedSearches422
    | CancelQueuedSearches500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof cancelQueuedSearches>>,
      TError,
      CancelQueuedSearchesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof cancelQueuedSearches>>,
  TError,
  CancelQueuedSearchesMutationVariables,
  TContext
> => {
  return useMutation(getCancelQueuedSearchesMutationOptions(options), queryClient);
};
export const getGetSearchSettingsUrl = () => {
  return `/api/v1/search-settings`;
};

/**
 * @summary Get the Automatic search settings
 */
export const getSearchSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsSearchSettings> => {
  return apiFetch<GrabsSearchSettings>(getGetSearchSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetSearchSettingsQueryKey = () => {
  return [`/api/v1/search-settings`] as const;
};

export const getGetSearchSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSearchSettings>>,
  TError = ErrorType<
    GetSearchSettings401 | GetSearchSettings403 | GetSearchSettings409 | GetSearchSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSearchSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSearchSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSearchSettings>>> = ({ signal }) =>
    getSearchSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSearchSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSearchSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getSearchSettings>>
>;
export type GetSearchSettingsSuspenseQueryError = ErrorType<
  GetSearchSettings401 | GetSearchSettings403 | GetSearchSettings409 | GetSearchSettings500
>;

export function useGetSearchSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSearchSettings>>,
  TError = ErrorType<
    GetSearchSettings401 | GetSearchSettings403 | GetSearchSettings409 | GetSearchSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSearchSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSearchSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSearchSettings>>,
  TError = ErrorType<
    GetSearchSettings401 | GetSearchSettings403 | GetSearchSettings409 | GetSearchSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSearchSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSearchSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSearchSettings>>,
  TError = ErrorType<
    GetSearchSettings401 | GetSearchSettings403 | GetSearchSettings409 | GetSearchSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSearchSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Automatic search settings
 */

export function useGetSearchSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSearchSettings>>,
  TError = ErrorType<
    GetSearchSettings401 | GetSearchSettings403 | GetSearchSettings409 | GetSearchSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSearchSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSearchSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateSearchSettingsUrl = () => {
  return `/api/v1/search-settings`;
};

/**
 * @summary Change Automatic search settings by key, all or none
 */
export const updateSearchSettings = async (
  searchSettingsInputBody: SearchSettingsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsSearchSettings> => {
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
  return apiFetch<GrabsSearchSettings>(getUpdateSearchSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(searchSettingsInputBody),
  });
};

export const getUpdateSearchSettingsMutationKey = () => ["updateSearchSettings"] as const;

export const getUpdateSearchSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateSearchSettings401
    | UpdateSearchSettings403
    | UpdateSearchSettings409
    | UpdateSearchSettings422
    | UpdateSearchSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateSearchSettings>>,
    TError,
    UpdateSearchSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateSearchSettings>>,
  TError,
  UpdateSearchSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateSearchSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateSearchSettings>>,
    UpdateSearchSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateSearchSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateSearchSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateSearchSettings>>
>;
export type UpdateSearchSettingsMutationBody = SearchSettingsInputBody;
export type UpdateSearchSettingsMutationError = ErrorType<
  | UpdateSearchSettings401
  | UpdateSearchSettings403
  | UpdateSearchSettings409
  | UpdateSearchSettings422
  | UpdateSearchSettings500
>;
export type UpdateSearchSettingsMutationVariables = { data: SearchSettingsInputBody };

/**
 * @summary Change Automatic search settings by key, all or none
 */
export const useUpdateSearchSettings = <
  TError = ErrorType<
    | UpdateSearchSettings401
    | UpdateSearchSettings403
    | UpdateSearchSettings409
    | UpdateSearchSettings422
    | UpdateSearchSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateSearchSettings>>,
      TError,
      UpdateSearchSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateSearchSettings>>,
  TError,
  UpdateSearchSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateSearchSettingsMutationOptions(options), queryClient);
};
export const getListSearchReleasesUrl = (searchId: string) => {
  return `/api/v1/searches/${searchId}/releases`;
};

/**
 * @summary List the Releases a search run saw, with what it did with each and why
 */
export const listSearchReleases = async (
  searchId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsReleaseList> => {
  return apiFetch<GrabsReleaseList>(getListSearchReleasesUrl(searchId), {
    ...options,
    method: "GET",
  });
};

export const getListSearchReleasesQueryKey = (searchId: string) => {
  return [`/api/v1/searches/${searchId}/releases`] as const;
};

export const getListSearchReleasesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listSearchReleases>>,
  TError = ErrorType<
    | ListSearchReleases401
    | ListSearchReleases403
    | ListSearchReleases404
    | ListSearchReleases409
    | ListSearchReleases422
    | ListSearchReleases500
  >,
>(
  searchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearchReleases>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListSearchReleasesQueryKey(searchId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listSearchReleases>>> = ({ signal }) =>
    listSearchReleases(searchId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearchReleases>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListSearchReleasesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listSearchReleases>>
>;
export type ListSearchReleasesSuspenseQueryError = ErrorType<
  | ListSearchReleases401
  | ListSearchReleases403
  | ListSearchReleases404
  | ListSearchReleases409
  | ListSearchReleases422
  | ListSearchReleases500
>;

export function useListSearchReleasesSuspense<
  TData = Awaited<ReturnType<typeof listSearchReleases>>,
  TError = ErrorType<
    | ListSearchReleases401
    | ListSearchReleases403
    | ListSearchReleases404
    | ListSearchReleases409
    | ListSearchReleases422
    | ListSearchReleases500
  >,
>(
  searchId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearchReleases>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListSearchReleasesSuspense<
  TData = Awaited<ReturnType<typeof listSearchReleases>>,
  TError = ErrorType<
    | ListSearchReleases401
    | ListSearchReleases403
    | ListSearchReleases404
    | ListSearchReleases409
    | ListSearchReleases422
    | ListSearchReleases500
  >,
>(
  searchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearchReleases>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListSearchReleasesSuspense<
  TData = Awaited<ReturnType<typeof listSearchReleases>>,
  TError = ErrorType<
    | ListSearchReleases401
    | ListSearchReleases403
    | ListSearchReleases404
    | ListSearchReleases409
    | ListSearchReleases422
    | ListSearchReleases500
  >,
>(
  searchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearchReleases>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Releases a search run saw, with what it did with each and why
 */

export function useListSearchReleasesSuspense<
  TData = Awaited<ReturnType<typeof listSearchReleases>>,
  TError = ErrorType<
    | ListSearchReleases401
    | ListSearchReleases403
    | ListSearchReleases404
    | ListSearchReleases409
    | ListSearchReleases422
    | ListSearchReleases500
  >,
>(
  searchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearchReleases>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListSearchReleasesSuspenseQueryOptions(searchId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getListWantedUrl = (params?: ListWantedParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/wanted?${stringifiedParams}` : `/api/v1/wanted`;
};

/**
 * @summary List the Works that want a Release, 100 at a time, with what the search queue is doing
 */
export const listWanted = async (
  params?: ListWantedParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsWantedPage> => {
  return apiFetch<GrabsWantedPage>(getListWantedUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListWantedQueryKey = (params?: ListWantedParams) => {
  return [`/api/v1/wanted`, ...(params ? [params] : [])] as const;
};

export const getListWantedSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listWanted>>,
  TError = ErrorType<
    ListWanted400 | ListWanted401 | ListWanted403 | ListWanted409 | ListWanted422 | ListWanted500
  >,
>(
  params?: ListWantedParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWanted>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListWantedQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listWanted>>> = ({ signal }) =>
    listWanted(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWanted>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListWantedSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listWanted>>>;
export type ListWantedSuspenseQueryError = ErrorType<
  ListWanted400 | ListWanted401 | ListWanted403 | ListWanted409 | ListWanted422 | ListWanted500
>;

export function useListWantedSuspense<
  TData = Awaited<ReturnType<typeof listWanted>>,
  TError = ErrorType<
    ListWanted400 | ListWanted401 | ListWanted403 | ListWanted409 | ListWanted422 | ListWanted500
  >,
>(
  params: undefined | ListWantedParams,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWanted>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListWantedSuspense<
  TData = Awaited<ReturnType<typeof listWanted>>,
  TError = ErrorType<
    ListWanted400 | ListWanted401 | ListWanted403 | ListWanted409 | ListWanted422 | ListWanted500
  >,
>(
  params?: ListWantedParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWanted>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListWantedSuspense<
  TData = Awaited<ReturnType<typeof listWanted>>,
  TError = ErrorType<
    ListWanted400 | ListWanted401 | ListWanted403 | ListWanted409 | ListWanted422 | ListWanted500
  >,
>(
  params?: ListWantedParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWanted>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Works that want a Release, 100 at a time, with what the search queue is doing
 */

export function useListWantedSuspense<
  TData = Awaited<ReturnType<typeof listWanted>>,
  TError = ErrorType<
    ListWanted400 | ListWanted401 | ListWanted403 | ListWanted409 | ListWanted422 | ListWanted500
  >,
>(
  params?: ListWantedParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWanted>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListWantedSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSearchSeveralWorksUrl = () => {
  return `/api/v1/works/search`;
};

/**
 * @summary Queue a search of several Works, by id or by a Wanted filter, behind every search already queued
 */
export const searchSeveralWorks = async (
  grabsSeveralBody: GrabsSeveralBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsSeveralQueued> => {
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
  return apiFetch<GrabsSeveralQueued>(getSearchSeveralWorksUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(grabsSeveralBody),
  });
};

export const getSearchSeveralWorksMutationKey = () => ["searchSeveralWorks"] as const;

export const getSearchSeveralWorksMutationOptions = <
  TError = ErrorType<
    | SearchSeveralWorks401
    | SearchSeveralWorks403
    | SearchSeveralWorks409
    | SearchSeveralWorks422
    | SearchSeveralWorks500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof searchSeveralWorks>>,
    TError,
    SearchSeveralWorksMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof searchSeveralWorks>>,
  TError,
  SearchSeveralWorksMutationVariables,
  TContext
> => {
  const mutationKey = getSearchSeveralWorksMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof searchSeveralWorks>>,
    SearchSeveralWorksMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return searchSeveralWorks(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SearchSeveralWorksMutationResult = NonNullable<
  Awaited<ReturnType<typeof searchSeveralWorks>>
>;
export type SearchSeveralWorksMutationBody = GrabsSeveralBody;
export type SearchSeveralWorksMutationError = ErrorType<
  | SearchSeveralWorks401
  | SearchSeveralWorks403
  | SearchSeveralWorks409
  | SearchSeveralWorks422
  | SearchSeveralWorks500
>;
export type SearchSeveralWorksMutationVariables = { data: GrabsSeveralBody };

/**
 * @summary Queue a search of several Works, by id or by a Wanted filter, behind every search already queued
 */
export const useSearchSeveralWorks = <
  TError = ErrorType<
    | SearchSeveralWorks401
    | SearchSeveralWorks403
    | SearchSeveralWorks409
    | SearchSeveralWorks422
    | SearchSeveralWorks500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof searchSeveralWorks>>,
      TError,
      SearchSeveralWorksMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof searchSeveralWorks>>,
  TError,
  SearchSeveralWorksMutationVariables,
  TContext
> => {
  return useMutation(getSearchSeveralWorksMutationOptions(options), queryClient);
};
export const getGetManualSearchUrl = (workId: string, params?: GetManualSearchParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/works/${workId}/manual-search?${stringifiedParams}`
    : `/api/v1/works/${workId}/manual-search`;
};

/**
 * @summary Get a Work's Manual search with its plan worked out around the pins
 */
export const getManualSearch = async (
  workId: string,
  params?: GetManualSearchParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsManualSearch> => {
  return apiFetch<GrabsManualSearch>(getGetManualSearchUrl(workId, params), {
    ...options,
    method: "GET",
  });
};

export const getGetManualSearchQueryKey = (workId: string, params?: GetManualSearchParams) => {
  return [`/api/v1/works/${workId}/manual-search`, ...(params ? [params] : [])] as const;
};

export const getGetManualSearchSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getManualSearch>>,
  TError = ErrorType<
    | GetManualSearch401
    | GetManualSearch403
    | GetManualSearch404
    | GetManualSearch409
    | GetManualSearch422
    | GetManualSearch500
  >,
>(
  workId: string,
  params?: GetManualSearchParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getManualSearch>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetManualSearchQueryKey(workId, params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getManualSearch>>> = ({ signal }) =>
    getManualSearch(workId, params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getManualSearch>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetManualSearchSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getManualSearch>>
>;
export type GetManualSearchSuspenseQueryError = ErrorType<
  | GetManualSearch401
  | GetManualSearch403
  | GetManualSearch404
  | GetManualSearch409
  | GetManualSearch422
  | GetManualSearch500
>;

export function useGetManualSearchSuspense<
  TData = Awaited<ReturnType<typeof getManualSearch>>,
  TError = ErrorType<
    | GetManualSearch401
    | GetManualSearch403
    | GetManualSearch404
    | GetManualSearch409
    | GetManualSearch422
    | GetManualSearch500
  >,
>(
  workId: string,
  params: undefined | GetManualSearchParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getManualSearch>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetManualSearchSuspense<
  TData = Awaited<ReturnType<typeof getManualSearch>>,
  TError = ErrorType<
    | GetManualSearch401
    | GetManualSearch403
    | GetManualSearch404
    | GetManualSearch409
    | GetManualSearch422
    | GetManualSearch500
  >,
>(
  workId: string,
  params?: GetManualSearchParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getManualSearch>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetManualSearchSuspense<
  TData = Awaited<ReturnType<typeof getManualSearch>>,
  TError = ErrorType<
    | GetManualSearch401
    | GetManualSearch403
    | GetManualSearch404
    | GetManualSearch409
    | GetManualSearch422
    | GetManualSearch500
  >,
>(
  workId: string,
  params?: GetManualSearchParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getManualSearch>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Work's Manual search with its plan worked out around the pins
 */

export function useGetManualSearchSuspense<
  TData = Awaited<ReturnType<typeof getManualSearch>>,
  TError = ErrorType<
    | GetManualSearch401
    | GetManualSearch403
    | GetManualSearch404
    | GetManualSearch409
    | GetManualSearch422
    | GetManualSearch500
  >,
>(
  workId: string,
  params?: GetManualSearchParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getManualSearch>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetManualSearchSuspenseQueryOptions(workId, params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getStartManualSearchUrl = (workId: string) => {
  return `/api/v1/works/${workId}/manual-search`;
};

/**
 * @summary Start a Manual search of a Work, or of some of its levels and Items, or of one Item at one Target. Responds 200 with the existing Manual search when it already has this scope and query
 */
export const startManualSearch = async (
  workId: string,
  grabsStartManual?: GrabsStartManual,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsManualStarted> => {
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
  return apiFetch<GrabsManualStarted>(getStartManualSearchUrl(workId), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(grabsStartManual),
  });
};

export const getStartManualSearchMutationKey = () => ["startManualSearch"] as const;

export const getStartManualSearchMutationOptions = <
  TError = ErrorType<
    | StartManualSearch401
    | StartManualSearch403
    | StartManualSearch404
    | StartManualSearch409
    | StartManualSearch422
    | StartManualSearch500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof startManualSearch>>,
    TError,
    StartManualSearchMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof startManualSearch>>,
  TError,
  StartManualSearchMutationVariables,
  TContext
> => {
  const mutationKey = getStartManualSearchMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof startManualSearch>>,
    StartManualSearchMutationVariables
  > = (props) => {
    const { workId, data } = props ?? {};

    return startManualSearch(workId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type StartManualSearchMutationResult = NonNullable<
  Awaited<ReturnType<typeof startManualSearch>>
>;
export type StartManualSearchMutationBody = GrabsStartManual | undefined;
export type StartManualSearchMutationError = ErrorType<
  | StartManualSearch401
  | StartManualSearch403
  | StartManualSearch404
  | StartManualSearch409
  | StartManualSearch422
  | StartManualSearch500
>;
export type StartManualSearchMutationVariables = { workId: string; data?: GrabsStartManual };

/**
 * @summary Start a Manual search of a Work, or of some of its levels and Items, or of one Item at one Target. Responds 200 with the existing Manual search when it already has this scope and query
 */
export const useStartManualSearch = <
  TError = ErrorType<
    | StartManualSearch401
    | StartManualSearch403
    | StartManualSearch404
    | StartManualSearch409
    | StartManualSearch422
    | StartManualSearch500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof startManualSearch>>,
      TError,
      StartManualSearchMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof startManualSearch>>,
  TError,
  StartManualSearchMutationVariables,
  TContext
> => {
  return useMutation(getStartManualSearchMutationOptions(options), queryClient);
};
export const getGrabFromManualSearchUrl = (workId: string) => {
  return `/api/v1/works/${workId}/manual-search/grabs`;
};

/**
 * @summary Grab the picked Releases of a Work's Manual search in one go, each judged again and refused alone
 */
export const grabFromManualSearch = async (
  workId: string,
  grabManualInputBody: GrabManualInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsPickResults> => {
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
  return apiFetch<GrabsPickResults>(getGrabFromManualSearchUrl(workId), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(grabManualInputBody),
  });
};

export const getGrabFromManualSearchMutationKey = () => ["grabFromManualSearch"] as const;

export const getGrabFromManualSearchMutationOptions = <
  TError = ErrorType<
    | GrabFromManualSearch401
    | GrabFromManualSearch403
    | GrabFromManualSearch404
    | GrabFromManualSearch409
    | GrabFromManualSearch422
    | GrabFromManualSearch500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof grabFromManualSearch>>,
    TError,
    GrabFromManualSearchMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof grabFromManualSearch>>,
  TError,
  GrabFromManualSearchMutationVariables,
  TContext
> => {
  const mutationKey = getGrabFromManualSearchMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof grabFromManualSearch>>,
    GrabFromManualSearchMutationVariables
  > = (props) => {
    const { workId, data } = props ?? {};

    return grabFromManualSearch(workId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type GrabFromManualSearchMutationResult = NonNullable<
  Awaited<ReturnType<typeof grabFromManualSearch>>
>;
export type GrabFromManualSearchMutationBody = GrabManualInputBody;
export type GrabFromManualSearchMutationError = ErrorType<
  | GrabFromManualSearch401
  | GrabFromManualSearch403
  | GrabFromManualSearch404
  | GrabFromManualSearch409
  | GrabFromManualSearch422
  | GrabFromManualSearch500
>;
export type GrabFromManualSearchMutationVariables = { workId: string; data: GrabManualInputBody };

/**
 * @summary Grab the picked Releases of a Work's Manual search in one go, each judged again and refused alone
 */
export const useGrabFromManualSearch = <
  TError = ErrorType<
    | GrabFromManualSearch401
    | GrabFromManualSearch403
    | GrabFromManualSearch404
    | GrabFromManualSearch409
    | GrabFromManualSearch422
    | GrabFromManualSearch500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof grabFromManualSearch>>,
      TError,
      GrabFromManualSearchMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof grabFromManualSearch>>,
  TError,
  GrabFromManualSearchMutationVariables,
  TContext
> => {
  return useMutation(getGrabFromManualSearchMutationOptions(options), queryClient);
};
export const getListManualSearchReleasesUrl = (workId: string) => {
  return `/api/v1/works/${workId}/manual-search/releases`;
};

/**
 * @summary List every Release of a Work's Manual search with what grabbing it alone does
 */
export const listManualSearchReleases = async (
  workId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsOfferList> => {
  return apiFetch<GrabsOfferList>(getListManualSearchReleasesUrl(workId), {
    ...options,
    method: "GET",
  });
};

export const getListManualSearchReleasesQueryKey = (workId: string) => {
  return [`/api/v1/works/${workId}/manual-search/releases`] as const;
};

export const getListManualSearchReleasesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listManualSearchReleases>>,
  TError = ErrorType<
    | ListManualSearchReleases401
    | ListManualSearchReleases403
    | ListManualSearchReleases404
    | ListManualSearchReleases409
    | ListManualSearchReleases422
    | ListManualSearchReleases500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listManualSearchReleases>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListManualSearchReleasesQueryKey(workId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listManualSearchReleases>>> = ({
    signal,
  }) => listManualSearchReleases(workId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listManualSearchReleases>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListManualSearchReleasesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listManualSearchReleases>>
>;
export type ListManualSearchReleasesSuspenseQueryError = ErrorType<
  | ListManualSearchReleases401
  | ListManualSearchReleases403
  | ListManualSearchReleases404
  | ListManualSearchReleases409
  | ListManualSearchReleases422
  | ListManualSearchReleases500
>;

export function useListManualSearchReleasesSuspense<
  TData = Awaited<ReturnType<typeof listManualSearchReleases>>,
  TError = ErrorType<
    | ListManualSearchReleases401
    | ListManualSearchReleases403
    | ListManualSearchReleases404
    | ListManualSearchReleases409
    | ListManualSearchReleases422
    | ListManualSearchReleases500
  >,
>(
  workId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listManualSearchReleases>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListManualSearchReleasesSuspense<
  TData = Awaited<ReturnType<typeof listManualSearchReleases>>,
  TError = ErrorType<
    | ListManualSearchReleases401
    | ListManualSearchReleases403
    | ListManualSearchReleases404
    | ListManualSearchReleases409
    | ListManualSearchReleases422
    | ListManualSearchReleases500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listManualSearchReleases>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListManualSearchReleasesSuspense<
  TData = Awaited<ReturnType<typeof listManualSearchReleases>>,
  TError = ErrorType<
    | ListManualSearchReleases401
    | ListManualSearchReleases403
    | ListManualSearchReleases404
    | ListManualSearchReleases409
    | ListManualSearchReleases422
    | ListManualSearchReleases500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listManualSearchReleases>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List every Release of a Work's Manual search with what grabbing it alone does
 */

export function useListManualSearchReleasesSuspense<
  TData = Awaited<ReturnType<typeof listManualSearchReleases>>,
  TError = ErrorType<
    | ListManualSearchReleases401
    | ListManualSearchReleases403
    | ListManualSearchReleases404
    | ListManualSearchReleases409
    | ListManualSearchReleases422
    | ListManualSearchReleases500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listManualSearchReleases>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListManualSearchReleasesSuspenseQueryOptions(workId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSearchWorkUrl = (workId: string) => {
  return `/api/v1/works/${workId}/search`;
};

/**
 * @summary Queue a search of each wanted Item of a Work, or of some of its levels and Items, to grab the best release
 */
export const searchWork = async (
  workId: string,
  grabsSearchScope?: GrabsSearchScope,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsSearchQueued> => {
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
  return apiFetch<GrabsSearchQueued>(getSearchWorkUrl(workId), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(grabsSearchScope),
  });
};

export const getSearchWorkMutationKey = () => ["searchWork"] as const;

export const getSearchWorkMutationOptions = <
  TError = ErrorType<
    SearchWork401 | SearchWork403 | SearchWork404 | SearchWork409 | SearchWork422 | SearchWork500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof searchWork>>,
    TError,
    SearchWorkMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof searchWork>>,
  TError,
  SearchWorkMutationVariables,
  TContext
> => {
  const mutationKey = getSearchWorkMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof searchWork>>,
    SearchWorkMutationVariables
  > = (props) => {
    const { workId, data } = props ?? {};

    return searchWork(workId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SearchWorkMutationResult = NonNullable<Awaited<ReturnType<typeof searchWork>>>;
export type SearchWorkMutationBody = GrabsSearchScope | undefined;
export type SearchWorkMutationError = ErrorType<
  SearchWork401 | SearchWork403 | SearchWork404 | SearchWork409 | SearchWork422 | SearchWork500
>;
export type SearchWorkMutationVariables = { workId: string; data?: GrabsSearchScope };

/**
 * @summary Queue a search of each wanted Item of a Work, or of some of its levels and Items, to grab the best release
 */
export const useSearchWork = <
  TError = ErrorType<
    SearchWork401 | SearchWork403 | SearchWork404 | SearchWork409 | SearchWork422 | SearchWork500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof searchWork>>,
      TError,
      SearchWorkMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof searchWork>>,
  TError,
  SearchWorkMutationVariables,
  TContext
> => {
  return useMutation(getSearchWorkMutationOptions(options), queryClient);
};
export const getListSearchesUrl = (workId: string) => {
  return `/api/v1/works/${workId}/searches`;
};

/**
 * @summary List a Work's search runs and their outcomes
 */
export const listSearches = async (
  workId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsSearchList> => {
  return apiFetch<GrabsSearchList>(getListSearchesUrl(workId), {
    ...options,
    method: "GET",
  });
};

export const getListSearchesQueryKey = (workId: string) => {
  return [`/api/v1/works/${workId}/searches`] as const;
};

export const getListSearchesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listSearches>>,
  TError = ErrorType<
    | ListSearches401
    | ListSearches403
    | ListSearches404
    | ListSearches409
    | ListSearches422
    | ListSearches500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearches>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListSearchesQueryKey(workId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listSearches>>> = ({ signal }) =>
    listSearches(workId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearches>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListSearchesSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listSearches>>>;
export type ListSearchesSuspenseQueryError = ErrorType<
  | ListSearches401
  | ListSearches403
  | ListSearches404
  | ListSearches409
  | ListSearches422
  | ListSearches500
>;

export function useListSearchesSuspense<
  TData = Awaited<ReturnType<typeof listSearches>>,
  TError = ErrorType<
    | ListSearches401
    | ListSearches403
    | ListSearches404
    | ListSearches409
    | ListSearches422
    | ListSearches500
  >,
>(
  workId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearches>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListSearchesSuspense<
  TData = Awaited<ReturnType<typeof listSearches>>,
  TError = ErrorType<
    | ListSearches401
    | ListSearches403
    | ListSearches404
    | ListSearches409
    | ListSearches422
    | ListSearches500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearches>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListSearchesSuspense<
  TData = Awaited<ReturnType<typeof listSearches>>,
  TError = ErrorType<
    | ListSearches401
    | ListSearches403
    | ListSearches404
    | ListSearches409
    | ListSearches422
    | ListSearches500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearches>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List a Work's search runs and their outcomes
 */

export function useListSearchesSuspense<
  TData = Awaited<ReturnType<typeof listSearches>>,
  TError = ErrorType<
    | ListSearches401
    | ListSearches403
    | ListSearches404
    | ListSearches409
    | ListSearches422
    | ListSearches500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSearches>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListSearchesSuspenseQueryOptions(workId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetWorkSearchingUrl = (workId: string) => {
  return `/api/v1/works/${workId}/searching`;
};

/**
 * @summary Get what Automatic search is doing for a Work: each search group's state, its waits and the Items and Targets it stopped searching again for
 */
export const getWorkSearching = async (
  workId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsWorkSearching> => {
  return apiFetch<GrabsWorkSearching>(getGetWorkSearchingUrl(workId), {
    ...options,
    method: "GET",
  });
};

export const getGetWorkSearchingQueryKey = (workId: string) => {
  return [`/api/v1/works/${workId}/searching`] as const;
};

export const getGetWorkSearchingSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getWorkSearching>>,
  TError = ErrorType<
    | GetWorkSearching401
    | GetWorkSearching403
    | GetWorkSearching404
    | GetWorkSearching409
    | GetWorkSearching422
    | GetWorkSearching500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSearching>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetWorkSearchingQueryKey(workId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getWorkSearching>>> = ({ signal }) =>
    getWorkSearching(workId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSearching>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetWorkSearchingSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getWorkSearching>>
>;
export type GetWorkSearchingSuspenseQueryError = ErrorType<
  | GetWorkSearching401
  | GetWorkSearching403
  | GetWorkSearching404
  | GetWorkSearching409
  | GetWorkSearching422
  | GetWorkSearching500
>;

export function useGetWorkSearchingSuspense<
  TData = Awaited<ReturnType<typeof getWorkSearching>>,
  TError = ErrorType<
    | GetWorkSearching401
    | GetWorkSearching403
    | GetWorkSearching404
    | GetWorkSearching409
    | GetWorkSearching422
    | GetWorkSearching500
  >,
>(
  workId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSearching>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkSearchingSuspense<
  TData = Awaited<ReturnType<typeof getWorkSearching>>,
  TError = ErrorType<
    | GetWorkSearching401
    | GetWorkSearching403
    | GetWorkSearching404
    | GetWorkSearching409
    | GetWorkSearching422
    | GetWorkSearching500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSearching>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkSearchingSuspense<
  TData = Awaited<ReturnType<typeof getWorkSearching>>,
  TError = ErrorType<
    | GetWorkSearching401
    | GetWorkSearching403
    | GetWorkSearching404
    | GetWorkSearching409
    | GetWorkSearching422
    | GetWorkSearching500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSearching>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get what Automatic search is doing for a Work: each search group's state, its waits and the Items and Targets it stopped searching again for
 */

export function useGetWorkSearchingSuspense<
  TData = Awaited<ReturnType<typeof getWorkSearching>>,
  TError = ErrorType<
    | GetWorkSearching401
    | GetWorkSearching403
    | GetWorkSearching404
    | GetWorkSearching409
    | GetWorkSearching422
    | GetWorkSearching500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSearching>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetWorkSearchingSuspenseQueryOptions(workId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGrabWaitingUrl = (workId: string) => {
  return `/api/v1/works/${workId}/waits/grab`;
};

/**
 * @summary End a Work's waits for a better Release and grab the best one found for each now
 */
export const grabWaiting = async (
  workId: string,
  grabWaitingInputBody: GrabWaitingInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsGrabbing> => {
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
  return apiFetch<GrabsGrabbing>(getGrabWaitingUrl(workId), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(grabWaitingInputBody),
  });
};

export const getGrabWaitingMutationKey = () => ["grabWaiting"] as const;

export const getGrabWaitingMutationOptions = <
  TError = ErrorType<
    | GrabWaiting401
    | GrabWaiting403
    | GrabWaiting404
    | GrabWaiting409
    | GrabWaiting422
    | GrabWaiting500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof grabWaiting>>,
    TError,
    GrabWaitingMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof grabWaiting>>,
  TError,
  GrabWaitingMutationVariables,
  TContext
> => {
  const mutationKey = getGrabWaitingMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof grabWaiting>>,
    GrabWaitingMutationVariables
  > = (props) => {
    const { workId, data } = props ?? {};

    return grabWaiting(workId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type GrabWaitingMutationResult = NonNullable<Awaited<ReturnType<typeof grabWaiting>>>;
export type GrabWaitingMutationBody = GrabWaitingInputBody;
export type GrabWaitingMutationError = ErrorType<
  | GrabWaiting401
  | GrabWaiting403
  | GrabWaiting404
  | GrabWaiting409
  | GrabWaiting422
  | GrabWaiting500
>;
export type GrabWaitingMutationVariables = { workId: string; data: GrabWaitingInputBody };

/**
 * @summary End a Work's waits for a better Release and grab the best one found for each now
 */
export const useGrabWaiting = <
  TError = ErrorType<
    | GrabWaiting401
    | GrabWaiting403
    | GrabWaiting404
    | GrabWaiting409
    | GrabWaiting422
    | GrabWaiting500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof grabWaiting>>,
      TError,
      GrabWaitingMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof grabWaiting>>,
  TError,
  GrabWaitingMutationVariables,
  TContext
> => {
  return useMutation(getGrabWaitingMutationOptions(options), queryClient);
};
