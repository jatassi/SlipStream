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
  CreateRoot401,
  CreateRoot403,
  CreateRoot404,
  CreateRoot409,
  CreateRoot422,
  CreateRoot500,
  ListRoots401,
  ListRoots403,
  ListRoots409,
  ListRoots500,
  MakeDefaultRoot401,
  MakeDefaultRoot403,
  MakeDefaultRoot404,
  MakeDefaultRoot409,
  MakeDefaultRoot422,
  MakeDefaultRoot500,
  RootsNewRoot,
  RootsRoot,
  RootsRootList,
  RootsWorkRoot,
  SetWorkRoot401,
  SetWorkRoot403,
  SetWorkRoot404,
  SetWorkRoot409,
  SetWorkRoot422,
  SetWorkRoot500,
  SetWorkRootInputBody,
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
 * @summary Make a folder on the Controller a library root of a Content Type
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
 * @summary Make a folder on the Controller a library root of a Content Type
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
