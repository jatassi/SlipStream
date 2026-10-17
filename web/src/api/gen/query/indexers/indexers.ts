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
  IndexersIndexer,
  IndexersIndexerList,
  IndexersNewAPIKey,
  IndexersNewIndexer,
  ListIndexers401,
  ListIndexers403,
  ListIndexers409,
  ListIndexers500,
  SetIndexerAPIKey401,
  SetIndexerAPIKey403,
  SetIndexerAPIKey404,
  SetIndexerAPIKey409,
  SetIndexerAPIKey422,
  SetIndexerAPIKey500,
  SetIndexerAPIKey502,
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
 * @summary Add a Torznab Indexer, once its home Node has read its caps
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
 * @summary Add a Torznab Indexer, once its home Node has read its caps
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
export const getSetIndexerAPIKeyUrl = (id: string) => {
  return `/api/v1/indexers/${id}/api-key`;
};

/**
 * @summary Enter an Indexer's API key again, once its home Node has read its caps with it
 */
export const setIndexerAPIKey = async (
  id: string,
  indexersNewAPIKey: IndexersNewAPIKey,
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
  return apiFetch<IndexersIndexer>(getSetIndexerAPIKeyUrl(id), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(indexersNewAPIKey),
  });
};

export const getSetIndexerAPIKeyMutationKey = () => ["setIndexerAPIKey"] as const;

export const getSetIndexerAPIKeyMutationOptions = <
  TError = ErrorType<
    | SetIndexerAPIKey401
    | SetIndexerAPIKey403
    | SetIndexerAPIKey404
    | SetIndexerAPIKey409
    | SetIndexerAPIKey422
    | SetIndexerAPIKey500
    | SetIndexerAPIKey502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setIndexerAPIKey>>,
    TError,
    SetIndexerAPIKeyMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setIndexerAPIKey>>,
  TError,
  SetIndexerAPIKeyMutationVariables,
  TContext
> => {
  const mutationKey = getSetIndexerAPIKeyMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setIndexerAPIKey>>,
    SetIndexerAPIKeyMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return setIndexerAPIKey(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetIndexerAPIKeyMutationResult = NonNullable<
  Awaited<ReturnType<typeof setIndexerAPIKey>>
>;
export type SetIndexerAPIKeyMutationBody = IndexersNewAPIKey;
export type SetIndexerAPIKeyMutationError = ErrorType<
  | SetIndexerAPIKey401
  | SetIndexerAPIKey403
  | SetIndexerAPIKey404
  | SetIndexerAPIKey409
  | SetIndexerAPIKey422
  | SetIndexerAPIKey500
  | SetIndexerAPIKey502
>;
export type SetIndexerAPIKeyMutationVariables = { id: string; data: IndexersNewAPIKey };

/**
 * @summary Enter an Indexer's API key again, once its home Node has read its caps with it
 */
export const useSetIndexerAPIKey = <
  TError = ErrorType<
    | SetIndexerAPIKey401
    | SetIndexerAPIKey403
    | SetIndexerAPIKey404
    | SetIndexerAPIKey409
    | SetIndexerAPIKey422
    | SetIndexerAPIKey500
    | SetIndexerAPIKey502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setIndexerAPIKey>>,
      TError,
      SetIndexerAPIKeyMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setIndexerAPIKey>>,
  TError,
  SetIndexerAPIKeyMutationVariables,
  TContext
> => {
  return useMutation(getSetIndexerAPIKeyMutationOptions(options), queryClient);
};
