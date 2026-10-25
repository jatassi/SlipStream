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
  ClearMetadataCache401,
  ClearMetadataCache403,
  ClearMetadataCache409,
  ClearMetadataCache500,
  ClearMetadataKey401,
  ClearMetadataKey403,
  ClearMetadataKey404,
  ClearMetadataKey409,
  ClearMetadataKey422,
  ClearMetadataKey500,
  GetMetadataSettings401,
  GetMetadataSettings403,
  GetMetadataSettings409,
  GetMetadataSettings500,
  ListMetadataProviders401,
  ListMetadataProviders403,
  ListMetadataProviders409,
  ListMetadataProviders500,
  MetadataNewKey,
  MetadataProvider,
  MetadataProviderList,
  MetadataRefreshStarted,
  MetadataSettings,
  MetadataSettingsChange,
  RefreshContentType401,
  RefreshContentType403,
  RefreshContentType404,
  RefreshContentType409,
  RefreshContentType422,
  RefreshContentType500,
  SetMetadataKey401,
  SetMetadataKey403,
  SetMetadataKey404,
  SetMetadataKey409,
  SetMetadataKey422,
  SetMetadataKey500,
  SetMetadataKey502,
  TestMetadataProvider401,
  TestMetadataProvider403,
  TestMetadataProvider404,
  TestMetadataProvider409,
  TestMetadataProvider422,
  TestMetadataProvider500,
  TestMetadataProvider502,
  UpdateMetadataSettings401,
  UpdateMetadataSettings403,
  UpdateMetadataSettings409,
  UpdateMetadataSettings422,
  UpdateMetadataSettings500,
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

export const getRefreshContentTypeUrl = (contentType: string) => {
  return `/api/v1/content-types/${contentType}/refresh`;
};

/**
 * @summary Refresh every Work of a Content Type as a run of Metadata refresh now, or at its next run while one is open
 */
