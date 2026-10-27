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
  CreateIndexer401,
  CreateIndexer403,
  CreateIndexer409,
  CreateIndexer422,
  CreateIndexer500,
  CreateIndexer502,
  DeleteIndexer401,
  DeleteIndexer403,
  DeleteIndexer404,
  DeleteIndexer409,
  DeleteIndexer422,
  DeleteIndexer500,
  GetIndexer401,
  GetIndexer403,
  GetIndexer404,
  GetIndexer409,
  GetIndexer422,
  GetIndexer500,
  IndexersIndexer,
  IndexersIndexerEdit,
  IndexersIndexerList,
  IndexersNewIndexer,
  IndexersQueryList,
  IndexersReport,
  ListIndexerQueries401,
  ListIndexerQueries403,
  ListIndexerQueries404,
  ListIndexerQueries409,
  ListIndexerQueries422,
  ListIndexerQueries500,
  ListIndexerQueriesParams,
  ListIndexers401,
  ListIndexers403,
  ListIndexers409,
  ListIndexers500,
  TestIndexer401,
  TestIndexer403,
  TestIndexer404,
  TestIndexer409,
  TestIndexer422,
  TestIndexer500,
  TestIndexer502,
  TestIndexerDraft401,
  TestIndexerDraft403,
  TestIndexerDraft409,
  TestIndexerDraft422,
  TestIndexerDraft500,
  TestIndexerDraft502,
  UpdateIndexer401,
  UpdateIndexer403,
  UpdateIndexer404,
  UpdateIndexer409,
  UpdateIndexer422,
  UpdateIndexer500,
  UpdateIndexer502,
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

export const getListIndexersUrl = () => {
  return `/api/v1/indexers`;
};

/**
 * @summary List the Indexers
 */
