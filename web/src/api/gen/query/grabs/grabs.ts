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
  GrabsSearchList,
  GrabsSearchStarted,
  ListGrabs401,
  ListGrabs403,
  ListGrabs409,
  ListGrabs500,
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

export const getSearchWorkUrl = (workId: string) => {
  return `/api/v1/works/${workId}/search`;
};

/**
 * @summary Search for each of a Work's wanted cells and grab the best release
 */
export const searchWork = async (
  workId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<GrabsSearchStarted> => {
  return apiFetch<GrabsSearchStarted>(getSearchWorkUrl(workId), {
    ...options,
    method: "POST",
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
    const { workId } = props ?? {};

    return searchWork(workId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SearchWorkMutationResult = NonNullable<Awaited<ReturnType<typeof searchWork>>>;

export type SearchWorkMutationError = ErrorType<
  SearchWork401 | SearchWork403 | SearchWork404 | SearchWork409 | SearchWork422 | SearchWork500
>;
export type SearchWorkMutationVariables = { workId: string };

/**
 * @summary Search for each of a Work's wanted cells and grab the best release
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
