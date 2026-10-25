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
  BrowseFolders401,
  BrowseFolders403,
  BrowseFolders404,
  BrowseFolders409,
  BrowseFolders422,
  BrowseFolders500,
  BrowseFolders503,
  BrowseFoldersParams,
  CheckFolder401,
  CheckFolder403,
  CheckFolder404,
  CheckFolder409,
  CheckFolder422,
  CheckFolder500,
  CheckFolderInputBody,
  CreateRoot401,
  CreateRoot403,
  CreateRoot404,
  CreateRoot409,
  CreateRoot422,
  CreateRoot500,
  DeleteRoot401,
  DeleteRoot403,
  DeleteRoot404,
  DeleteRoot409,
  DeleteRoot422,
  DeleteRoot500,
  DeleteRootParams,
  GetRootSettings401,
  GetRootSettings403,
  GetRootSettings409,
  GetRootSettings500,
  ListRoots401,
  ListRoots403,
  ListRoots409,
  ListRoots500,
  ListVolumes401,
  ListVolumes403,
  ListVolumes409,
  ListVolumes500,
  MakeDefaultRoot401,
  MakeDefaultRoot403,
  MakeDefaultRoot404,
  MakeDefaultRoot409,
  MakeDefaultRoot422,
  MakeDefaultRoot500,
  MakeFolder401,
  MakeFolder403,
  MakeFolder404,
  MakeFolder409,
  MakeFolder422,
  MakeFolder500,
  MakeFolder503,
  MakeFolderInputBody,
  RehomeRoot401,
  RehomeRoot403,
  RehomeRoot404,
  RehomeRoot409,
  RehomeRoot422,
  RehomeRoot500,
  RehomeRoot503,
  RehomeRootInputBody,
  RootSettingsInputBody,
  RootsListing,
  RootsMadeFolder,
  RootsNewRoot,
  RootsRoot,
  RootsRootList,
  RootsRootSettings,
  RootsSuggestions,
  RootsVerdict,
  RootsVolumeList,
  RootsWorkRoot,
  SetWorkRoot401,
  SetWorkRoot403,
  SetWorkRoot404,
  SetWorkRoot409,
  SetWorkRoot422,
  SetWorkRoot500,
  SetWorkRootInputBody,
  SuggestFolders401,
  SuggestFolders403,
  SuggestFolders404,
  SuggestFolders409,
  SuggestFolders422,
  SuggestFolders500,
  SuggestFolders503,
  SuggestFoldersParams,
  UpdateRootSettings401,
  UpdateRootSettings403,
  UpdateRootSettings409,
  UpdateRootSettings422,
  UpdateRootSettings500,
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

