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
  AddCustomDefinition401,
  AddCustomDefinition403,
  AddCustomDefinition409,
  AddCustomDefinition422,
  AddCustomDefinition500,
  CreateIndexer401,
  CreateIndexer403,
  CreateIndexer404,
  CreateIndexer409,
  CreateIndexer410,
  CreateIndexer422,
  CreateIndexer500,
  CreateIndexer502,
  CreateIndexer503,
  DeleteCustomDefinition401,
  DeleteCustomDefinition403,
  DeleteCustomDefinition404,
  DeleteCustomDefinition409,
  DeleteCustomDefinition422,
  DeleteCustomDefinition500,
  DeleteIndexer401,
  DeleteIndexer403,
  DeleteIndexer404,
  DeleteIndexer409,
  DeleteIndexer422,
  DeleteIndexer500,
  GetCatalogDefinition401,
  GetCatalogDefinition403,
  GetCatalogDefinition404,
  GetCatalogDefinition409,
  GetCatalogDefinition422,
  GetCatalogDefinition500,
  GetCatalogDefinition503,
  GetIndexer401,
  GetIndexer403,
  GetIndexer404,
  GetIndexer409,
  GetIndexer422,
  GetIndexer500,
  GetIndexerCaptcha401,
  GetIndexerCaptcha403,
  GetIndexerCaptcha404,
  GetIndexerCaptcha409,
  GetIndexerCaptcha422,
  GetIndexerCaptcha500,
  GetIndexerCaptcha502,
  GetIndexerCaptchaDraft401,
  GetIndexerCaptchaDraft403,
  GetIndexerCaptchaDraft404,
  GetIndexerCaptchaDraft409,
  GetIndexerCaptchaDraft410,
  GetIndexerCaptchaDraft422,
  GetIndexerCaptchaDraft500,
  GetIndexerCaptchaDraft502,
  GetIndexerCaptchaDraft503,
  GetIndexerCatalog401,
  GetIndexerCatalog403,
  GetIndexerCatalog409,
  GetIndexerCatalog500,
  GetIndexerSettings401,
  GetIndexerSettings403,
  GetIndexerSettings409,
  GetIndexerSettings500,
  GetTracker401,
  GetTracker403,
  GetTracker404,
  GetTracker409,
  GetTracker422,
  GetTracker500,
  IndexersCaptchaImage,
  IndexersCatalogAddress,
  IndexersCatalogEntry,
  IndexersCustomDefinition,
  IndexersDefinitionForm,
  IndexersIndexer,
  IndexersIndexerCatalog,
  IndexersIndexerEdit,
  IndexersIndexerList,
  IndexersNewIndexer,
  IndexersQueryList,
  IndexersReport,
  IndexersSettings,
  IndexersSettingsChange,
  IndexersTestBody,
  IndexersTracker,
  IndexersTrackerList,
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
  ListTrackers401,
  ListTrackers403,
  ListTrackers409,
  ListTrackers500,
  MergeTracker401,
  MergeTracker403,
  MergeTracker404,
  MergeTracker409,
  MergeTracker422,
  MergeTracker500,
  MergeTrackerInputBody,
  RenameTracker401,
  RenameTracker403,
  RenameTracker404,
  RenameTracker409,
  RenameTracker422,
  RenameTracker500,
  RenameTrackerInputBody,
  SetCatalogAddress401,
  SetCatalogAddress403,
  SetCatalogAddress409,
  SetCatalogAddress422,
  SetCatalogAddress500,
  TestIndexer401,
  TestIndexer403,
  TestIndexer404,
  TestIndexer409,
  TestIndexer410,
  TestIndexer422,
  TestIndexer500,
  TestIndexer502,
  TestIndexerDraft401,
  TestIndexerDraft403,
  TestIndexerDraft404,
  TestIndexerDraft409,
  TestIndexerDraft410,
  TestIndexerDraft422,
  TestIndexerDraft500,
  TestIndexerDraft502,
  TestIndexerDraft503,
  UpdateIndexer401,
  UpdateIndexer403,
  UpdateIndexer404,
  UpdateIndexer409,
  UpdateIndexer410,
  UpdateIndexer422,
  UpdateIndexer500,
  UpdateIndexer502,
  UpdateIndexerSettings401,
  UpdateIndexerSettings403,
  UpdateIndexerSettings409,
  UpdateIndexerSettings422,
  UpdateIndexerSettings500,
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

export const getGetIndexerCatalogUrl = () => {
  return `/api/v1/indexer-catalog`;
};

/**
 * @summary List the Indexer catalog's definitions and the custom ones, with when the catalog last refreshed
 */
