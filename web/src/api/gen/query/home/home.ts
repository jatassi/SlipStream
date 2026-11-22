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
  GetHome401,
  GetHome403,
  GetHome409,
  GetHome422,
  GetHome500,
  GetHomeParams,
  GetHomeSettings401,
  GetHomeSettings403,
  GetHomeSettings409,
  GetHomeSettings500,
  HomeSettings,
  HomeSummary,
  SettingsInputBody,
  UpdateHomeSettings401,
  UpdateHomeSettings403,
  UpdateHomeSettings409,
  UpdateHomeSettings422,
  UpdateHomeSettings500,
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

export const getGetHomeUrl = (params: GetHomeParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/home?${stringifiedParams}` : `/api/v1/home`;
};

/**
 * @summary Get Home: the moving Grabs, what needs attention and the newest finished Imports
 */
export const getHome = async (
  params: GetHomeParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<HomeSummary> => {
  return apiFetch<HomeSummary>(getGetHomeUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getGetHomeQueryKey = (params?: GetHomeParams) => {
  return [`/api/v1/home`, ...(params ? [params] : [])] as const;
};

export const getGetHomeSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getHome>>,
  TError = ErrorType<GetHome401 | GetHome403 | GetHome409 | GetHome422 | GetHome500>,
>(
  params: GetHomeParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHome>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetHomeQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getHome>>> = ({ signal }) =>
    getHome(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHome>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetHomeSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getHome>>>;
export type GetHomeSuspenseQueryError = ErrorType<
  GetHome401 | GetHome403 | GetHome409 | GetHome422 | GetHome500
>;

export function useGetHomeSuspense<
  TData = Awaited<ReturnType<typeof getHome>>,
  TError = ErrorType<GetHome401 | GetHome403 | GetHome409 | GetHome422 | GetHome500>,
>(
  params: GetHomeParams,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHome>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHomeSuspense<
  TData = Awaited<ReturnType<typeof getHome>>,
  TError = ErrorType<GetHome401 | GetHome403 | GetHome409 | GetHome422 | GetHome500>,
>(
  params: GetHomeParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHome>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHomeSuspense<
  TData = Awaited<ReturnType<typeof getHome>>,
  TError = ErrorType<GetHome401 | GetHome403 | GetHome409 | GetHome422 | GetHome500>,
>(
  params: GetHomeParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHome>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get Home: the moving Grabs, what needs attention and the newest finished Imports
 */

export function useGetHomeSuspense<
  TData = Awaited<ReturnType<typeof getHome>>,
  TError = ErrorType<GetHome401 | GetHome403 | GetHome409 | GetHome422 | GetHome500>,
>(
  params: GetHomeParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHome>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetHomeSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetHomeSettingsUrl = () => {
  return `/api/v1/home-settings`;
};

/**
 * @summary Get Home's settings
 */
export const getHomeSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<HomeSettings> => {
  return apiFetch<HomeSettings>(getGetHomeSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetHomeSettingsQueryKey = () => {
  return [`/api/v1/home-settings`] as const;
};

export const getGetHomeSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getHomeSettings>>,
  TError = ErrorType<
    GetHomeSettings401 | GetHomeSettings403 | GetHomeSettings409 | GetHomeSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHomeSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetHomeSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getHomeSettings>>> = ({ signal }) =>
    getHomeSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHomeSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetHomeSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getHomeSettings>>
>;
export type GetHomeSettingsSuspenseQueryError = ErrorType<
  GetHomeSettings401 | GetHomeSettings403 | GetHomeSettings409 | GetHomeSettings500
>;

export function useGetHomeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getHomeSettings>>,
  TError = ErrorType<
    GetHomeSettings401 | GetHomeSettings403 | GetHomeSettings409 | GetHomeSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHomeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHomeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getHomeSettings>>,
  TError = ErrorType<
    GetHomeSettings401 | GetHomeSettings403 | GetHomeSettings409 | GetHomeSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHomeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHomeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getHomeSettings>>,
  TError = ErrorType<
    GetHomeSettings401 | GetHomeSettings403 | GetHomeSettings409 | GetHomeSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHomeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get Home's settings
 */

export function useGetHomeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getHomeSettings>>,
  TError = ErrorType<
    GetHomeSettings401 | GetHomeSettings403 | GetHomeSettings409 | GetHomeSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHomeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetHomeSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateHomeSettingsUrl = () => {
  return `/api/v1/home-settings`;
};

/**
 * @summary Change Home's settings by key, all or none
 */
export const updateHomeSettings = async (
  settingsInputBody: SettingsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<HomeSettings> => {
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
  return apiFetch<HomeSettings>(getUpdateHomeSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(settingsInputBody),
  });
};

export const getUpdateHomeSettingsMutationKey = () => ["updateHomeSettings"] as const;

export const getUpdateHomeSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateHomeSettings401
    | UpdateHomeSettings403
    | UpdateHomeSettings409
    | UpdateHomeSettings422
    | UpdateHomeSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateHomeSettings>>,
    TError,
    UpdateHomeSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateHomeSettings>>,
  TError,
  UpdateHomeSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateHomeSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateHomeSettings>>,
    UpdateHomeSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateHomeSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateHomeSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateHomeSettings>>
>;
export type UpdateHomeSettingsMutationBody = SettingsInputBody;
export type UpdateHomeSettingsMutationError = ErrorType<
  | UpdateHomeSettings401
  | UpdateHomeSettings403
  | UpdateHomeSettings409
  | UpdateHomeSettings422
  | UpdateHomeSettings500
>;
export type UpdateHomeSettingsMutationVariables = { data: SettingsInputBody };

/**
 * @summary Change Home's settings by key, all or none
 */
export const useUpdateHomeSettings = <
  TError = ErrorType<
    | UpdateHomeSettings401
    | UpdateHomeSettings403
    | UpdateHomeSettings409
    | UpdateHomeSettings422
    | UpdateHomeSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateHomeSettings>>,
      TError,
      UpdateHomeSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateHomeSettings>>,
  TError,
  UpdateHomeSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateHomeSettingsMutationOptions(options), queryClient);
};