export const listIndexers = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersIndexerList> => {
  return apiFetch<IndexersIndexerList>(getListIndexersUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListIndexersQueryKey = () => {
  return [`/api/v1/indexers`] as const;
};

export const getListIndexersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listIndexers>>,
  TError = ErrorType<ListIndexers401 | ListIndexers403 | ListIndexers409 | ListIndexers500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexers>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListIndexersQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listIndexers>>> = ({ signal }) =>
    listIndexers({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexers>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListIndexersSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listIndexers>>>;
export type ListIndexersSuspenseQueryError = ErrorType<
  ListIndexers401 | ListIndexers403 | ListIndexers409 | ListIndexers500
>;

export function useListIndexersSuspense<
  TData = Awaited<ReturnType<typeof listIndexers>>,
  TError = ErrorType<ListIndexers401 | ListIndexers403 | ListIndexers409 | ListIndexers500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListIndexersSuspense<
  TData = Awaited<ReturnType<typeof listIndexers>>,
  TError = ErrorType<ListIndexers401 | ListIndexers403 | ListIndexers409 | ListIndexers500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListIndexersSuspense<
  TData = Awaited<ReturnType<typeof listIndexers>>,
  TError = ErrorType<ListIndexers401 | ListIndexers403 | ListIndexers409 | ListIndexers500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Indexers
 */

export function useListIndexersSuspense<
  TData = Awaited<ReturnType<typeof listIndexers>>,
  TError = ErrorType<ListIndexers401 | ListIndexers403 | ListIndexers409 | ListIndexers500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListIndexersSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCreateIndexerUrl = () => {
  return `/api/v1/indexers`;
};

/**
 * @summary Add an Indexer once its Test passed on the Node it searches from
 */
export const createIndexer = async (
  indexersNewIndexer: IndexersNewIndexer,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersIndexer> => {
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
  return apiFetch<IndexersIndexer>(getCreateIndexerUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(indexersNewIndexer),
  });
};

export const getCreateIndexerMutationKey = () => ["createIndexer"] as const;

export const getCreateIndexerMutationOptions = <
  TError = ErrorType<
    | CreateIndexer401
    | CreateIndexer403
    | CreateIndexer409
    | CreateIndexer422
    | CreateIndexer500
    | CreateIndexer502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof createIndexer>>,
    TError,
    CreateIndexerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof createIndexer>>,
  TError,
  CreateIndexerMutationVariables,
  TContext
> => {
  const mutationKey = getCreateIndexerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof createIndexer>>,
    CreateIndexerMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return createIndexer(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateIndexerMutationResult = NonNullable<Awaited<ReturnType<typeof createIndexer>>>;
export type CreateIndexerMutationBody = IndexersNewIndexer;
export type CreateIndexerMutationError = ErrorType<
  | CreateIndexer401
  | CreateIndexer403
  | CreateIndexer409
  | CreateIndexer422
  | CreateIndexer500
  | CreateIndexer502
>;
export type CreateIndexerMutationVariables = { data: IndexersNewIndexer };

/**
 * @summary Add an Indexer once its Test passed on the Node it searches from
 */
export const useCreateIndexer = <
  TError = ErrorType<
    | CreateIndexer401
    | CreateIndexer403
    | CreateIndexer409
    | CreateIndexer422
    | CreateIndexer500
    | CreateIndexer502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createIndexer>>,
      TError,
      CreateIndexerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof createIndexer>>,
  TError,
  CreateIndexerMutationVariables,
  TContext
> => {
  return useMutation(getCreateIndexerMutationOptions(options), queryClient);
};
export const getTestIndexerDraftUrl = () => {
  return `/api/v1/indexers/test`;
};

/**
 * @summary Test an Indexer without saving it
 */
export const testIndexerDraft = async (
  indexersNewIndexer: IndexersNewIndexer,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersReport> => {
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
  return apiFetch<IndexersReport>(getTestIndexerDraftUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(indexersNewIndexer),
  });
};

export const getTestIndexerDraftMutationKey = () => ["testIndexerDraft"] as const;

export const getTestIndexerDraftMutationOptions = <
  TError = ErrorType<
    | TestIndexerDraft401
    | TestIndexerDraft403
    | TestIndexerDraft409
    | TestIndexerDraft422
    | TestIndexerDraft500
    | TestIndexerDraft502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof testIndexerDraft>>,
    TError,
    TestIndexerDraftMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof testIndexerDraft>>,
  TError,
  TestIndexerDraftMutationVariables,
  TContext
> => {
  const mutationKey = getTestIndexerDraftMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof testIndexerDraft>>,
    TestIndexerDraftMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return testIndexerDraft(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type TestIndexerDraftMutationResult = NonNullable<
  Awaited<ReturnType<typeof testIndexerDraft>>
>;
export type TestIndexerDraftMutationBody = IndexersNewIndexer;
export type TestIndexerDraftMutationError = ErrorType<
  | TestIndexerDraft401
  | TestIndexerDraft403
  | TestIndexerDraft409
  | TestIndexerDraft422
  | TestIndexerDraft500
  | TestIndexerDraft502
>;
export type TestIndexerDraftMutationVariables = { data: IndexersNewIndexer };

/**
 * @summary Test an Indexer without saving it
 */
export const useTestIndexerDraft = <
  TError = ErrorType<
    | TestIndexerDraft401
    | TestIndexerDraft403
    | TestIndexerDraft409
    | TestIndexerDraft422
    | TestIndexerDraft500
    | TestIndexerDraft502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof testIndexerDraft>>,
      TError,
      TestIndexerDraftMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof testIndexerDraft>>,
  TError,
  TestIndexerDraftMutationVariables,
  TContext
> => {
  return useMutation(getTestIndexerDraftMutationOptions(options), queryClient);
};
export const getDeleteIndexerUrl = (id: string) => {
  return `/api/v1/indexers/${id}`;
};

/**
 * @summary Delete an Indexer with its status, sessions and queries. Its Grabs keep their history.
 */
export const deleteIndexer = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeleteIndexerUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteIndexerMutationKey = () => ["deleteIndexer"] as const;

export const getDeleteIndexerMutationOptions = <
  TError = ErrorType<
    | DeleteIndexer401
    | DeleteIndexer403
    | DeleteIndexer404
    | DeleteIndexer409
    | DeleteIndexer422
    | DeleteIndexer500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteIndexer>>,
    TError,
    DeleteIndexerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteIndexer>>,
  TError,
  DeleteIndexerMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteIndexerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteIndexer>>,
    DeleteIndexerMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return deleteIndexer(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteIndexerMutationResult = NonNullable<Awaited<ReturnType<typeof deleteIndexer>>>;

export type DeleteIndexerMutationError = ErrorType<
  | DeleteIndexer401
  | DeleteIndexer403
  | DeleteIndexer404
  | DeleteIndexer409
  | DeleteIndexer422
  | DeleteIndexer500
>;
export type DeleteIndexerMutationVariables = { id: string };

/**
 * @summary Delete an Indexer with its status, sessions and queries. Its Grabs keep their history.
 */
export const useDeleteIndexer = <
  TError = ErrorType<
    | DeleteIndexer401
    | DeleteIndexer403
    | DeleteIndexer404
    | DeleteIndexer409
    | DeleteIndexer422
    | DeleteIndexer500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteIndexer>>,
      TError,
      DeleteIndexerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteIndexer>>,
  TError,
  DeleteIndexerMutationVariables,
  TContext
> => {
  return useMutation(getDeleteIndexerMutationOptions(options), queryClient);
};
export const getGetIndexerUrl = (id: string) => {
  return `/api/v1/indexers/${id}`;
};

/**
 * @summary Get an Indexer
 */
export const getIndexer = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersIndexer> => {
  return apiFetch<IndexersIndexer>(getGetIndexerUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetIndexerQueryKey = (id: string) => {
  return [`/api/v1/indexers/${id}`] as const;
};

export const getGetIndexerSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getIndexer>>,
  TError = ErrorType<
    GetIndexer401 | GetIndexer403 | GetIndexer404 | GetIndexer409 | GetIndexer422 | GetIndexer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexer>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetIndexerQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getIndexer>>> = ({ signal }) =>
    getIndexer(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexer>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetIndexerSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getIndexer>>>;
export type GetIndexerSuspenseQueryError = ErrorType<
  GetIndexer401 | GetIndexer403 | GetIndexer404 | GetIndexer409 | GetIndexer422 | GetIndexer500
>;

export function useGetIndexerSuspense<
  TData = Awaited<ReturnType<typeof getIndexer>>,
  TError = ErrorType<
    GetIndexer401 | GetIndexer403 | GetIndexer404 | GetIndexer409 | GetIndexer422 | GetIndexer500
  >,
>(
  id: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexer>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetIndexerSuspense<
  TData = Awaited<ReturnType<typeof getIndexer>>,
  TError = ErrorType<
    GetIndexer401 | GetIndexer403 | GetIndexer404 | GetIndexer409 | GetIndexer422 | GetIndexer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexer>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetIndexerSuspense<
  TData = Awaited<ReturnType<typeof getIndexer>>,
  TError = ErrorType<
    GetIndexer401 | GetIndexer403 | GetIndexer404 | GetIndexer409 | GetIndexer422 | GetIndexer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexer>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get an Indexer
 */

export function useGetIndexerSuspense<
  TData = Awaited<ReturnType<typeof getIndexer>>,
  TError = ErrorType<
    GetIndexer401 | GetIndexer403 | GetIndexer404 | GetIndexer409 | GetIndexer422 | GetIndexer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexer>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetIndexerSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateIndexerUrl = (id: string) => {
  return `/api/v1/indexers/${id}`;
};

/**
 * @summary Change an Indexer, testing it first when the change is to its home, address or API key
 */
export const updateIndexer = async (
  id: string,
  indexersIndexerEdit: IndexersIndexerEdit,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersIndexer> => {
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
  return apiFetch<IndexersIndexer>(getUpdateIndexerUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(indexersIndexerEdit),
  });
};

export const getUpdateIndexerMutationKey = () => ["updateIndexer"] as const;

export const getUpdateIndexerMutationOptions = <
  TError = ErrorType<
    | UpdateIndexer401
    | UpdateIndexer403
    | UpdateIndexer404
    | UpdateIndexer409
    | UpdateIndexer422
    | UpdateIndexer500
    | UpdateIndexer502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateIndexer>>,
    TError,
    UpdateIndexerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateIndexer>>,
  TError,
  UpdateIndexerMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateIndexerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateIndexer>>,
    UpdateIndexerMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return updateIndexer(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateIndexerMutationResult = NonNullable<Awaited<ReturnType<typeof updateIndexer>>>;
export type UpdateIndexerMutationBody = IndexersIndexerEdit;
export type UpdateIndexerMutationError = ErrorType<
  | UpdateIndexer401
  | UpdateIndexer403
  | UpdateIndexer404
  | UpdateIndexer409
  | UpdateIndexer422
  | UpdateIndexer500
  | UpdateIndexer502
>;
export type UpdateIndexerMutationVariables = { id: string; data: IndexersIndexerEdit };

/**
 * @summary Change an Indexer, testing it first when the change is to its home, address or API key
 */
export const useUpdateIndexer = <
  TError = ErrorType<
    | UpdateIndexer401
    | UpdateIndexer403
    | UpdateIndexer404
    | UpdateIndexer409
    | UpdateIndexer422
    | UpdateIndexer500
    | UpdateIndexer502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateIndexer>>,
      TError,
      UpdateIndexerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateIndexer>>,
  TError,
  UpdateIndexerMutationVariables,
  TContext
> => {
  return useMutation(getUpdateIndexerMutationOptions(options), queryClient);
};
export const getListIndexerQueriesUrl = (id: string, params?: ListIndexerQueriesParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/indexers/${id}/queries?${stringifiedParams}`
    : `/api/v1/indexers/${id}/queries`;
};

/**
 * @summary List an Indexer's searches, fetches and Tests, 50 at a time
 */
export const listIndexerQueries = async (
  id: string,
  params?: ListIndexerQueriesParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersQueryList> => {
  return apiFetch<IndexersQueryList>(getListIndexerQueriesUrl(id, params), {
    ...options,
    method: "GET",
  });
};

export const getListIndexerQueriesQueryKey = (id: string, params?: ListIndexerQueriesParams) => {
  return [`/api/v1/indexers/${id}/queries`, ...(params ? [params] : [])] as const;
};

export const getListIndexerQueriesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listIndexerQueries>>,
  TError = ErrorType<
    | ListIndexerQueries401
    | ListIndexerQueries403
    | ListIndexerQueries404
    | ListIndexerQueries409
    | ListIndexerQueries422
    | ListIndexerQueries500
  >,
>(
  id: string,
  params?: ListIndexerQueriesParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerQueries>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListIndexerQueriesQueryKey(id, params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listIndexerQueries>>> = ({ signal }) =>
    listIndexerQueries(id, params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerQueries>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListIndexerQueriesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listIndexerQueries>>
>;
export type ListIndexerQueriesSuspenseQueryError = ErrorType<
  | ListIndexerQueries401
  | ListIndexerQueries403
  | ListIndexerQueries404
  | ListIndexerQueries409
  | ListIndexerQueries422
  | ListIndexerQueries500
>;

export function useListIndexerQueriesSuspense<
  TData = Awaited<ReturnType<typeof listIndexerQueries>>,
  TError = ErrorType<
    | ListIndexerQueries401
    | ListIndexerQueries403
    | ListIndexerQueries404
    | ListIndexerQueries409
    | ListIndexerQueries422
    | ListIndexerQueries500
  >,
>(
  id: string,
  params: undefined | ListIndexerQueriesParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerQueries>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListIndexerQueriesSuspense<
  TData = Awaited<ReturnType<typeof listIndexerQueries>>,
  TError = ErrorType<
    | ListIndexerQueries401
    | ListIndexerQueries403
    | ListIndexerQueries404
    | ListIndexerQueries409
    | ListIndexerQueries422
    | ListIndexerQueries500
  >,
>(
  id: string,
  params?: ListIndexerQueriesParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerQueries>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListIndexerQueriesSuspense<
  TData = Awaited<ReturnType<typeof listIndexerQueries>>,
  TError = ErrorType<
    | ListIndexerQueries401
    | ListIndexerQueries403
    | ListIndexerQueries404
    | ListIndexerQueries409
    | ListIndexerQueries422
    | ListIndexerQueries500
  >,
>(
  id: string,
  params?: ListIndexerQueriesParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerQueries>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List an Indexer's searches, fetches and Tests, 50 at a time
 */

export function useListIndexerQueriesSuspense<
  TData = Awaited<ReturnType<typeof listIndexerQueries>>,
  TError = ErrorType<
    | ListIndexerQueries401
    | ListIndexerQueries403
    | ListIndexerQueries404
    | ListIndexerQueries409
    | ListIndexerQueries422
    | ListIndexerQueries500
  >,
>(
  id: string,
  params?: ListIndexerQueriesParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listIndexerQueries>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListIndexerQueriesSuspenseQueryOptions(id, params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getTestIndexerUrl = (id: string) => {
  return `/api/v1/indexers/${id}/test`;
};

/**
 * @summary Test an Indexer from the Node it searches from. A pass clears its backoff and any stop.
 */
export const testIndexer = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersReport> => {
  return apiFetch<IndexersReport>(getTestIndexerUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getTestIndexerMutationKey = () => ["testIndexer"] as const;

export const getTestIndexerMutationOptions = <
  TError = ErrorType<
    | TestIndexer401
    | TestIndexer403
    | TestIndexer404
    | TestIndexer409
    | TestIndexer422
    | TestIndexer500
    | TestIndexer502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof testIndexer>>,
    TError,
    TestIndexerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof testIndexer>>,
  TError,
  TestIndexerMutationVariables,
  TContext
> => {
  const mutationKey = getTestIndexerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof testIndexer>>,
    TestIndexerMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return testIndexer(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type TestIndexerMutationResult = NonNullable<Awaited<ReturnType<typeof testIndexer>>>;

export type TestIndexerMutationError = ErrorType<
  | TestIndexer401
  | TestIndexer403
  | TestIndexer404
  | TestIndexer409
  | TestIndexer422
  | TestIndexer500
  | TestIndexer502
>;
export type TestIndexerMutationVariables = { id: string };

/**
 * @summary Test an Indexer from the Node it searches from. A pass clears its backoff and any stop.
 */
export const useTestIndexer = <
  TError = ErrorType<
    | TestIndexer401
    | TestIndexer403
    | TestIndexer404
    | TestIndexer409
    | TestIndexer422
    | TestIndexer500
    | TestIndexer502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof testIndexer>>,
      TError,
      TestIndexerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof testIndexer>>,
  TError,
  TestIndexerMutationVariables,
  TContext
> => {
  return useMutation(getTestIndexerMutationOptions(options), queryClient);
};