export const getIndexerCatalog = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersIndexerCatalog> => {
  return apiFetch<IndexersIndexerCatalog>(getGetIndexerCatalogUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetIndexerCatalogQueryKey = () => {
  return [`/api/v1/indexer-catalog`] as const;
};

export const getGetIndexerCatalogSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getIndexerCatalog>>,
  TError = ErrorType<
    GetIndexerCatalog401 | GetIndexerCatalog403 | GetIndexerCatalog409 | GetIndexerCatalog500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerCatalog>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetIndexerCatalogQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getIndexerCatalog>>> = ({ signal }) =>
    getIndexerCatalog({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerCatalog>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetIndexerCatalogSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getIndexerCatalog>>
>;
export type GetIndexerCatalogSuspenseQueryError = ErrorType<
  GetIndexerCatalog401 | GetIndexerCatalog403 | GetIndexerCatalog409 | GetIndexerCatalog500
>;

export function useGetIndexerCatalogSuspense<
  TData = Awaited<ReturnType<typeof getIndexerCatalog>>,
  TError = ErrorType<
    GetIndexerCatalog401 | GetIndexerCatalog403 | GetIndexerCatalog409 | GetIndexerCatalog500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerCatalog>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetIndexerCatalogSuspense<
  TData = Awaited<ReturnType<typeof getIndexerCatalog>>,
  TError = ErrorType<
    GetIndexerCatalog401 | GetIndexerCatalog403 | GetIndexerCatalog409 | GetIndexerCatalog500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerCatalog>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetIndexerCatalogSuspense<
  TData = Awaited<ReturnType<typeof getIndexerCatalog>>,
  TError = ErrorType<
    GetIndexerCatalog401 | GetIndexerCatalog403 | GetIndexerCatalog409 | GetIndexerCatalog500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerCatalog>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Indexer catalog's definitions and the custom ones, with when the catalog last refreshed
 */

export function useGetIndexerCatalogSuspense<
  TData = Awaited<ReturnType<typeof getIndexerCatalog>>,
  TError = ErrorType<
    GetIndexerCatalog401 | GetIndexerCatalog403 | GetIndexerCatalog409 | GetIndexerCatalog500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerCatalog>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetIndexerCatalogSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetCatalogAddressUrl = () => {
  return `/api/v1/indexer-catalog/address`;
};

/**
 * @summary Change the Indexer catalog's address once SlipStream reads a schema 12 index there. The next refresh reads it in full.
 */
export const setCatalogAddress = async (
  indexersCatalogAddress: IndexersCatalogAddress,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersIndexerCatalog> => {
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
  return apiFetch<IndexersIndexerCatalog>(getSetCatalogAddressUrl(), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(indexersCatalogAddress),
  });
};

export const getSetCatalogAddressMutationKey = () => ["setCatalogAddress"] as const;

export const getSetCatalogAddressMutationOptions = <
  TError = ErrorType<
    | SetCatalogAddress401
    | SetCatalogAddress403
    | SetCatalogAddress409
    | SetCatalogAddress422
    | SetCatalogAddress500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setCatalogAddress>>,
    TError,
    SetCatalogAddressMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setCatalogAddress>>,
  TError,
  SetCatalogAddressMutationVariables,
  TContext
> => {
  const mutationKey = getSetCatalogAddressMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setCatalogAddress>>,
    SetCatalogAddressMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return setCatalogAddress(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetCatalogAddressMutationResult = NonNullable<
  Awaited<ReturnType<typeof setCatalogAddress>>
>;
export type SetCatalogAddressMutationBody = IndexersCatalogAddress;
export type SetCatalogAddressMutationError = ErrorType<
  | SetCatalogAddress401
  | SetCatalogAddress403
  | SetCatalogAddress409
  | SetCatalogAddress422
  | SetCatalogAddress500
>;
export type SetCatalogAddressMutationVariables = { data: IndexersCatalogAddress };

/**
 * @summary Change the Indexer catalog's address once SlipStream reads a schema 12 index there. The next refresh reads it in full.
 */
export const useSetCatalogAddress = <
  TError = ErrorType<
    | SetCatalogAddress401
    | SetCatalogAddress403
    | SetCatalogAddress409
    | SetCatalogAddress422
    | SetCatalogAddress500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setCatalogAddress>>,
      TError,
      SetCatalogAddressMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setCatalogAddress>>,
  TError,
  SetCatalogAddressMutationVariables,
  TContext
> => {
  return useMutation(getSetCatalogAddressMutationOptions(options), queryClient);
};
export const getAddCustomDefinitionUrl = () => {
  return `/api/v1/indexer-catalog/custom`;
};

/**
 * @summary Add a custom definition, or replace the custom definition with its id and move that definition's Indexers to it
 */
export const addCustomDefinition = async (
  indexersCustomDefinition: IndexersCustomDefinition,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersCatalogEntry> => {
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
  return apiFetch<IndexersCatalogEntry>(getAddCustomDefinitionUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(indexersCustomDefinition),
  });
};

export const getAddCustomDefinitionMutationKey = () => ["addCustomDefinition"] as const;

export const getAddCustomDefinitionMutationOptions = <
  TError = ErrorType<
    | AddCustomDefinition401
    | AddCustomDefinition403
    | AddCustomDefinition409
    | AddCustomDefinition422
    | AddCustomDefinition500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof addCustomDefinition>>,
    TError,
    AddCustomDefinitionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof addCustomDefinition>>,
  TError,
  AddCustomDefinitionMutationVariables,
  TContext
> => {
  const mutationKey = getAddCustomDefinitionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof addCustomDefinition>>,
    AddCustomDefinitionMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return addCustomDefinition(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type AddCustomDefinitionMutationResult = NonNullable<
  Awaited<ReturnType<typeof addCustomDefinition>>
>;
export type AddCustomDefinitionMutationBody = IndexersCustomDefinition;
export type AddCustomDefinitionMutationError = ErrorType<
  | AddCustomDefinition401
  | AddCustomDefinition403
  | AddCustomDefinition409
  | AddCustomDefinition422
  | AddCustomDefinition500
>;
export type AddCustomDefinitionMutationVariables = { data: IndexersCustomDefinition };

/**
 * @summary Add a custom definition, or replace the custom definition with its id and move that definition's Indexers to it
 */
export const useAddCustomDefinition = <
  TError = ErrorType<
    | AddCustomDefinition401
    | AddCustomDefinition403
    | AddCustomDefinition409
    | AddCustomDefinition422
    | AddCustomDefinition500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof addCustomDefinition>>,
      TError,
      AddCustomDefinitionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof addCustomDefinition>>,
  TError,
  AddCustomDefinitionMutationVariables,
  TContext
> => {
  return useMutation(getAddCustomDefinitionMutationOptions(options), queryClient);
};
export const getDeleteCustomDefinitionUrl = (id: string) => {
  return `/api/v1/indexer-catalog/custom/${id}`;
};

/**
 * @summary Delete a custom definition that no Indexer runs
 */
export const deleteCustomDefinition = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeleteCustomDefinitionUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteCustomDefinitionMutationKey = () => ["deleteCustomDefinition"] as const;

export const getDeleteCustomDefinitionMutationOptions = <
  TError = ErrorType<
    | DeleteCustomDefinition401
    | DeleteCustomDefinition403
    | DeleteCustomDefinition404
    | DeleteCustomDefinition409
    | DeleteCustomDefinition422
    | DeleteCustomDefinition500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteCustomDefinition>>,
    TError,
    DeleteCustomDefinitionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteCustomDefinition>>,
  TError,
  DeleteCustomDefinitionMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteCustomDefinitionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteCustomDefinition>>,
    DeleteCustomDefinitionMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return deleteCustomDefinition(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteCustomDefinitionMutationResult = NonNullable<
  Awaited<ReturnType<typeof deleteCustomDefinition>>
>;

export type DeleteCustomDefinitionMutationError = ErrorType<
  | DeleteCustomDefinition401
  | DeleteCustomDefinition403
  | DeleteCustomDefinition404
  | DeleteCustomDefinition409
  | DeleteCustomDefinition422
  | DeleteCustomDefinition500
>;
export type DeleteCustomDefinitionMutationVariables = { id: string };

/**
 * @summary Delete a custom definition that no Indexer runs
 */
export const useDeleteCustomDefinition = <
  TError = ErrorType<
    | DeleteCustomDefinition401
    | DeleteCustomDefinition403
    | DeleteCustomDefinition404
    | DeleteCustomDefinition409
    | DeleteCustomDefinition422
    | DeleteCustomDefinition500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteCustomDefinition>>,
      TError,
      DeleteCustomDefinitionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteCustomDefinition>>,
  TError,
  DeleteCustomDefinitionMutationVariables,
  TContext
> => {
  return useMutation(getDeleteCustomDefinitionMutationOptions(options), queryClient);
};
export const getGetCatalogDefinitionUrl = (id: string) => {
  return `/api/v1/indexer-catalog/${id}`;
};

/**
 * @summary Get the fields of a definition's add form, from the custom definition with the id or else the catalog's
 */
export const getCatalogDefinition = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersDefinitionForm> => {
  return apiFetch<IndexersDefinitionForm>(getGetCatalogDefinitionUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetCatalogDefinitionQueryKey = (id: string) => {
  return [`/api/v1/indexer-catalog/${id}`] as const;
};

export const getGetCatalogDefinitionSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getCatalogDefinition>>,
  TError = ErrorType<
    | GetCatalogDefinition401
    | GetCatalogDefinition403
    | GetCatalogDefinition404
    | GetCatalogDefinition409
    | GetCatalogDefinition422
    | GetCatalogDefinition500
    | GetCatalogDefinition503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCatalogDefinition>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetCatalogDefinitionQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getCatalogDefinition>>> = ({ signal }) =>
    getCatalogDefinition(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCatalogDefinition>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetCatalogDefinitionSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getCatalogDefinition>>
>;
export type GetCatalogDefinitionSuspenseQueryError = ErrorType<
  | GetCatalogDefinition401
  | GetCatalogDefinition403
  | GetCatalogDefinition404
  | GetCatalogDefinition409
  | GetCatalogDefinition422
  | GetCatalogDefinition500
  | GetCatalogDefinition503
>;

export function useGetCatalogDefinitionSuspense<
  TData = Awaited<ReturnType<typeof getCatalogDefinition>>,
  TError = ErrorType<
    | GetCatalogDefinition401
    | GetCatalogDefinition403
    | GetCatalogDefinition404
    | GetCatalogDefinition409
    | GetCatalogDefinition422
    | GetCatalogDefinition500
    | GetCatalogDefinition503
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCatalogDefinition>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetCatalogDefinitionSuspense<
  TData = Awaited<ReturnType<typeof getCatalogDefinition>>,
  TError = ErrorType<
    | GetCatalogDefinition401
    | GetCatalogDefinition403
    | GetCatalogDefinition404
    | GetCatalogDefinition409
    | GetCatalogDefinition422
    | GetCatalogDefinition500
    | GetCatalogDefinition503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCatalogDefinition>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetCatalogDefinitionSuspense<
  TData = Awaited<ReturnType<typeof getCatalogDefinition>>,
  TError = ErrorType<
    | GetCatalogDefinition401
    | GetCatalogDefinition403
    | GetCatalogDefinition404
    | GetCatalogDefinition409
    | GetCatalogDefinition422
    | GetCatalogDefinition500
    | GetCatalogDefinition503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCatalogDefinition>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the fields of a definition's add form, from the custom definition with the id or else the catalog's
 */

export function useGetCatalogDefinitionSuspense<
  TData = Awaited<ReturnType<typeof getCatalogDefinition>>,
  TError = ErrorType<
    | GetCatalogDefinition401
    | GetCatalogDefinition403
    | GetCatalogDefinition404
    | GetCatalogDefinition409
    | GetCatalogDefinition422
    | GetCatalogDefinition500
    | GetCatalogDefinition503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCatalogDefinition>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetCatalogDefinitionSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetIndexerSettingsUrl = () => {
  return `/api/v1/indexer-settings`;
};

/**
 * @summary Get the Indexer settings
 */
export const getIndexerSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersSettings> => {
  return apiFetch<IndexersSettings>(getGetIndexerSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetIndexerSettingsQueryKey = () => {
  return [`/api/v1/indexer-settings`] as const;
};

export const getGetIndexerSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getIndexerSettings>>,
  TError = ErrorType<
    GetIndexerSettings401 | GetIndexerSettings403 | GetIndexerSettings409 | GetIndexerSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetIndexerSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getIndexerSettings>>> = ({ signal }) =>
    getIndexerSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetIndexerSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getIndexerSettings>>
>;
export type GetIndexerSettingsSuspenseQueryError = ErrorType<
  GetIndexerSettings401 | GetIndexerSettings403 | GetIndexerSettings409 | GetIndexerSettings500
>;

export function useGetIndexerSettingsSuspense<
  TData = Awaited<ReturnType<typeof getIndexerSettings>>,
  TError = ErrorType<
    GetIndexerSettings401 | GetIndexerSettings403 | GetIndexerSettings409 | GetIndexerSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetIndexerSettingsSuspense<
  TData = Awaited<ReturnType<typeof getIndexerSettings>>,
  TError = ErrorType<
    GetIndexerSettings401 | GetIndexerSettings403 | GetIndexerSettings409 | GetIndexerSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetIndexerSettingsSuspense<
  TData = Awaited<ReturnType<typeof getIndexerSettings>>,
  TError = ErrorType<
    GetIndexerSettings401 | GetIndexerSettings403 | GetIndexerSettings409 | GetIndexerSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Indexer settings
 */

export function useGetIndexerSettingsSuspense<
  TData = Awaited<ReturnType<typeof getIndexerSettings>>,
  TError = ErrorType<
    GetIndexerSettings401 | GetIndexerSettings403 | GetIndexerSettings409 | GetIndexerSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIndexerSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetIndexerSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateIndexerSettingsUrl = () => {
  return `/api/v1/indexer-settings`;
};

/**
 * @summary Change Indexer settings by key, all or none
 */
export const updateIndexerSettings = async (
  indexersSettingsChange: IndexersSettingsChange,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersSettings> => {
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
  return apiFetch<IndexersSettings>(getUpdateIndexerSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(indexersSettingsChange),
  });
};

export const getUpdateIndexerSettingsMutationKey = () => ["updateIndexerSettings"] as const;

export const getUpdateIndexerSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateIndexerSettings401
    | UpdateIndexerSettings403
    | UpdateIndexerSettings409
    | UpdateIndexerSettings422
    | UpdateIndexerSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateIndexerSettings>>,
    TError,
    UpdateIndexerSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateIndexerSettings>>,
  TError,
  UpdateIndexerSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateIndexerSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateIndexerSettings>>,
    UpdateIndexerSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateIndexerSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateIndexerSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateIndexerSettings>>
>;
export type UpdateIndexerSettingsMutationBody = IndexersSettingsChange;
export type UpdateIndexerSettingsMutationError = ErrorType<
  | UpdateIndexerSettings401
  | UpdateIndexerSettings403
  | UpdateIndexerSettings409
  | UpdateIndexerSettings422
  | UpdateIndexerSettings500
>;
export type UpdateIndexerSettingsMutationVariables = { data: IndexersSettingsChange };

/**
 * @summary Change Indexer settings by key, all or none
 */
export const useUpdateIndexerSettings = <
  TError = ErrorType<
    | UpdateIndexerSettings401
    | UpdateIndexerSettings403
    | UpdateIndexerSettings409
    | UpdateIndexerSettings422
    | UpdateIndexerSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateIndexerSettings>>,
      TError,
      UpdateIndexerSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateIndexerSettings>>,
  TError,
  UpdateIndexerSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateIndexerSettingsMutationOptions(options), queryClient);
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
    | CreateIndexer404
    | CreateIndexer409
    | CreateIndexer410
    | CreateIndexer422
    | CreateIndexer500
    | CreateIndexer502
    | CreateIndexer503
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
  | CreateIndexer404
  | CreateIndexer409
  | CreateIndexer410
  | CreateIndexer422
  | CreateIndexer500
  | CreateIndexer502
  | CreateIndexer503
>;
export type CreateIndexerMutationVariables = { data: IndexersNewIndexer };

/**
 * @summary Add an Indexer once its Test passed on the Node it searches from
 */
export const useCreateIndexer = <
  TError = ErrorType<
    | CreateIndexer401
    | CreateIndexer403
    | CreateIndexer404
    | CreateIndexer409
    | CreateIndexer410
    | CreateIndexer422
    | CreateIndexer500
    | CreateIndexer502
    | CreateIndexer503
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
export const getGetIndexerCaptchaDraftUrl = () => {
  return `/api/v1/indexers/captcha`;
};

/**
 * @summary Load the captcha of an unsaved Indexer's sign-in, from the Node it would search from
 */
export const getIndexerCaptchaDraft = async (
  indexersNewIndexer: IndexersNewIndexer,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersCaptchaImage> => {
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
  return apiFetch<IndexersCaptchaImage>(getGetIndexerCaptchaDraftUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(indexersNewIndexer),
  });
};

export const getGetIndexerCaptchaDraftMutationKey = () => ["getIndexerCaptchaDraft"] as const;

export const getGetIndexerCaptchaDraftMutationOptions = <
  TError = ErrorType<
    | GetIndexerCaptchaDraft401
    | GetIndexerCaptchaDraft403
    | GetIndexerCaptchaDraft404
    | GetIndexerCaptchaDraft409
    | GetIndexerCaptchaDraft410
    | GetIndexerCaptchaDraft422
    | GetIndexerCaptchaDraft500
    | GetIndexerCaptchaDraft502
    | GetIndexerCaptchaDraft503
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof getIndexerCaptchaDraft>>,
    TError,
    GetIndexerCaptchaDraftMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof getIndexerCaptchaDraft>>,
  TError,
  GetIndexerCaptchaDraftMutationVariables,
  TContext
> => {
  const mutationKey = getGetIndexerCaptchaDraftMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof getIndexerCaptchaDraft>>,
    GetIndexerCaptchaDraftMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return getIndexerCaptchaDraft(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type GetIndexerCaptchaDraftMutationResult = NonNullable<
  Awaited<ReturnType<typeof getIndexerCaptchaDraft>>
>;
export type GetIndexerCaptchaDraftMutationBody = IndexersNewIndexer;
export type GetIndexerCaptchaDraftMutationError = ErrorType<
  | GetIndexerCaptchaDraft401
  | GetIndexerCaptchaDraft403
  | GetIndexerCaptchaDraft404
  | GetIndexerCaptchaDraft409
  | GetIndexerCaptchaDraft410
  | GetIndexerCaptchaDraft422
  | GetIndexerCaptchaDraft500
  | GetIndexerCaptchaDraft502
  | GetIndexerCaptchaDraft503
>;
export type GetIndexerCaptchaDraftMutationVariables = { data: IndexersNewIndexer };

/**
 * @summary Load the captcha of an unsaved Indexer's sign-in, from the Node it would search from
 */
export const useGetIndexerCaptchaDraft = <
  TError = ErrorType<
    | GetIndexerCaptchaDraft401
    | GetIndexerCaptchaDraft403
    | GetIndexerCaptchaDraft404
    | GetIndexerCaptchaDraft409
    | GetIndexerCaptchaDraft410
    | GetIndexerCaptchaDraft422
    | GetIndexerCaptchaDraft500
    | GetIndexerCaptchaDraft502
    | GetIndexerCaptchaDraft503
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof getIndexerCaptchaDraft>>,
      TError,
      GetIndexerCaptchaDraftMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof getIndexerCaptchaDraft>>,
  TError,
  GetIndexerCaptchaDraftMutationVariables,
  TContext
> => {
  return useMutation(getGetIndexerCaptchaDraftMutationOptions(options), queryClient);
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
    | TestIndexerDraft404
    | TestIndexerDraft409
    | TestIndexerDraft410
    | TestIndexerDraft422
    | TestIndexerDraft500
    | TestIndexerDraft502
    | TestIndexerDraft503
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
  | TestIndexerDraft404
  | TestIndexerDraft409
  | TestIndexerDraft410
  | TestIndexerDraft422
  | TestIndexerDraft500
  | TestIndexerDraft502
  | TestIndexerDraft503
>;
export type TestIndexerDraftMutationVariables = { data: IndexersNewIndexer };

/**
 * @summary Test an Indexer without saving it
 */
export const useTestIndexerDraft = <
  TError = ErrorType<
    | TestIndexerDraft401
    | TestIndexerDraft403
    | TestIndexerDraft404
    | TestIndexerDraft409
    | TestIndexerDraft410
    | TestIndexerDraft422
    | TestIndexerDraft500
    | TestIndexerDraft502
    | TestIndexerDraft503
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
 * @summary Change an Indexer, testing it first when the change is to its home, address, API key, cookie or catalog settings
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
    | UpdateIndexer410
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
  | UpdateIndexer410
  | UpdateIndexer422
  | UpdateIndexer500
  | UpdateIndexer502
>;
export type UpdateIndexerMutationVariables = { id: string; data: IndexersIndexerEdit };

/**
 * @summary Change an Indexer, testing it first when the change is to its home, address, API key, cookie or catalog settings
 */
export const useUpdateIndexer = <
  TError = ErrorType<
    | UpdateIndexer401
    | UpdateIndexer403
    | UpdateIndexer404
    | UpdateIndexer409
    | UpdateIndexer410
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
export const getGetIndexerCaptchaUrl = (id: string) => {
  return `/api/v1/indexers/${id}/captcha`;
};

/**
 * @summary Load the captcha of an Indexer's sign-in, from the Node it searches from
 */
export const getIndexerCaptcha = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersCaptchaImage> => {
  return apiFetch<IndexersCaptchaImage>(getGetIndexerCaptchaUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getGetIndexerCaptchaMutationKey = () => ["getIndexerCaptcha"] as const;

export const getGetIndexerCaptchaMutationOptions = <
  TError = ErrorType<
    | GetIndexerCaptcha401
    | GetIndexerCaptcha403
    | GetIndexerCaptcha404
    | GetIndexerCaptcha409
    | GetIndexerCaptcha422
    | GetIndexerCaptcha500
    | GetIndexerCaptcha502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof getIndexerCaptcha>>,
    TError,
    GetIndexerCaptchaMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof getIndexerCaptcha>>,
  TError,
  GetIndexerCaptchaMutationVariables,
  TContext
> => {
  const mutationKey = getGetIndexerCaptchaMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof getIndexerCaptcha>>,
    GetIndexerCaptchaMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return getIndexerCaptcha(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type GetIndexerCaptchaMutationResult = NonNullable<
  Awaited<ReturnType<typeof getIndexerCaptcha>>
>;

export type GetIndexerCaptchaMutationError = ErrorType<
  | GetIndexerCaptcha401
  | GetIndexerCaptcha403
  | GetIndexerCaptcha404
  | GetIndexerCaptcha409
  | GetIndexerCaptcha422
  | GetIndexerCaptcha500
  | GetIndexerCaptcha502
>;
export type GetIndexerCaptchaMutationVariables = { id: string };

/**
 * @summary Load the captcha of an Indexer's sign-in, from the Node it searches from
 */
export const useGetIndexerCaptcha = <
  TError = ErrorType<
    | GetIndexerCaptcha401
    | GetIndexerCaptcha403
    | GetIndexerCaptcha404
    | GetIndexerCaptcha409
    | GetIndexerCaptcha422
    | GetIndexerCaptcha500
    | GetIndexerCaptcha502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof getIndexerCaptcha>>,
      TError,
      GetIndexerCaptchaMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof getIndexerCaptcha>>,
  TError,
  GetIndexerCaptchaMutationVariables,
  TContext
> => {
  return useMutation(getGetIndexerCaptchaMutationOptions(options), queryClient);
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
  indexersTestBody?: IndexersTestBody,
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
  return apiFetch<IndexersReport>(getTestIndexerUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(indexersTestBody),
  });
};

export const getTestIndexerMutationKey = () => ["testIndexer"] as const;

export const getTestIndexerMutationOptions = <
  TError = ErrorType<
    | TestIndexer401
    | TestIndexer403
    | TestIndexer404
    | TestIndexer409
    | TestIndexer410
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
    const { id, data } = props ?? {};

    return testIndexer(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type TestIndexerMutationResult = NonNullable<Awaited<ReturnType<typeof testIndexer>>>;
export type TestIndexerMutationBody = IndexersTestBody | undefined;
export type TestIndexerMutationError = ErrorType<
  | TestIndexer401
  | TestIndexer403
  | TestIndexer404
  | TestIndexer409
  | TestIndexer410
  | TestIndexer422
  | TestIndexer500
  | TestIndexer502
>;
export type TestIndexerMutationVariables = { id: string; data?: IndexersTestBody };

/**
 * @summary Test an Indexer from the Node it searches from. A pass clears its backoff and any stop.
 */
export const useTestIndexer = <
  TError = ErrorType<
    | TestIndexer401
    | TestIndexer403
    | TestIndexer404
    | TestIndexer409
    | TestIndexer410
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
export const getListTrackersUrl = () => {
  return `/api/v1/trackers`;
};

/**
 * @summary List the Trackers and the Indexers that reach each
 */
export const listTrackers = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersTrackerList> => {
  return apiFetch<IndexersTrackerList>(getListTrackersUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListTrackersQueryKey = () => {
  return [`/api/v1/trackers`] as const;
};

export const getListTrackersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listTrackers>>,
  TError = ErrorType<ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListTrackersQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listTrackers>>> = ({ signal }) =>
    listTrackers({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListTrackersSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listTrackers>>>;
export type ListTrackersSuspenseQueryError = ErrorType<
  ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500
>;

export function useListTrackersSuspense<
  TData = Awaited<ReturnType<typeof listTrackers>>,
  TError = ErrorType<ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTrackersSuspense<
  TData = Awaited<ReturnType<typeof listTrackers>>,
  TError = ErrorType<ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTrackersSuspense<
  TData = Awaited<ReturnType<typeof listTrackers>>,
  TError = ErrorType<ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Trackers and the Indexers that reach each
 */

export function useListTrackersSuspense<
  TData = Awaited<ReturnType<typeof listTrackers>>,
  TError = ErrorType<ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListTrackersSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetTrackerUrl = (id: string) => {
  return `/api/v1/trackers/${id}`;
};

/**
 * @summary Get a Tracker
 */
export const getTracker = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersTracker> => {
  return apiFetch<IndexersTracker>(getGetTrackerUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetTrackerQueryKey = (id: string) => {
  return [`/api/v1/trackers/${id}`] as const;
};

export const getGetTrackerSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getTracker>>,
  TError = ErrorType<
    GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetTrackerQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getTracker>>> = ({ signal }) =>
    getTracker(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetTrackerSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getTracker>>>;
export type GetTrackerSuspenseQueryError = ErrorType<
  GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
>;

export function useGetTrackerSuspense<
  TData = Awaited<ReturnType<typeof getTracker>>,
  TError = ErrorType<
    GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
  >,
>(
  id: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTrackerSuspense<
  TData = Awaited<ReturnType<typeof getTracker>>,
  TError = ErrorType<
    GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTrackerSuspense<
  TData = Awaited<ReturnType<typeof getTracker>>,
  TError = ErrorType<
    GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Tracker
 */

export function useGetTrackerSuspense<
  TData = Awaited<ReturnType<typeof getTracker>>,
  TError = ErrorType<
    GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetTrackerSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getRenameTrackerUrl = (id: string) => {
  return `/api/v1/trackers/${id}`;
};

/**
 * @summary Rename a Tracker. Every Indexer and Release that reached it still does.
 */
export const renameTracker = async (
  id: string,
  renameTrackerInputBody: RenameTrackerInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersTracker> => {
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
  return apiFetch<IndexersTracker>(getRenameTrackerUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(renameTrackerInputBody),
  });
};

export const getRenameTrackerMutationKey = () => ["renameTracker"] as const;

export const getRenameTrackerMutationOptions = <
  TError = ErrorType<
    | RenameTracker401
    | RenameTracker403
    | RenameTracker404
    | RenameTracker409
    | RenameTracker422
    | RenameTracker500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof renameTracker>>,
    TError,
    RenameTrackerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof renameTracker>>,
  TError,
  RenameTrackerMutationVariables,
  TContext
> => {
  const mutationKey = getRenameTrackerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof renameTracker>>,
    RenameTrackerMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return renameTracker(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RenameTrackerMutationResult = NonNullable<Awaited<ReturnType<typeof renameTracker>>>;
export type RenameTrackerMutationBody = RenameTrackerInputBody;
export type RenameTrackerMutationError = ErrorType<
  | RenameTracker401
  | RenameTracker403
  | RenameTracker404
  | RenameTracker409
  | RenameTracker422
  | RenameTracker500
>;
export type RenameTrackerMutationVariables = { id: string; data: RenameTrackerInputBody };

/**
 * @summary Rename a Tracker. Every Indexer and Release that reached it still does.
 */
export const useRenameTracker = <
  TError = ErrorType<
    | RenameTracker401
    | RenameTracker403
    | RenameTracker404
    | RenameTracker409
    | RenameTracker422
    | RenameTracker500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof renameTracker>>,
      TError,
      RenameTrackerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof renameTracker>>,
  TError,
  RenameTrackerMutationVariables,
  TContext
> => {
  return useMutation(getRenameTrackerMutationOptions(options), queryClient);
};
export const getMergeTrackerUrl = (id: string) => {
  return `/api/v1/trackers/${id}/merge`;
};

/**
 * @summary Merge a Tracker into another, which keeps its name and takes its Indexers, Grabs, Torrents and hosts. The merged Tracker is deleted.
 */
export const mergeTracker = async (
  id: string,
  mergeTrackerInputBody: MergeTrackerInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IndexersTracker> => {
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
  return apiFetch<IndexersTracker>(getMergeTrackerUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(mergeTrackerInputBody),
  });
};

export const getMergeTrackerMutationKey = () => ["mergeTracker"] as const;

export const getMergeTrackerMutationOptions = <
  TError = ErrorType<
    | MergeTracker401
    | MergeTracker403
    | MergeTracker404
    | MergeTracker409
    | MergeTracker422
    | MergeTracker500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof mergeTracker>>,
    TError,
    MergeTrackerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof mergeTracker>>,
  TError,
  MergeTrackerMutationVariables,
  TContext
> => {
  const mutationKey = getMergeTrackerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof mergeTracker>>,
    MergeTrackerMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return mergeTracker(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MergeTrackerMutationResult = NonNullable<Awaited<ReturnType<typeof mergeTracker>>>;
export type MergeTrackerMutationBody = MergeTrackerInputBody;
export type MergeTrackerMutationError = ErrorType<
  | MergeTracker401
  | MergeTracker403
  | MergeTracker404
  | MergeTracker409
  | MergeTracker422
  | MergeTracker500
>;
export type MergeTrackerMutationVariables = { id: string; data: MergeTrackerInputBody };

/**
 * @summary Merge a Tracker into another, which keeps its name and takes its Indexers, Grabs, Torrents and hosts. The merged Tracker is deleted.
 */
export const useMergeTracker = <
  TError = ErrorType<
    | MergeTracker401
    | MergeTracker403
    | MergeTracker404
    | MergeTracker409
    | MergeTracker422
    | MergeTracker500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof mergeTracker>>,
      TError,
      MergeTrackerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof mergeTracker>>,
  TError,
  MergeTrackerMutationVariables,
  TContext
> => {
  return useMutation(getMergeTrackerMutationOptions(options), queryClient);
};