export const refreshContentType = async (
  contentType: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MetadataRefreshStarted> => {
  return apiFetch<MetadataRefreshStarted>(getRefreshContentTypeUrl(contentType), {
    ...options,
    method: "POST",
  });
};

export const getRefreshContentTypeMutationKey = () => ["refreshContentType"] as const;

export const getRefreshContentTypeMutationOptions = <
  TError = ErrorType<
    | RefreshContentType401
    | RefreshContentType403
    | RefreshContentType404
    | RefreshContentType409
    | RefreshContentType422
    | RefreshContentType500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof refreshContentType>>,
    TError,
    RefreshContentTypeMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof refreshContentType>>,
  TError,
  RefreshContentTypeMutationVariables,
  TContext
> => {
  const mutationKey = getRefreshContentTypeMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof refreshContentType>>,
    RefreshContentTypeMutationVariables
  > = (props) => {
    const { contentType } = props ?? {};

    return refreshContentType(contentType, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RefreshContentTypeMutationResult = NonNullable<
  Awaited<ReturnType<typeof refreshContentType>>
>;

export type RefreshContentTypeMutationError = ErrorType<
  | RefreshContentType401
  | RefreshContentType403
  | RefreshContentType404
  | RefreshContentType409
  | RefreshContentType422
  | RefreshContentType500
>;
export type RefreshContentTypeMutationVariables = { contentType: string };

/**
 * @summary Refresh every Work of a Content Type as a run of Metadata refresh now, or at its next run while one is open
 */
export const useRefreshContentType = <
  TError = ErrorType<
    | RefreshContentType401
    | RefreshContentType403
    | RefreshContentType404
    | RefreshContentType409
    | RefreshContentType422
    | RefreshContentType500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof refreshContentType>>,
      TError,
      RefreshContentTypeMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof refreshContentType>>,
  TError,
  RefreshContentTypeMutationVariables,
  TContext
> => {
  return useMutation(getRefreshContentTypeMutationOptions(options), queryClient);
};
export const getClearMetadataCacheUrl = () => {
  return `/api/v1/metadata/cache`;
};

/**
 * @summary Have every Node reading metadata drop what it cached, the next time it reads
 */
export const clearMetadataCache = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getClearMetadataCacheUrl(), {
    ...options,
    method: "DELETE",
  });
};

export const getClearMetadataCacheMutationKey = () => ["clearMetadataCache"] as const;

export const getClearMetadataCacheMutationOptions = <
  TError = ErrorType<
    ClearMetadataCache401 | ClearMetadataCache403 | ClearMetadataCache409 | ClearMetadataCache500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof clearMetadataCache>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof clearMetadataCache>>, TError, void, TContext> => {
  const mutationKey = getClearMetadataCacheMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof clearMetadataCache>>, void> = () => {
    return clearMetadataCache(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ClearMetadataCacheMutationResult = NonNullable<
  Awaited<ReturnType<typeof clearMetadataCache>>
>;

export type ClearMetadataCacheMutationError = ErrorType<
  ClearMetadataCache401 | ClearMetadataCache403 | ClearMetadataCache409 | ClearMetadataCache500
>;

/**
 * @summary Have every Node reading metadata drop what it cached, the next time it reads
 */
export const useClearMetadataCache = <
  TError = ErrorType<
    ClearMetadataCache401 | ClearMetadataCache403 | ClearMetadataCache409 | ClearMetadataCache500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof clearMetadataCache>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof clearMetadataCache>>, TError, void, TContext> => {
  return useMutation(getClearMetadataCacheMutationOptions(options), queryClient);
};
export const getListMetadataProvidersUrl = () => {
  return `/api/v1/metadata/providers`;
};

/**
 * @summary List each metadata provider's status and whose key it is called with
 */
export const listMetadataProviders = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<MetadataProviderList> => {
  return apiFetch<MetadataProviderList>(getListMetadataProvidersUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListMetadataProvidersQueryKey = () => {
  return [`/api/v1/metadata/providers`] as const;
};

export const getListMetadataProvidersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMetadataProviders>>,
  TError = ErrorType<
    | ListMetadataProviders401
    | ListMetadataProviders403
    | ListMetadataProviders409
    | ListMetadataProviders500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMetadataProviders>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMetadataProvidersQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMetadataProviders>>> = ({ signal }) =>
    listMetadataProviders({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listMetadataProviders>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMetadataProvidersSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listMetadataProviders>>
>;
export type ListMetadataProvidersSuspenseQueryError = ErrorType<
  | ListMetadataProviders401
  | ListMetadataProviders403
  | ListMetadataProviders409
  | ListMetadataProviders500
>;

export function useListMetadataProvidersSuspense<
  TData = Awaited<ReturnType<typeof listMetadataProviders>>,
  TError = ErrorType<
    | ListMetadataProviders401
    | ListMetadataProviders403
    | ListMetadataProviders409
    | ListMetadataProviders500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMetadataProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMetadataProvidersSuspense<
  TData = Awaited<ReturnType<typeof listMetadataProviders>>,
  TError = ErrorType<
    | ListMetadataProviders401
    | ListMetadataProviders403
    | ListMetadataProviders409
    | ListMetadataProviders500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMetadataProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMetadataProvidersSuspense<
  TData = Awaited<ReturnType<typeof listMetadataProviders>>,
  TError = ErrorType<
    | ListMetadataProviders401
    | ListMetadataProviders403
    | ListMetadataProviders409
    | ListMetadataProviders500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMetadataProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List each metadata provider's status and whose key it is called with
 */

export function useListMetadataProvidersSuspense<
  TData = Awaited<ReturnType<typeof listMetadataProviders>>,
  TError = ErrorType<
    | ListMetadataProviders401
    | ListMetadataProviders403
    | ListMetadataProviders409
    | ListMetadataProviders500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMetadataProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMetadataProvidersSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getClearMetadataKeyUrl = (provider: string) => {
  return `/api/v1/metadata/providers/${provider}/key`;
};

/**
 * @summary Remove your key for a provider and go back to SlipStream's
 */
export const clearMetadataKey = async (
  provider: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MetadataProvider> => {
  return apiFetch<MetadataProvider>(getClearMetadataKeyUrl(provider), {
    ...options,
    method: "DELETE",
  });
};

export const getClearMetadataKeyMutationKey = () => ["clearMetadataKey"] as const;

export const getClearMetadataKeyMutationOptions = <
  TError = ErrorType<
    | ClearMetadataKey401
    | ClearMetadataKey403
    | ClearMetadataKey404
    | ClearMetadataKey409
    | ClearMetadataKey422
    | ClearMetadataKey500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof clearMetadataKey>>,
    TError,
    ClearMetadataKeyMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof clearMetadataKey>>,
  TError,
  ClearMetadataKeyMutationVariables,
  TContext
> => {
  const mutationKey = getClearMetadataKeyMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof clearMetadataKey>>,
    ClearMetadataKeyMutationVariables
  > = (props) => {
    const { provider } = props ?? {};

    return clearMetadataKey(provider, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ClearMetadataKeyMutationResult = NonNullable<
  Awaited<ReturnType<typeof clearMetadataKey>>
>;

export type ClearMetadataKeyMutationError = ErrorType<
  | ClearMetadataKey401
  | ClearMetadataKey403
  | ClearMetadataKey404
  | ClearMetadataKey409
  | ClearMetadataKey422
  | ClearMetadataKey500
>;
export type ClearMetadataKeyMutationVariables = { provider: string };

/**
 * @summary Remove your key for a provider and go back to SlipStream's
 */
export const useClearMetadataKey = <
  TError = ErrorType<
    | ClearMetadataKey401
    | ClearMetadataKey403
    | ClearMetadataKey404
    | ClearMetadataKey409
    | ClearMetadataKey422
    | ClearMetadataKey500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof clearMetadataKey>>,
      TError,
      ClearMetadataKeyMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof clearMetadataKey>>,
  TError,
  ClearMetadataKeyMutationVariables,
  TContext
> => {
  return useMutation(getClearMetadataKeyMutationOptions(options), queryClient);
};
export const getSetMetadataKeyUrl = (provider: string) => {
  return `/api/v1/metadata/providers/${provider}/key`;
};

/**
 * @summary Use your own key for a provider, once the provider takes it on a test call
 */
export const setMetadataKey = async (
  provider: string,
  metadataNewKey: MetadataNewKey,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MetadataProvider> => {
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
  return apiFetch<MetadataProvider>(getSetMetadataKeyUrl(provider), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(metadataNewKey),
  });
};

export const getSetMetadataKeyMutationKey = () => ["setMetadataKey"] as const;

export const getSetMetadataKeyMutationOptions = <
  TError = ErrorType<
    | SetMetadataKey401
    | SetMetadataKey403
    | SetMetadataKey404
    | SetMetadataKey409
    | SetMetadataKey422
    | SetMetadataKey500
    | SetMetadataKey502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setMetadataKey>>,
    TError,
    SetMetadataKeyMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setMetadataKey>>,
  TError,
  SetMetadataKeyMutationVariables,
  TContext
> => {
  const mutationKey = getSetMetadataKeyMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setMetadataKey>>,
    SetMetadataKeyMutationVariables
  > = (props) => {
    const { provider, data } = props ?? {};

    return setMetadataKey(provider, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetMetadataKeyMutationResult = NonNullable<Awaited<ReturnType<typeof setMetadataKey>>>;
export type SetMetadataKeyMutationBody = MetadataNewKey;
export type SetMetadataKeyMutationError = ErrorType<
  | SetMetadataKey401
  | SetMetadataKey403
  | SetMetadataKey404
  | SetMetadataKey409
  | SetMetadataKey422
  | SetMetadataKey500
  | SetMetadataKey502
>;
export type SetMetadataKeyMutationVariables = { provider: string; data: MetadataNewKey };

/**
 * @summary Use your own key for a provider, once the provider takes it on a test call
 */
export const useSetMetadataKey = <
  TError = ErrorType<
    | SetMetadataKey401
    | SetMetadataKey403
    | SetMetadataKey404
    | SetMetadataKey409
    | SetMetadataKey422
    | SetMetadataKey500
    | SetMetadataKey502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setMetadataKey>>,
      TError,
      SetMetadataKeyMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setMetadataKey>>,
  TError,
  SetMetadataKeyMutationVariables,
  TContext
> => {
  return useMutation(getSetMetadataKeyMutationOptions(options), queryClient);
};
export const getTestMetadataProviderUrl = (provider: string) => {
  return `/api/v1/metadata/providers/${provider}/test`;
};

/**
 * @summary Make one cheap call to a provider with the key in use, and return the provider's status after it
 */
export const testMetadataProvider = async (
  provider: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MetadataProvider> => {
  return apiFetch<MetadataProvider>(getTestMetadataProviderUrl(provider), {
    ...options,
    method: "POST",
  });
};

export const getTestMetadataProviderMutationKey = () => ["testMetadataProvider"] as const;

export const getTestMetadataProviderMutationOptions = <
  TError = ErrorType<
    | TestMetadataProvider401
    | TestMetadataProvider403
    | TestMetadataProvider404
    | TestMetadataProvider409
    | TestMetadataProvider422
    | TestMetadataProvider500
    | TestMetadataProvider502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof testMetadataProvider>>,
    TError,
    TestMetadataProviderMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof testMetadataProvider>>,
  TError,
  TestMetadataProviderMutationVariables,
  TContext
> => {
  const mutationKey = getTestMetadataProviderMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof testMetadataProvider>>,
    TestMetadataProviderMutationVariables
  > = (props) => {
    const { provider } = props ?? {};

    return testMetadataProvider(provider, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type TestMetadataProviderMutationResult = NonNullable<
  Awaited<ReturnType<typeof testMetadataProvider>>
>;

export type TestMetadataProviderMutationError = ErrorType<
  | TestMetadataProvider401
  | TestMetadataProvider403
  | TestMetadataProvider404
  | TestMetadataProvider409
  | TestMetadataProvider422
  | TestMetadataProvider500
  | TestMetadataProvider502
>;
export type TestMetadataProviderMutationVariables = { provider: string };

/**
 * @summary Make one cheap call to a provider with the key in use, and return the provider's status after it
 */
export const useTestMetadataProvider = <
  TError = ErrorType<
    | TestMetadataProvider401
    | TestMetadataProvider403
    | TestMetadataProvider404
    | TestMetadataProvider409
    | TestMetadataProvider422
    | TestMetadataProvider500
    | TestMetadataProvider502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof testMetadataProvider>>,
      TError,
      TestMetadataProviderMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof testMetadataProvider>>,
  TError,
  TestMetadataProviderMutationVariables,
  TContext
> => {
  return useMutation(getTestMetadataProviderMutationOptions(options), queryClient);
};
export const getGetMetadataSettingsUrl = () => {
  return `/api/v1/metadata/settings`;
};

/**
 * @summary Get the metadata settings, with the language and region options
 */
export const getMetadataSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<MetadataSettings> => {
  return apiFetch<MetadataSettings>(getGetMetadataSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetMetadataSettingsQueryKey = () => {
  return [`/api/v1/metadata/settings`] as const;
};

export const getGetMetadataSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getMetadataSettings>>,
  TError = ErrorType<
    | GetMetadataSettings401
    | GetMetadataSettings403
    | GetMetadataSettings409
    | GetMetadataSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMetadataSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetMetadataSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getMetadataSettings>>> = ({ signal }) =>
    getMetadataSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMetadataSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetMetadataSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getMetadataSettings>>
>;
export type GetMetadataSettingsSuspenseQueryError = ErrorType<
  GetMetadataSettings401 | GetMetadataSettings403 | GetMetadataSettings409 | GetMetadataSettings500
>;

export function useGetMetadataSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMetadataSettings>>,
  TError = ErrorType<
    | GetMetadataSettings401
    | GetMetadataSettings403
    | GetMetadataSettings409
    | GetMetadataSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMetadataSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMetadataSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMetadataSettings>>,
  TError = ErrorType<
    | GetMetadataSettings401
    | GetMetadataSettings403
    | GetMetadataSettings409
    | GetMetadataSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMetadataSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMetadataSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMetadataSettings>>,
  TError = ErrorType<
    | GetMetadataSettings401
    | GetMetadataSettings403
    | GetMetadataSettings409
    | GetMetadataSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMetadataSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the metadata settings, with the language and region options
 */

export function useGetMetadataSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMetadataSettings>>,
  TError = ErrorType<
    | GetMetadataSettings401
    | GetMetadataSettings403
    | GetMetadataSettings409
    | GetMetadataSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMetadataSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetMetadataSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateMetadataSettingsUrl = () => {
  return `/api/v1/metadata/settings`;
};

/**
 * @summary Change metadata settings by key, all or none
 */
export const updateMetadataSettings = async (
  metadataSettingsChange: MetadataSettingsChange,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MetadataSettings> => {
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
  return apiFetch<MetadataSettings>(getUpdateMetadataSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(metadataSettingsChange),
  });
};

export const getUpdateMetadataSettingsMutationKey = () => ["updateMetadataSettings"] as const;

export const getUpdateMetadataSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateMetadataSettings401
    | UpdateMetadataSettings403
    | UpdateMetadataSettings409
    | UpdateMetadataSettings422
    | UpdateMetadataSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateMetadataSettings>>,
    TError,
    UpdateMetadataSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateMetadataSettings>>,
  TError,
  UpdateMetadataSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateMetadataSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateMetadataSettings>>,
    UpdateMetadataSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateMetadataSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateMetadataSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateMetadataSettings>>
>;
export type UpdateMetadataSettingsMutationBody = MetadataSettingsChange;
export type UpdateMetadataSettingsMutationError = ErrorType<
  | UpdateMetadataSettings401
  | UpdateMetadataSettings403
  | UpdateMetadataSettings409
  | UpdateMetadataSettings422
  | UpdateMetadataSettings500
>;
export type UpdateMetadataSettingsMutationVariables = { data: MetadataSettingsChange };

/**
 * @summary Change metadata settings by key, all or none
 */
export const useUpdateMetadataSettings = <
  TError = ErrorType<
    | UpdateMetadataSettings401
    | UpdateMetadataSettings403
    | UpdateMetadataSettings409
    | UpdateMetadataSettings422
    | UpdateMetadataSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateMetadataSettings>>,
      TError,
      UpdateMetadataSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateMetadataSettings>>,
  TError,
  UpdateMetadataSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateMetadataSettingsMutationOptions(options), queryClient);
};
