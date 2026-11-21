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
  ContinueReconcilingNode401,
  ContinueReconcilingNode403,
  ContinueReconcilingNode404,
  ContinueReconcilingNode409,
  ContinueReconcilingNode422,
  ContinueReconcilingNode500,
  ContinueReconcilingRoot401,
  ContinueReconcilingRoot403,
  ContinueReconcilingRoot404,
  ContinueReconcilingRoot409,
  ContinueReconcilingRoot422,
  ContinueReconcilingRoot500,
  FinishReconciling401,
  FinishReconciling403,
  FinishReconciling409,
  FinishReconciling500,
  ForgetReconcilingTorrents401,
  ForgetReconcilingTorrents403,
  ForgetReconcilingTorrents404,
  ForgetReconcilingTorrents409,
  ForgetReconcilingTorrents422,
  ForgetReconcilingTorrents500,
  GetReconciling401,
  GetReconciling403,
  GetReconciling409,
  GetReconciling500,
  ListReconcilingTorrents401,
  ListReconcilingTorrents403,
  ListReconcilingTorrents409,
  ListReconcilingTorrents422,
  ListReconcilingTorrents500,
  ListReconcilingTorrentsParams,
  ReaddReconcilingTorrents401,
  ReaddReconcilingTorrents403,
  ReaddReconcilingTorrents404,
  ReaddReconcilingTorrents409,
  ReaddReconcilingTorrents422,
  ReaddReconcilingTorrents500,
  ReconcilingForgetRequest,
  ReconcilingPage,
  ReconcilingReaddRequest,
  ReconcilingRehomeRequest,
  ReconcilingRehomed,
  ReconcilingReviewPage,
  RehomeReconciling401,
  RehomeReconciling403,
  RehomeReconciling404,
  RehomeReconciling409,
  RehomeReconciling422,
  RehomeReconciling500,
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

export const getGetReconcilingUrl = () => {
  return `/api/v1/reconciling`;
};

/**
 * @summary Read what Reconciling waits on after a restore
 */
