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
  ListNaming401,
  ListNaming403,
  ListNaming409,
  ListNaming500,
  ListNamingTokens401,
  ListNamingTokens403,
  ListNamingTokens404,
  ListNamingTokens409,
  ListNamingTokens422,
  ListNamingTokens500,
  NamingNamingList,
  NamingPreview,
  NamingPreviewInputBody,
  NamingTokenList,
  NamingTypeNaming,
  PreviewNaming401,
  PreviewNaming403,
  PreviewNaming404,
  PreviewNaming409,
  PreviewNaming422,
  PreviewNaming500,
  SetNaming401,
  SetNaming403,
  SetNaming404,
  SetNaming409,
  SetNaming422,
  SetNaming500,
  SetNamingInputBody,
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

export const getListNamingUrl = () => {
  return `/api/v1/naming`;
};

/**
 * @summary List each Content Type's naming settings, with what the save check refuses once renaming is on
 */
export const listNaming = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<NamingNamingList> => {
  return apiFetch<NamingNamingList>(getListNamingUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListNamingQueryKey = () => {
  return [`/api/v1/naming`] as const;
};

export const getListNamingSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listNaming>>,
  TError = ErrorType<ListNaming401 | ListNaming403 | ListNaming409 | ListNaming500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNaming>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListNamingQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listNaming>>> = ({ signal }) =>
    listNaming({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNaming>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListNamingSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listNaming>>>;
export type ListNamingSuspenseQueryError = ErrorType<
  ListNaming401 | ListNaming403 | ListNaming409 | ListNaming500
>;

export function useListNamingSuspense<
  TData = Awaited<ReturnType<typeof listNaming>>,
  TError = ErrorType<ListNaming401 | ListNaming403 | ListNaming409 | ListNaming500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNaming>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNamingSuspense<
  TData = Awaited<ReturnType<typeof listNaming>>,
  TError = ErrorType<ListNaming401 | ListNaming403 | ListNaming409 | ListNaming500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNaming>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNamingSuspense<
  TData = Awaited<ReturnType<typeof listNaming>>,
  TError = ErrorType<ListNaming401 | ListNaming403 | ListNaming409 | ListNaming500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNaming>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List each Content Type's naming settings, with what the save check refuses once renaming is on
 */

export function useListNamingSuspense<
  TData = Awaited<ReturnType<typeof listNaming>>,
  TError = ErrorType<ListNaming401 | ListNaming403 | ListNaming409 | ListNaming500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNaming>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListNamingSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetNamingUrl = (contentType: string) => {
  return `/api/v1/naming/${contentType}`;
};

/**
 * @summary Change a Content Type's naming settings by key, all or none. Every setting must pass the save check as the change leaves them
 */
export const setNaming = async (
  contentType: string,
  setNamingInputBody: SetNamingInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NamingTypeNaming> => {
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
  return apiFetch<NamingTypeNaming>(getSetNamingUrl(contentType), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(setNamingInputBody),
  });
};

export const getSetNamingMutationKey = () => ["setNaming"] as const;

export const getSetNamingMutationOptions = <
  TError = ErrorType<
    SetNaming401 | SetNaming403 | SetNaming404 | SetNaming409 | SetNaming422 | SetNaming500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setNaming>>,
    TError,
    SetNamingMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setNaming>>,
  TError,
  SetNamingMutationVariables,
  TContext
> => {
  const mutationKey = getSetNamingMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setNaming>>,
    SetNamingMutationVariables
  > = (props) => {
    const { contentType, data } = props ?? {};

    return setNaming(contentType, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetNamingMutationResult = NonNullable<Awaited<ReturnType<typeof setNaming>>>;
export type SetNamingMutationBody = SetNamingInputBody;
export type SetNamingMutationError = ErrorType<
  SetNaming401 | SetNaming403 | SetNaming404 | SetNaming409 | SetNaming422 | SetNaming500
>;
export type SetNamingMutationVariables = { contentType: string; data: SetNamingInputBody };

/**
 * @summary Change a Content Type's naming settings by key, all or none. Every setting must pass the save check as the change leaves them
 */
export const useSetNaming = <
  TError = ErrorType<
    SetNaming401 | SetNaming403 | SetNaming404 | SetNaming409 | SetNaming422 | SetNaming500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setNaming>>,
      TError,
      SetNamingMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setNaming>>,
  TError,
  SetNamingMutationVariables,
  TContext
> => {
  return useMutation(getSetNamingMutationOptions(options), queryClient);
};
export const getPreviewNamingUrl = (contentType: string) => {
  return `/api/v1/naming/${contentType}/preview`;
};

/**
 * @summary Name a Content Type's samples, a Work's files and tested file names under a draft of its naming settings, with what the save check refuses. Nothing is stored
 */
export const previewNaming = async (
  contentType: string,
  namingPreviewInputBody: NamingPreviewInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NamingPreview> => {
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
  return apiFetch<NamingPreview>(getPreviewNamingUrl(contentType), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(namingPreviewInputBody),
  });
};

export const getPreviewNamingMutationKey = () => ["previewNaming"] as const;

export const getPreviewNamingMutationOptions = <
  TError = ErrorType<
    | PreviewNaming401
    | PreviewNaming403
    | PreviewNaming404
    | PreviewNaming409
    | PreviewNaming422
    | PreviewNaming500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof previewNaming>>,
    TError,
    PreviewNamingMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof previewNaming>>,
  TError,
  PreviewNamingMutationVariables,
  TContext
> => {
  const mutationKey = getPreviewNamingMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof previewNaming>>,
    PreviewNamingMutationVariables
  > = (props) => {
    const { contentType, data } = props ?? {};

    return previewNaming(contentType, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PreviewNamingMutationResult = NonNullable<Awaited<ReturnType<typeof previewNaming>>>;
export type PreviewNamingMutationBody = NamingPreviewInputBody;
export type PreviewNamingMutationError = ErrorType<
  | PreviewNaming401
  | PreviewNaming403
  | PreviewNaming404
  | PreviewNaming409
  | PreviewNaming422
  | PreviewNaming500
>;
export type PreviewNamingMutationVariables = { contentType: string; data: NamingPreviewInputBody };

/**
 * @summary Name a Content Type's samples, a Work's files and tested file names under a draft of its naming settings, with what the save check refuses. Nothing is stored
 */
export const usePreviewNaming = <
  TError = ErrorType<
    | PreviewNaming401
    | PreviewNaming403
    | PreviewNaming404
    | PreviewNaming409
    | PreviewNaming422
    | PreviewNaming500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof previewNaming>>,
      TError,
      PreviewNamingMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof previewNaming>>,
  TError,
  PreviewNamingMutationVariables,
  TContext
> => {
  return useMutation(getPreviewNamingMutationOptions(options), queryClient);
};
export const getListNamingTokensUrl = (contentType: string) => {
  return `/api/v1/naming/${contentType}/tokens`;
};

/**
 * @summary List the tokens each of a Content Type's templates may name
 */
export const listNamingTokens = async (
  contentType: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NamingTokenList> => {
  return apiFetch<NamingTokenList>(getListNamingTokensUrl(contentType), {
    ...options,
    method: "GET",
  });
};

export const getListNamingTokensQueryKey = (contentType: string) => {
  return [`/api/v1/naming/${contentType}/tokens`] as const;
};

export const getListNamingTokensSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listNamingTokens>>,
  TError = ErrorType<
    | ListNamingTokens401
    | ListNamingTokens403
    | ListNamingTokens404
    | ListNamingTokens409
    | ListNamingTokens422
    | ListNamingTokens500
  >,
>(
  contentType: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNamingTokens>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListNamingTokensQueryKey(contentType);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listNamingTokens>>> = ({ signal }) =>
    listNamingTokens(contentType, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNamingTokens>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListNamingTokensSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listNamingTokens>>
>;
export type ListNamingTokensSuspenseQueryError = ErrorType<
  | ListNamingTokens401
  | ListNamingTokens403
  | ListNamingTokens404
  | ListNamingTokens409
  | ListNamingTokens422
  | ListNamingTokens500
>;

export function useListNamingTokensSuspense<
  TData = Awaited<ReturnType<typeof listNamingTokens>>,
  TError = ErrorType<
    | ListNamingTokens401
    | ListNamingTokens403
    | ListNamingTokens404
    | ListNamingTokens409
    | ListNamingTokens422
    | ListNamingTokens500
  >,
>(
  contentType: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNamingTokens>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNamingTokensSuspense<
  TData = Awaited<ReturnType<typeof listNamingTokens>>,
  TError = ErrorType<
    | ListNamingTokens401
    | ListNamingTokens403
    | ListNamingTokens404
    | ListNamingTokens409
    | ListNamingTokens422
    | ListNamingTokens500
  >,
>(
  contentType: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNamingTokens>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNamingTokensSuspense<
  TData = Awaited<ReturnType<typeof listNamingTokens>>,
  TError = ErrorType<
    | ListNamingTokens401
    | ListNamingTokens403
    | ListNamingTokens404
    | ListNamingTokens409
    | ListNamingTokens422
    | ListNamingTokens500
  >,
>(
  contentType: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNamingTokens>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the tokens each of a Content Type's templates may name
 */

export function useListNamingTokensSuspense<
  TData = Awaited<ReturnType<typeof listNamingTokens>>,
  TError = ErrorType<
    | ListNamingTokens401
    | ListNamingTokens403
    | ListNamingTokens404
    | ListNamingTokens409
    | ListNamingTokens422
    | ListNamingTokens500
  >,
>(
  contentType: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNamingTokens>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListNamingTokensSuspenseQueryOptions(contentType, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
