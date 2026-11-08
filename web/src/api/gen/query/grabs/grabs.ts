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
  GetGrab401,
  GetGrab403,
  GetGrab404,
  GetGrab409,
  GetGrab422,
  GetGrab500,
  GrabsGrab,
  GrabsGrabList,
  GrabsIndexerGrabList,
  GrabsSearchList,
  GrabsSearchScope,
  GrabsSearchStarted,
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
  ListSearches401,
  ListSearches403,
  ListSearches404,
  ListSearches409,
  ListSearches422,
  ListSearches500,
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

export const getSearchWorkUrl = (workId: string) => {
  return `/api/v1/works/${workId}/search`;
};

/**
 * @summary Search for each wanted Item of a Work, or of some of its levels and Items, and grab the best release
 */
export const searchWork = async (
  workId: string,
  grabsSearchScope?: GrabsSearchScope,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsSearchStarted> => {
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
  return apiFetch<GrabsSearchStarted>(getSearchWorkUrl(workId), {
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
 * @summary Search for each wanted Item of a Work, or of some of its levels and Items, and grab the best release
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