export const getBrowseFoldersUrl = (id: string, params?: BrowseFoldersParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/nodes/${id}/folders?${stringifiedParams}`
    : `/api/v1/nodes/${id}/folders`;
};

/**
 * @summary List a folder on a Node, or its places
 */
export const browseFolders = async (
  id: string,
  params?: BrowseFoldersParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsListing> => {
  return apiFetch<RootsListing>(getBrowseFoldersUrl(id, params), {
    ...options,
    method: "GET",
  });
};

export const getBrowseFoldersQueryKey = (id: string, params?: BrowseFoldersParams) => {
  return [`/api/v1/nodes/${id}/folders`, ...(params ? [params] : [])] as const;
};

export const getBrowseFoldersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof browseFolders>>,
  TError = ErrorType<
    | BrowseFolders401
    | BrowseFolders403
    | BrowseFolders404
    | BrowseFolders409
    | BrowseFolders422
    | BrowseFolders500
    | BrowseFolders503
  >,
>(
  id: string,
  params?: BrowseFoldersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof browseFolders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getBrowseFoldersQueryKey(id, params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof browseFolders>>> = ({ signal }) =>
    browseFolders(id, params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof browseFolders>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type BrowseFoldersSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof browseFolders>>
>;
export type BrowseFoldersSuspenseQueryError = ErrorType<
  | BrowseFolders401
  | BrowseFolders403
  | BrowseFolders404
  | BrowseFolders409
  | BrowseFolders422
  | BrowseFolders500
  | BrowseFolders503
>;

export function useBrowseFoldersSuspense<
  TData = Awaited<ReturnType<typeof browseFolders>>,
  TError = ErrorType<
    | BrowseFolders401
    | BrowseFolders403
    | BrowseFolders404
    | BrowseFolders409
    | BrowseFolders422
    | BrowseFolders500
    | BrowseFolders503
  >,
>(
  id: string,
  params: undefined | BrowseFoldersParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof browseFolders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useBrowseFoldersSuspense<
  TData = Awaited<ReturnType<typeof browseFolders>>,
  TError = ErrorType<
    | BrowseFolders401
    | BrowseFolders403
    | BrowseFolders404
    | BrowseFolders409
    | BrowseFolders422
    | BrowseFolders500
    | BrowseFolders503
  >,
>(
  id: string,
  params?: BrowseFoldersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof browseFolders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useBrowseFoldersSuspense<
  TData = Awaited<ReturnType<typeof browseFolders>>,
  TError = ErrorType<
    | BrowseFolders401
    | BrowseFolders403
    | BrowseFolders404
    | BrowseFolders409
    | BrowseFolders422
    | BrowseFolders500
    | BrowseFolders503
  >,
>(
  id: string,
  params?: BrowseFoldersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof browseFolders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List a folder on a Node, or its places
 */

export function useBrowseFoldersSuspense<
  TData = Awaited<ReturnType<typeof browseFolders>>,
  TError = ErrorType<
    | BrowseFolders401
    | BrowseFolders403
    | BrowseFolders404
    | BrowseFolders409
    | BrowseFolders422
    | BrowseFolders500
    | BrowseFolders503
  >,
>(
  id: string,
  params?: BrowseFoldersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof browseFolders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getBrowseFoldersSuspenseQueryOptions(id, params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getMakeFolderUrl = (id: string) => {
  return `/api/v1/nodes/${id}/folders`;
};

/**
 * @summary Make a folder in an existing folder on a Node
 */
export const makeFolder = async (
  id: string,
  makeFolderInputBody: MakeFolderInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsMadeFolder> => {
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
  return apiFetch<RootsMadeFolder>(getMakeFolderUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(makeFolderInputBody),
  });
};

export const getMakeFolderMutationKey = () => ["makeFolder"] as const;

export const getMakeFolderMutationOptions = <
  TError = ErrorType<
    | MakeFolder401
    | MakeFolder403
    | MakeFolder404
    | MakeFolder409
    | MakeFolder422
    | MakeFolder500
    | MakeFolder503
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof makeFolder>>,
    TError,
    MakeFolderMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof makeFolder>>,
  TError,
  MakeFolderMutationVariables,
  TContext
> => {
  const mutationKey = getMakeFolderMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof makeFolder>>,
    MakeFolderMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return makeFolder(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MakeFolderMutationResult = NonNullable<Awaited<ReturnType<typeof makeFolder>>>;
export type MakeFolderMutationBody = MakeFolderInputBody;
export type MakeFolderMutationError = ErrorType<
  | MakeFolder401
  | MakeFolder403
  | MakeFolder404
  | MakeFolder409
  | MakeFolder422
  | MakeFolder500
  | MakeFolder503
>;
export type MakeFolderMutationVariables = { id: string; data: MakeFolderInputBody };

/**
 * @summary Make a folder in an existing folder on a Node
 */
export const useMakeFolder = <
  TError = ErrorType<
    | MakeFolder401
    | MakeFolder403
    | MakeFolder404
    | MakeFolder409
    | MakeFolder422
    | MakeFolder500
    | MakeFolder503
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof makeFolder>>,
      TError,
      MakeFolderMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof makeFolder>>,
  TError,
  MakeFolderMutationVariables,
  TContext
> => {
  return useMutation(getMakeFolderMutationOptions(options), queryClient);
};
export const getCheckFolderUrl = (id: string) => {
  return `/api/v1/nodes/${id}/folders/check`;
};

/**
 * @summary Judge a folder on a Node for a purpose without saving it
 */
export const checkFolder = async (
  id: string,
  checkFolderInputBody: CheckFolderInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsVerdict> => {
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
  return apiFetch<RootsVerdict>(getCheckFolderUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(checkFolderInputBody),
  });
};

export const getCheckFolderMutationKey = () => ["checkFolder"] as const;

export const getCheckFolderMutationOptions = <
  TError = ErrorType<
    | CheckFolder401
    | CheckFolder403
    | CheckFolder404
    | CheckFolder409
    | CheckFolder422
    | CheckFolder500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof checkFolder>>,
    TError,
    CheckFolderMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof checkFolder>>,
  TError,
  CheckFolderMutationVariables,
  TContext
> => {
  const mutationKey = getCheckFolderMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof checkFolder>>,
    CheckFolderMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return checkFolder(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CheckFolderMutationResult = NonNullable<Awaited<ReturnType<typeof checkFolder>>>;
export type CheckFolderMutationBody = CheckFolderInputBody;
export type CheckFolderMutationError = ErrorType<
  | CheckFolder401
  | CheckFolder403
  | CheckFolder404
  | CheckFolder409
  | CheckFolder422
  | CheckFolder500
>;
export type CheckFolderMutationVariables = { id: string; data: CheckFolderInputBody };

/**
 * @summary Judge a folder on a Node for a purpose without saving it
 */
export const useCheckFolder = <
  TError = ErrorType<
    | CheckFolder401
    | CheckFolder403
    | CheckFolder404
    | CheckFolder409
    | CheckFolder422
    | CheckFolder500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof checkFolder>>,
      TError,
      CheckFolderMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof checkFolder>>,
  TError,
  CheckFolderMutationVariables,
  TContext
> => {
  return useMutation(getCheckFolderMutationOptions(options), queryClient);
};
export const getSuggestFoldersUrl = (id: string, params?: SuggestFoldersParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/nodes/${id}/folders/suggest?${stringifiedParams}`
    : `/api/v1/nodes/${id}/folders/suggest`;
};