export const getReconciling = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReconcilingPage> => {
  return apiFetch<ReconcilingPage>(getGetReconcilingUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetReconcilingQueryKey = () => {
  return [`/api/v1/reconciling`] as const;
};

export const getGetReconcilingSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getReconciling>>,
  TError = ErrorType<GetReconciling401 | GetReconciling403 | GetReconciling409 | GetReconciling500>,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getReconciling>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetReconcilingQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getReconciling>>> = ({ signal }) =>
    getReconciling({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getReconciling>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetReconcilingSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getReconciling>>
>;
export type GetReconcilingSuspenseQueryError = ErrorType<
  GetReconciling401 | GetReconciling403 | GetReconciling409 | GetReconciling500
>;

export function useGetReconcilingSuspense<
  TData = Awaited<ReturnType<typeof getReconciling>>,
  TError = ErrorType<GetReconciling401 | GetReconciling403 | GetReconciling409 | GetReconciling500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getReconciling>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetReconcilingSuspense<
  TData = Awaited<ReturnType<typeof getReconciling>>,
  TError = ErrorType<GetReconciling401 | GetReconciling403 | GetReconciling409 | GetReconciling500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getReconciling>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetReconcilingSuspense<
  TData = Awaited<ReturnType<typeof getReconciling>>,
  TError = ErrorType<GetReconciling401 | GetReconciling403 | GetReconciling409 | GetReconciling500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getReconciling>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Read what Reconciling waits on after a restore
 */

export function useGetReconcilingSuspense<
  TData = Awaited<ReturnType<typeof getReconciling>>,
  TError = ErrorType<GetReconciling401 | GetReconciling403 | GetReconciling409 | GetReconciling500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getReconciling>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetReconcilingSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getFinishReconcilingUrl = () => {
  return `/api/v1/reconciling/finish`;
};

/**
 * @summary Finish Reconciling now
 */
export const finishReconciling = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getFinishReconcilingUrl(), {
    ...options,
    method: "POST",
  });
};

export const getFinishReconcilingMutationKey = () => ["finishReconciling"] as const;

export const getFinishReconcilingMutationOptions = <
  TError = ErrorType<
    FinishReconciling401 | FinishReconciling403 | FinishReconciling409 | FinishReconciling500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof finishReconciling>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof finishReconciling>>, TError, void, TContext> => {
  const mutationKey = getFinishReconcilingMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof finishReconciling>>, void> = () => {
    return finishReconciling(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type FinishReconcilingMutationResult = NonNullable<
  Awaited<ReturnType<typeof finishReconciling>>
>;

export type FinishReconcilingMutationError = ErrorType<
  FinishReconciling401 | FinishReconciling403 | FinishReconciling409 | FinishReconciling500
>;

/**
 * @summary Finish Reconciling now
 */
export const useFinishReconciling = <
  TError = ErrorType<
    FinishReconciling401 | FinishReconciling403 | FinishReconciling409 | FinishReconciling500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof finishReconciling>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof finishReconciling>>, TError, void, TContext> => {
  return useMutation(getFinishReconcilingMutationOptions(options), queryClient);
};
export const getContinueReconcilingNodeUrl = (nodeId: string) => {
  return `/api/v1/reconciling/nodes/${nodeId}/continue`;
};

/**
 * @summary Stop waiting for a Node's torrents
 */
export const continueReconcilingNode = async (
  nodeId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getContinueReconcilingNodeUrl(nodeId), {
    ...options,
    method: "POST",
  });
};

export const getContinueReconcilingNodeMutationKey = () => ["continueReconcilingNode"] as const;

export const getContinueReconcilingNodeMutationOptions = <
  TError = ErrorType<
    | ContinueReconcilingNode401
    | ContinueReconcilingNode403
    | ContinueReconcilingNode404
    | ContinueReconcilingNode409
    | ContinueReconcilingNode422
    | ContinueReconcilingNode500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof continueReconcilingNode>>,
    TError,
    ContinueReconcilingNodeMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof continueReconcilingNode>>,
  TError,
  ContinueReconcilingNodeMutationVariables,
  TContext
> => {
  const mutationKey = getContinueReconcilingNodeMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof continueReconcilingNode>>,
    ContinueReconcilingNodeMutationVariables
  > = (props) => {
    const { nodeId } = props ?? {};

    return continueReconcilingNode(nodeId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ContinueReconcilingNodeMutationResult = NonNullable<
  Awaited<ReturnType<typeof continueReconcilingNode>>
>;

export type ContinueReconcilingNodeMutationError = ErrorType<
  | ContinueReconcilingNode401
  | ContinueReconcilingNode403
  | ContinueReconcilingNode404
  | ContinueReconcilingNode409
  | ContinueReconcilingNode422
  | ContinueReconcilingNode500
>;
export type ContinueReconcilingNodeMutationVariables = { nodeId: string };

/**
 * @summary Stop waiting for a Node's torrents
 */
export const useContinueReconcilingNode = <
  TError = ErrorType<
    | ContinueReconcilingNode401
    | ContinueReconcilingNode403
    | ContinueReconcilingNode404
    | ContinueReconcilingNode409
    | ContinueReconcilingNode422
    | ContinueReconcilingNode500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof continueReconcilingNode>>,
      TError,
      ContinueReconcilingNodeMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof continueReconcilingNode>>,
  TError,
  ContinueReconcilingNodeMutationVariables,
  TContext
> => {
  return useMutation(getContinueReconcilingNodeMutationOptions(options), queryClient);
};
export const getRehomeReconcilingUrl = () => {
  return `/api/v1/reconciling/rehome`;
};

/**
 * @summary Save the old Controller's library roots at their places on this machine
 */
export const rehomeReconciling = async (
  reconcilingRehomeRequest: ReconcilingRehomeRequest,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReconcilingRehomed> => {
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
  return apiFetch<ReconcilingRehomed>(getRehomeReconcilingUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(reconcilingRehomeRequest),
  });
};

export const getRehomeReconcilingMutationKey = () => ["rehomeReconciling"] as const;

export const getRehomeReconcilingMutationOptions = <
  TError = ErrorType<
    | RehomeReconciling401
    | RehomeReconciling403
    | RehomeReconciling404
    | RehomeReconciling409
    | RehomeReconciling422
    | RehomeReconciling500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof rehomeReconciling>>,
    TError,
    RehomeReconcilingMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof rehomeReconciling>>,
  TError,
  RehomeReconcilingMutationVariables,
  TContext
> => {
  const mutationKey = getRehomeReconcilingMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof rehomeReconciling>>,
    RehomeReconcilingMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return rehomeReconciling(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RehomeReconcilingMutationResult = NonNullable<
  Awaited<ReturnType<typeof rehomeReconciling>>
>;
export type RehomeReconcilingMutationBody = ReconcilingRehomeRequest;
export type RehomeReconcilingMutationError = ErrorType<
  | RehomeReconciling401
  | RehomeReconciling403
  | RehomeReconciling404
  | RehomeReconciling409
  | RehomeReconciling422
  | RehomeReconciling500
>;
export type RehomeReconcilingMutationVariables = { data: ReconcilingRehomeRequest };

/**
 * @summary Save the old Controller's library roots at their places on this machine
 */
export const useRehomeReconciling = <
  TError = ErrorType<
    | RehomeReconciling401
    | RehomeReconciling403
    | RehomeReconciling404
    | RehomeReconciling409
    | RehomeReconciling422
    | RehomeReconciling500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof rehomeReconciling>>,
      TError,
      RehomeReconcilingMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof rehomeReconciling>>,
  TError,
  RehomeReconcilingMutationVariables,
  TContext
> => {
  return useMutation(getRehomeReconcilingMutationOptions(options), queryClient);
};
export const getContinueReconcilingRootUrl = (rootId: string) => {
  return `/api/v1/reconciling/roots/${rootId}/continue`;
};

/**
 * @summary Stop waiting for a library root's scan
 */
export const continueReconcilingRoot = async (
  rootId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getContinueReconcilingRootUrl(rootId), {
    ...options,
    method: "POST",
  });
};

export const getContinueReconcilingRootMutationKey = () => ["continueReconcilingRoot"] as const;

export const getContinueReconcilingRootMutationOptions = <
  TError = ErrorType<
    | ContinueReconcilingRoot401
    | ContinueReconcilingRoot403
    | ContinueReconcilingRoot404
    | ContinueReconcilingRoot409
    | ContinueReconcilingRoot422
    | ContinueReconcilingRoot500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof continueReconcilingRoot>>,
    TError,
    ContinueReconcilingRootMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof continueReconcilingRoot>>,
  TError,
  ContinueReconcilingRootMutationVariables,
  TContext
> => {
  const mutationKey = getContinueReconcilingRootMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof continueReconcilingRoot>>,
    ContinueReconcilingRootMutationVariables
  > = (props) => {
    const { rootId } = props ?? {};

    return continueReconcilingRoot(rootId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ContinueReconcilingRootMutationResult = NonNullable<
  Awaited<ReturnType<typeof continueReconcilingRoot>>
>;

export type ContinueReconcilingRootMutationError = ErrorType<
  | ContinueReconcilingRoot401
  | ContinueReconcilingRoot403
  | ContinueReconcilingRoot404
  | ContinueReconcilingRoot409
  | ContinueReconcilingRoot422
  | ContinueReconcilingRoot500
>;
export type ContinueReconcilingRootMutationVariables = { rootId: string };

/**
 * @summary Stop waiting for a library root's scan
 */
export const useContinueReconcilingRoot = <
  TError = ErrorType<
    | ContinueReconcilingRoot401
    | ContinueReconcilingRoot403
    | ContinueReconcilingRoot404
    | ContinueReconcilingRoot409
    | ContinueReconcilingRoot422
    | ContinueReconcilingRoot500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof continueReconcilingRoot>>,
      TError,
      ContinueReconcilingRootMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof continueReconcilingRoot>>,
  TError,
  ContinueReconcilingRootMutationVariables,
  TContext
> => {
  return useMutation(getContinueReconcilingRootMutationOptions(options), queryClient);
};
export const getListReconcilingTorrentsUrl = (params?: ListReconcilingTorrentsParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/reconciling/torrents?${stringifiedParams}`
    : `/api/v1/reconciling/torrents`;
};

/**
 * @summary List a page of the review list
 */
export const listReconcilingTorrents = async (
  params?: ListReconcilingTorrentsParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReconcilingReviewPage> => {
  return apiFetch<ReconcilingReviewPage>(getListReconcilingTorrentsUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListReconcilingTorrentsQueryKey = (params?: ListReconcilingTorrentsParams) => {
  return [`/api/v1/reconciling/torrents`, ...(params ? [params] : [])] as const;
};

export const getListReconcilingTorrentsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listReconcilingTorrents>>,
  TError = ErrorType<
    | ListReconcilingTorrents401
    | ListReconcilingTorrents403
    | ListReconcilingTorrents409
    | ListReconcilingTorrents422
    | ListReconcilingTorrents500
  >,
>(
  params?: ListReconcilingTorrentsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listReconcilingTorrents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListReconcilingTorrentsQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listReconcilingTorrents>>> = ({
    signal,
  }) => listReconcilingTorrents(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listReconcilingTorrents>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListReconcilingTorrentsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listReconcilingTorrents>>
>;
export type ListReconcilingTorrentsSuspenseQueryError = ErrorType<
  | ListReconcilingTorrents401
  | ListReconcilingTorrents403
  | ListReconcilingTorrents409
  | ListReconcilingTorrents422
  | ListReconcilingTorrents500
>;

export function useListReconcilingTorrentsSuspense<
  TData = Awaited<ReturnType<typeof listReconcilingTorrents>>,
  TError = ErrorType<
    | ListReconcilingTorrents401
    | ListReconcilingTorrents403
    | ListReconcilingTorrents409
    | ListReconcilingTorrents422
    | ListReconcilingTorrents500
  >,
>(
  params: undefined | ListReconcilingTorrentsParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listReconcilingTorrents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListReconcilingTorrentsSuspense<
  TData = Awaited<ReturnType<typeof listReconcilingTorrents>>,
  TError = ErrorType<
    | ListReconcilingTorrents401
    | ListReconcilingTorrents403
    | ListReconcilingTorrents409
    | ListReconcilingTorrents422
    | ListReconcilingTorrents500
  >,
>(
  params?: ListReconcilingTorrentsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listReconcilingTorrents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListReconcilingTorrentsSuspense<
  TData = Awaited<ReturnType<typeof listReconcilingTorrents>>,
  TError = ErrorType<
    | ListReconcilingTorrents401
    | ListReconcilingTorrents403
    | ListReconcilingTorrents409
    | ListReconcilingTorrents422
    | ListReconcilingTorrents500
  >,
>(
  params?: ListReconcilingTorrentsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listReconcilingTorrents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List a page of the review list
 */

export function useListReconcilingTorrentsSuspense<
  TData = Awaited<ReturnType<typeof listReconcilingTorrents>>,
  TError = ErrorType<
    | ListReconcilingTorrents401
    | ListReconcilingTorrents403
    | ListReconcilingTorrents409
    | ListReconcilingTorrents422
    | ListReconcilingTorrents500
  >,
>(
  params?: ListReconcilingTorrentsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listReconcilingTorrents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListReconcilingTorrentsSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getForgetReconcilingTorrentsUrl = () => {
  return `/api/v1/reconciling/torrents/forget`;
};

/**
 * @summary Forget torrents no Node holds
 */
export const forgetReconcilingTorrents = async (
  reconcilingForgetRequest: ReconcilingForgetRequest,
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
  return apiFetch<void>(getForgetReconcilingTorrentsUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(reconcilingForgetRequest),
  });
};

export const getForgetReconcilingTorrentsMutationKey = () => ["forgetReconcilingTorrents"] as const;

export const getForgetReconcilingTorrentsMutationOptions = <
  TError = ErrorType<
    | ForgetReconcilingTorrents401
    | ForgetReconcilingTorrents403
    | ForgetReconcilingTorrents404
    | ForgetReconcilingTorrents409
    | ForgetReconcilingTorrents422
    | ForgetReconcilingTorrents500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof forgetReconcilingTorrents>>,
    TError,
    ForgetReconcilingTorrentsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof forgetReconcilingTorrents>>,
  TError,
  ForgetReconcilingTorrentsMutationVariables,
  TContext
> => {
  const mutationKey = getForgetReconcilingTorrentsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof forgetReconcilingTorrents>>,
    ForgetReconcilingTorrentsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return forgetReconcilingTorrents(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ForgetReconcilingTorrentsMutationResult = NonNullable<
  Awaited<ReturnType<typeof forgetReconcilingTorrents>>
>;
export type ForgetReconcilingTorrentsMutationBody = ReconcilingForgetRequest;
export type ForgetReconcilingTorrentsMutationError = ErrorType<
  | ForgetReconcilingTorrents401
  | ForgetReconcilingTorrents403
  | ForgetReconcilingTorrents404
  | ForgetReconcilingTorrents409
  | ForgetReconcilingTorrents422
  | ForgetReconcilingTorrents500
>;
export type ForgetReconcilingTorrentsMutationVariables = { data: ReconcilingForgetRequest };

/**
 * @summary Forget torrents no Node holds
 */
export const useForgetReconcilingTorrents = <
  TError = ErrorType<
    | ForgetReconcilingTorrents401
    | ForgetReconcilingTorrents403
    | ForgetReconcilingTorrents404
    | ForgetReconcilingTorrents409
    | ForgetReconcilingTorrents422
    | ForgetReconcilingTorrents500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof forgetReconcilingTorrents>>,
      TError,
      ForgetReconcilingTorrentsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof forgetReconcilingTorrents>>,
  TError,
  ForgetReconcilingTorrentsMutationVariables,
  TContext
> => {
  return useMutation(getForgetReconcilingTorrentsMutationOptions(options), queryClient);
};
export const getReaddReconcilingTorrentsUrl = () => {
  return `/api/v1/reconciling/torrents/readd`;
};

/**
 * @summary Re-add torrents on the review list
 */
export const readdReconcilingTorrents = async (
  reconcilingReaddRequest: ReconcilingReaddRequest,
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
  return apiFetch<void>(getReaddReconcilingTorrentsUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(reconcilingReaddRequest),
  });
};

export const getReaddReconcilingTorrentsMutationKey = () => ["readdReconcilingTorrents"] as const;

export const getReaddReconcilingTorrentsMutationOptions = <
  TError = ErrorType<
    | ReaddReconcilingTorrents401
    | ReaddReconcilingTorrents403
    | ReaddReconcilingTorrents404
    | ReaddReconcilingTorrents409
    | ReaddReconcilingTorrents422
    | ReaddReconcilingTorrents500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof readdReconcilingTorrents>>,
    TError,
    ReaddReconcilingTorrentsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof readdReconcilingTorrents>>,
  TError,
  ReaddReconcilingTorrentsMutationVariables,
  TContext
> => {
  const mutationKey = getReaddReconcilingTorrentsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof readdReconcilingTorrents>>,
    ReaddReconcilingTorrentsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return readdReconcilingTorrents(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ReaddReconcilingTorrentsMutationResult = NonNullable<
  Awaited<ReturnType<typeof readdReconcilingTorrents>>
>;
export type ReaddReconcilingTorrentsMutationBody = ReconcilingReaddRequest;
export type ReaddReconcilingTorrentsMutationError = ErrorType<
  | ReaddReconcilingTorrents401
  | ReaddReconcilingTorrents403
  | ReaddReconcilingTorrents404
  | ReaddReconcilingTorrents409
  | ReaddReconcilingTorrents422
  | ReaddReconcilingTorrents500
>;
export type ReaddReconcilingTorrentsMutationVariables = { data: ReconcilingReaddRequest };

/**
 * @summary Re-add torrents on the review list
 */
export const useReaddReconcilingTorrents = <
  TError = ErrorType<
    | ReaddReconcilingTorrents401
    | ReaddReconcilingTorrents403
    | ReaddReconcilingTorrents404
    | ReaddReconcilingTorrents409
    | ReaddReconcilingTorrents422
    | ReaddReconcilingTorrents500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof readdReconcilingTorrents>>,
      TError,
      ReaddReconcilingTorrentsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof readdReconcilingTorrents>>,
  TError,
  ReaddReconcilingTorrentsMutationVariables,
  TContext
> => {
  return useMutation(getReaddReconcilingTorrentsMutationOptions(options), queryClient);
};