/**
 * @summary List the folders on a Node that complete a typed path
 */
export const suggestFolders = async (
  id: string,
  params?: SuggestFoldersParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsSuggestions> => {
  return apiFetch<RootsSuggestions>(getSuggestFoldersUrl(id, params), {
    ...options,
    method: "GET",
  });
};

export const getSuggestFoldersQueryKey = (id: string, params?: SuggestFoldersParams) => {
  return [`/api/v1/nodes/${id}/folders/suggest`, ...(params ? [params] : [])] as const;
};

export const getSuggestFoldersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof suggestFolders>>,
  TError = ErrorType<
    | SuggestFolders401
    | SuggestFolders403
    | SuggestFolders404
    | SuggestFolders409
    | SuggestFolders422
    | SuggestFolders500
    | SuggestFolders503
  >,
>(
  id: string,
  params?: SuggestFoldersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof suggestFolders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getSuggestFoldersQueryKey(id, params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof suggestFolders>>> = ({ signal }) =>
    suggestFolders(id, params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof suggestFolders>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type SuggestFoldersSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof suggestFolders>>
>;
export type SuggestFoldersSuspenseQueryError = ErrorType<
  | SuggestFolders401
  | SuggestFolders403
  | SuggestFolders404
  | SuggestFolders409
  | SuggestFolders422
  | SuggestFolders500
  | SuggestFolders503
>;

export function useSuggestFoldersSuspense<
  TData = Awaited<ReturnType<typeof suggestFolders>>,
  TError = ErrorType<
    | SuggestFolders401
    | SuggestFolders403
    | SuggestFolders404
    | SuggestFolders409
    | SuggestFolders422
    | SuggestFolders500
    | SuggestFolders503
  >,
>(
  id: string,
  params: undefined | SuggestFoldersParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof suggestFolders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useSuggestFoldersSuspense<
  TData = Awaited<ReturnType<typeof suggestFolders>>,
  TError = ErrorType<
    | SuggestFolders401
    | SuggestFolders403
    | SuggestFolders404
    | SuggestFolders409
    | SuggestFolders422
    | SuggestFolders500
    | SuggestFolders503
  >,
>(
  id: string,
  params?: SuggestFoldersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof suggestFolders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useSuggestFoldersSuspense<
  TData = Awaited<ReturnType<typeof suggestFolders>>,
  TError = ErrorType<
    | SuggestFolders401
    | SuggestFolders403
    | SuggestFolders404
    | SuggestFolders409
    | SuggestFolders422
    | SuggestFolders500
    | SuggestFolders503
  >,
>(
  id: string,
  params?: SuggestFoldersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof suggestFolders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the folders on a Node that complete a typed path
 */

export function useSuggestFoldersSuspense<
  TData = Awaited<ReturnType<typeof suggestFolders>>,
  TError = ErrorType<
    | SuggestFolders401
    | SuggestFolders403
    | SuggestFolders404
    | SuggestFolders409
    | SuggestFolders422
    | SuggestFolders500
    | SuggestFolders503
  >,
>(
  id: string,
  params?: SuggestFoldersParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof suggestFolders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getSuggestFoldersSuspenseQueryOptions(id, params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getListRootsUrl = () => {
  return `/api/v1/roots`;
};

/**
 * @summary List the library roots
 */
export const listRoots = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsRootList> => {
  return apiFetch<RootsRootList>(getListRootsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListRootsQueryKey = () => {
  return [`/api/v1/roots`] as const;
};

export const getListRootsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listRoots>>,
  TError = ErrorType<ListRoots401 | ListRoots403 | ListRoots409 | ListRoots500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listRoots>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListRootsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listRoots>>> = ({ signal }) =>
    listRoots({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listRoots>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListRootsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listRoots>>>;
export type ListRootsSuspenseQueryError = ErrorType<
  ListRoots401 | ListRoots403 | ListRoots409 | ListRoots500
>;

export function useListRootsSuspense<
  TData = Awaited<ReturnType<typeof listRoots>>,
  TError = ErrorType<ListRoots401 | ListRoots403 | ListRoots409 | ListRoots500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listRoots>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListRootsSuspense<
  TData = Awaited<ReturnType<typeof listRoots>>,
  TError = ErrorType<ListRoots401 | ListRoots403 | ListRoots409 | ListRoots500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listRoots>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListRootsSuspense<
  TData = Awaited<ReturnType<typeof listRoots>>,
  TError = ErrorType<ListRoots401 | ListRoots403 | ListRoots409 | ListRoots500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listRoots>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the library roots
 */

export function useListRootsSuspense<
  TData = Awaited<ReturnType<typeof listRoots>>,
  TError = ErrorType<ListRoots401 | ListRoots403 | ListRoots409 | ListRoots500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listRoots>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListRootsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCreateRootUrl = () => {
  return `/api/v1/roots`;
};

/**
 * @summary Make a folder on a Storage Node a library root of a Content Type
 */
export const createRoot = async (
  rootsNewRoot: RootsNewRoot,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsRoot> => {
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
  return apiFetch<RootsRoot>(getCreateRootUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(rootsNewRoot),
  });
};

export const getCreateRootMutationKey = () => ["createRoot"] as const;

export const getCreateRootMutationOptions = <
  TError = ErrorType<
    CreateRoot401 | CreateRoot403 | CreateRoot404 | CreateRoot409 | CreateRoot422 | CreateRoot500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof createRoot>>,
    TError,
    CreateRootMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof createRoot>>,
  TError,
  CreateRootMutationVariables,
  TContext
> => {
  const mutationKey = getCreateRootMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof createRoot>>,
    CreateRootMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return createRoot(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateRootMutationResult = NonNullable<Awaited<ReturnType<typeof createRoot>>>;
export type CreateRootMutationBody = RootsNewRoot;
export type CreateRootMutationError = ErrorType<
  CreateRoot401 | CreateRoot403 | CreateRoot404 | CreateRoot409 | CreateRoot422 | CreateRoot500
>;
export type CreateRootMutationVariables = { data: RootsNewRoot };

/**
 * @summary Make a folder on a Storage Node a library root of a Content Type
 */
export const useCreateRoot = <
  TError = ErrorType<
    CreateRoot401 | CreateRoot403 | CreateRoot404 | CreateRoot409 | CreateRoot422 | CreateRoot500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createRoot>>,
      TError,
      CreateRootMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof createRoot>>,
  TError,
  CreateRootMutationVariables,
  TContext
> => {
  return useMutation(getCreateRootMutationOptions(options), queryClient);
};
export const getGetRootSettingsUrl = () => {
  return `/api/v1/roots/settings`;
};

/**
 * @summary Get the library roots settings
 */
export const getRootSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsRootSettings> => {
  return apiFetch<RootsRootSettings>(getGetRootSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetRootSettingsQueryKey = () => {
  return [`/api/v1/roots/settings`] as const;
};

export const getGetRootSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getRootSettings>>,
  TError = ErrorType<
    GetRootSettings401 | GetRootSettings403 | GetRootSettings409 | GetRootSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRootSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetRootSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getRootSettings>>> = ({ signal }) =>
    getRootSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRootSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetRootSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getRootSettings>>
>;
export type GetRootSettingsSuspenseQueryError = ErrorType<
  GetRootSettings401 | GetRootSettings403 | GetRootSettings409 | GetRootSettings500
>;

export function useGetRootSettingsSuspense<
  TData = Awaited<ReturnType<typeof getRootSettings>>,
  TError = ErrorType<
    GetRootSettings401 | GetRootSettings403 | GetRootSettings409 | GetRootSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRootSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetRootSettingsSuspense<
  TData = Awaited<ReturnType<typeof getRootSettings>>,
  TError = ErrorType<
    GetRootSettings401 | GetRootSettings403 | GetRootSettings409 | GetRootSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRootSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetRootSettingsSuspense<
  TData = Awaited<ReturnType<typeof getRootSettings>>,
  TError = ErrorType<
    GetRootSettings401 | GetRootSettings403 | GetRootSettings409 | GetRootSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRootSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the library roots settings
 */

export function useGetRootSettingsSuspense<
  TData = Awaited<ReturnType<typeof getRootSettings>>,
  TError = ErrorType<
    GetRootSettings401 | GetRootSettings403 | GetRootSettings409 | GetRootSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRootSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetRootSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateRootSettingsUrl = () => {
  return `/api/v1/roots/settings`;
};

/**
 * @summary Change the library roots settings
 */
export const updateRootSettings = async (
  rootSettingsInputBody: RootSettingsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsRootSettings> => {
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
  return apiFetch<RootsRootSettings>(getUpdateRootSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(rootSettingsInputBody),
  });
};

export const getUpdateRootSettingsMutationKey = () => ["updateRootSettings"] as const;

export const getUpdateRootSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateRootSettings401
    | UpdateRootSettings403
    | UpdateRootSettings409
    | UpdateRootSettings422
    | UpdateRootSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateRootSettings>>,
    TError,
    UpdateRootSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateRootSettings>>,
  TError,
  UpdateRootSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateRootSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateRootSettings>>,
    UpdateRootSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateRootSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateRootSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateRootSettings>>
>;
export type UpdateRootSettingsMutationBody = RootSettingsInputBody;
export type UpdateRootSettingsMutationError = ErrorType<
  | UpdateRootSettings401
  | UpdateRootSettings403
  | UpdateRootSettings409
  | UpdateRootSettings422
  | UpdateRootSettings500
>;
export type UpdateRootSettingsMutationVariables = { data: RootSettingsInputBody };

/**
 * @summary Change the library roots settings
 */
export const useUpdateRootSettings = <
  TError = ErrorType<
    | UpdateRootSettings401
    | UpdateRootSettings403
    | UpdateRootSettings409
    | UpdateRootSettings422
    | UpdateRootSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateRootSettings>>,
      TError,
      UpdateRootSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateRootSettings>>,
  TError,
  UpdateRootSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateRootSettingsMutationOptions(options), queryClient);
};
export const getDeleteRootUrl = (rootId: string, params?: DeleteRootParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/roots/${rootId}?${stringifiedParams}`
    : `/api/v1/roots/${rootId}`;
};

/**
 * @summary Delete a root that holds no files, leaving its folder on disk
 */
export const deleteRoot = async (
  rootId: string,
  params?: DeleteRootParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeleteRootUrl(rootId, params), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteRootMutationKey = () => ["deleteRoot"] as const;

export const getDeleteRootMutationOptions = <
  TError = ErrorType<
    DeleteRoot401 | DeleteRoot403 | DeleteRoot404 | DeleteRoot409 | DeleteRoot422 | DeleteRoot500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteRoot>>,
    TError,
    DeleteRootMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteRoot>>,
  TError,
  DeleteRootMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteRootMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteRoot>>,
    DeleteRootMutationVariables
  > = (props) => {
    const { rootId, params } = props ?? {};

    return deleteRoot(rootId, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteRootMutationResult = NonNullable<Awaited<ReturnType<typeof deleteRoot>>>;

export type DeleteRootMutationError = ErrorType<
  DeleteRoot401 | DeleteRoot403 | DeleteRoot404 | DeleteRoot409 | DeleteRoot422 | DeleteRoot500
>;
export type DeleteRootMutationVariables = { rootId: string; params?: DeleteRootParams };

/**
 * @summary Delete a root that holds no files, leaving its folder on disk
 */
export const useDeleteRoot = <
  TError = ErrorType<
    DeleteRoot401 | DeleteRoot403 | DeleteRoot404 | DeleteRoot409 | DeleteRoot422 | DeleteRoot500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteRoot>>,
      TError,
      DeleteRootMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteRoot>>,
  TError,
  DeleteRootMutationVariables,
  TContext
> => {
  return useMutation(getDeleteRootMutationOptions(options), queryClient);
};
export const getRehomeRootUrl = (rootId: string) => {
  return `/api/v1/roots/${rootId}`;
};

/**
 * @summary Re-home a root onto another folder or Node once every file it records is there
 */
export const rehomeRoot = async (
  rootId: string,
  rehomeRootInputBody: RehomeRootInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsRoot> => {
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
  return apiFetch<RootsRoot>(getRehomeRootUrl(rootId), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(rehomeRootInputBody),
  });
};

export const getRehomeRootMutationKey = () => ["rehomeRoot"] as const;

export const getRehomeRootMutationOptions = <
  TError = ErrorType<
    | RehomeRoot401
    | RehomeRoot403
    | RehomeRoot404
    | RehomeRoot409
    | RehomeRoot422
    | RehomeRoot500
    | RehomeRoot503
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof rehomeRoot>>,
    TError,
    RehomeRootMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof rehomeRoot>>,
  TError,
  RehomeRootMutationVariables,
  TContext
> => {
  const mutationKey = getRehomeRootMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof rehomeRoot>>,
    RehomeRootMutationVariables
  > = (props) => {
    const { rootId, data } = props ?? {};

    return rehomeRoot(rootId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RehomeRootMutationResult = NonNullable<Awaited<ReturnType<typeof rehomeRoot>>>;
export type RehomeRootMutationBody = RehomeRootInputBody;
export type RehomeRootMutationError = ErrorType<
  | RehomeRoot401
  | RehomeRoot403
  | RehomeRoot404
  | RehomeRoot409
  | RehomeRoot422
  | RehomeRoot500
  | RehomeRoot503
>;
export type RehomeRootMutationVariables = { rootId: string; data: RehomeRootInputBody };

/**
 * @summary Re-home a root onto another folder or Node once every file it records is there
 */
export const useRehomeRoot = <
  TError = ErrorType<
    | RehomeRoot401
    | RehomeRoot403
    | RehomeRoot404
    | RehomeRoot409
    | RehomeRoot422
    | RehomeRoot500
    | RehomeRoot503
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof rehomeRoot>>,
      TError,
      RehomeRootMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof rehomeRoot>>,
  TError,
  RehomeRootMutationVariables,
  TContext
> => {
  return useMutation(getRehomeRootMutationOptions(options), queryClient);
};
export const getMakeDefaultRootUrl = (rootId: string) => {
  return `/api/v1/roots/${rootId}/default`;
};

/**
 * @summary Make a root where its Content Type's new Works go
 */
export const makeDefaultRoot = async (
  rootId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getMakeDefaultRootUrl(rootId), {
    ...options,
    method: "POST",
  });
};

export const getMakeDefaultRootMutationKey = () => ["makeDefaultRoot"] as const;

export const getMakeDefaultRootMutationOptions = <
  TError = ErrorType<
    | MakeDefaultRoot401
    | MakeDefaultRoot403
    | MakeDefaultRoot404
    | MakeDefaultRoot409
    | MakeDefaultRoot422
    | MakeDefaultRoot500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof makeDefaultRoot>>,
    TError,
    MakeDefaultRootMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof makeDefaultRoot>>,
  TError,
  MakeDefaultRootMutationVariables,
  TContext
> => {
  const mutationKey = getMakeDefaultRootMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof makeDefaultRoot>>,
    MakeDefaultRootMutationVariables
  > = (props) => {
    const { rootId } = props ?? {};

    return makeDefaultRoot(rootId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MakeDefaultRootMutationResult = NonNullable<
  Awaited<ReturnType<typeof makeDefaultRoot>>
>;

export type MakeDefaultRootMutationError = ErrorType<
  | MakeDefaultRoot401
  | MakeDefaultRoot403
  | MakeDefaultRoot404
  | MakeDefaultRoot409
  | MakeDefaultRoot422
  | MakeDefaultRoot500
>;
export type MakeDefaultRootMutationVariables = { rootId: string };

/**
 * @summary Make a root where its Content Type's new Works go
 */
export const useMakeDefaultRoot = <
  TError = ErrorType<
    | MakeDefaultRoot401
    | MakeDefaultRoot403
    | MakeDefaultRoot404
    | MakeDefaultRoot409
    | MakeDefaultRoot422
    | MakeDefaultRoot500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof makeDefaultRoot>>,
      TError,
      MakeDefaultRootMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof makeDefaultRoot>>,
  TError,
  MakeDefaultRootMutationVariables,
  TContext
> => {
  return useMutation(getMakeDefaultRootMutationOptions(options), queryClient);
};
export const getListVolumesUrl = () => {
  return `/api/v1/volumes`;
};

/**
 * @summary List each Storage Node's volumes that hold a library root
 */
export const listVolumes = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsVolumeList> => {
  return apiFetch<RootsVolumeList>(getListVolumesUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListVolumesQueryKey = () => {
  return [`/api/v1/volumes`] as const;
};

export const getListVolumesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listVolumes>>,
  TError = ErrorType<ListVolumes401 | ListVolumes403 | ListVolumes409 | ListVolumes500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listVolumes>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListVolumesQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listVolumes>>> = ({ signal }) =>
    listVolumes({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listVolumes>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListVolumesSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listVolumes>>>;
export type ListVolumesSuspenseQueryError = ErrorType<
  ListVolumes401 | ListVolumes403 | ListVolumes409 | ListVolumes500
>;

export function useListVolumesSuspense<
  TData = Awaited<ReturnType<typeof listVolumes>>,
  TError = ErrorType<ListVolumes401 | ListVolumes403 | ListVolumes409 | ListVolumes500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listVolumes>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListVolumesSuspense<
  TData = Awaited<ReturnType<typeof listVolumes>>,
  TError = ErrorType<ListVolumes401 | ListVolumes403 | ListVolumes409 | ListVolumes500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listVolumes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListVolumesSuspense<
  TData = Awaited<ReturnType<typeof listVolumes>>,
  TError = ErrorType<ListVolumes401 | ListVolumes403 | ListVolumes409 | ListVolumes500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listVolumes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List each Storage Node's volumes that hold a library root
 */

export function useListVolumesSuspense<
  TData = Awaited<ReturnType<typeof listVolumes>>,
  TError = ErrorType<ListVolumes401 | ListVolumes403 | ListVolumes409 | ListVolumes500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listVolumes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListVolumesSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetWorkRootUrl = (workId: string) => {
  return `/api/v1/works/${workId}`;
};

/**
 * @summary Move a Work that has no files into another root of its Content Type
 */
export const setWorkRoot = async (
  workId: string,
  setWorkRootInputBody: SetWorkRootInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RootsWorkRoot> => {
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
  return apiFetch<RootsWorkRoot>(getSetWorkRootUrl(workId), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(setWorkRootInputBody),
  });
};

export const getSetWorkRootMutationKey = () => ["setWorkRoot"] as const;

export const getSetWorkRootMutationOptions = <
  TError = ErrorType<
    | SetWorkRoot401
    | SetWorkRoot403
    | SetWorkRoot404
    | SetWorkRoot409
    | SetWorkRoot422
    | SetWorkRoot500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setWorkRoot>>,
    TError,
    SetWorkRootMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setWorkRoot>>,
  TError,
  SetWorkRootMutationVariables,
  TContext
> => {
  const mutationKey = getSetWorkRootMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setWorkRoot>>,
    SetWorkRootMutationVariables
  > = (props) => {
    const { workId, data } = props ?? {};

    return setWorkRoot(workId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetWorkRootMutationResult = NonNullable<Awaited<ReturnType<typeof setWorkRoot>>>;
export type SetWorkRootMutationBody = SetWorkRootInputBody;
export type SetWorkRootMutationError = ErrorType<
  | SetWorkRoot401
  | SetWorkRoot403
  | SetWorkRoot404
  | SetWorkRoot409
  | SetWorkRoot422
  | SetWorkRoot500
>;
export type SetWorkRootMutationVariables = { workId: string; data: SetWorkRootInputBody };

/**
 * @summary Move a Work that has no files into another root of its Content Type
 */
export const useSetWorkRoot = <
  TError = ErrorType<
    | SetWorkRoot401
    | SetWorkRoot403
    | SetWorkRoot404
    | SetWorkRoot409
    | SetWorkRoot422
    | SetWorkRoot500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setWorkRoot>>,
      TError,
      SetWorkRootMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setWorkRoot>>,
  TError,
  SetWorkRootMutationVariables,
  TContext
> => {
  return useMutation(getSetWorkRootMutationOptions(options), queryClient);
};
