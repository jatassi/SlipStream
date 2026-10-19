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
  CheckHealthCategory401,
  CheckHealthCategory403,
  CheckHealthCategory404,
  CheckHealthCategory409,
  CheckHealthCategory422,
  CheckHealthCategory500,
  CheckHealthItem401,
  CheckHealthItem403,
  CheckHealthItem404,
  CheckHealthItem409,
  CheckHealthItem422,
  CheckHealthItem500,
  CheckSystemHealth401,
  CheckSystemHealth403,
  CheckSystemHealth409,
  CheckSystemHealth500,
  DeleteHealthVolume401,
  DeleteHealthVolume403,
  DeleteHealthVolume409,
  DeleteHealthVolume422,
  DeleteHealthVolume500,
  DeleteHealthVolumeParams,
  GetHealthSettings401,
  GetHealthSettings403,
  GetHealthSettings409,
  GetHealthSettings500,
  GetSystemHealth401,
  GetSystemHealth403,
  GetSystemHealth409,
  GetSystemHealth500,
  HealthChecked,
  HealthReport,
  HealthSettings,
  HealthSettingsInputBody,
  HealthVolume,
  HealthVolumeInputBody,
  HealthVolumeList,
  ListHealthVolumes401,
  ListHealthVolumes403,
  ListHealthVolumes409,
  ListHealthVolumes500,
  UpdateHealthSettings401,
  UpdateHealthSettings403,
  UpdateHealthSettings409,
  UpdateHealthSettings422,
  UpdateHealthSettings500,
  UpdateHealthVolume401,
  UpdateHealthVolume403,
  UpdateHealthVolume404,
  UpdateHealthVolume409,
  UpdateHealthVolume422,
  UpdateHealthVolume500,
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

export const getGetSystemHealthUrl = () => {
  return `/api/v1/system/health`;
};

/**
 * @summary Get what is wrong with SlipStream, by category
 */
export const getSystemHealth = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<HealthReport> => {
  return apiFetch<HealthReport>(getGetSystemHealthUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetSystemHealthQueryKey = () => {
  return [`/api/v1/system/health`] as const;
};

export const getGetSystemHealthSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSystemHealth>>,
  TError = ErrorType<
    GetSystemHealth401 | GetSystemHealth403 | GetSystemHealth409 | GetSystemHealth500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystemHealth>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSystemHealthQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSystemHealth>>> = ({ signal }) =>
    getSystemHealth({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystemHealth>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSystemHealthSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getSystemHealth>>
>;
export type GetSystemHealthSuspenseQueryError = ErrorType<
  GetSystemHealth401 | GetSystemHealth403 | GetSystemHealth409 | GetSystemHealth500
>;

export function useGetSystemHealthSuspense<
  TData = Awaited<ReturnType<typeof getSystemHealth>>,
  TError = ErrorType<
    GetSystemHealth401 | GetSystemHealth403 | GetSystemHealth409 | GetSystemHealth500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystemHealth>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSystemHealthSuspense<
  TData = Awaited<ReturnType<typeof getSystemHealth>>,
  TError = ErrorType<
    GetSystemHealth401 | GetSystemHealth403 | GetSystemHealth409 | GetSystemHealth500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystemHealth>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSystemHealthSuspense<
  TData = Awaited<ReturnType<typeof getSystemHealth>>,
  TError = ErrorType<
    GetSystemHealth401 | GetSystemHealth403 | GetSystemHealth409 | GetSystemHealth500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystemHealth>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get what is wrong with SlipStream, by category
 */

export function useGetSystemHealthSuspense<
  TData = Awaited<ReturnType<typeof getSystemHealth>>,
  TError = ErrorType<
    GetSystemHealth401 | GetSystemHealth403 | GetSystemHealth409 | GetSystemHealth500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystemHealth>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSystemHealthSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCheckSystemHealthUrl = () => {
  return `/api/v1/system/health/check`;
};

/**
 * @summary Check every category now
 */
export const checkSystemHealth = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<HealthChecked> => {
  return apiFetch<HealthChecked>(getCheckSystemHealthUrl(), {
    ...options,
    method: "POST",
  });
};

export const getCheckSystemHealthMutationKey = () => ["checkSystemHealth"] as const;

export const getCheckSystemHealthMutationOptions = <
  TError = ErrorType<
    CheckSystemHealth401 | CheckSystemHealth403 | CheckSystemHealth409 | CheckSystemHealth500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof checkSystemHealth>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof checkSystemHealth>>, TError, void, TContext> => {
  const mutationKey = getCheckSystemHealthMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof checkSystemHealth>>, void> = () => {
    return checkSystemHealth(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CheckSystemHealthMutationResult = NonNullable<
  Awaited<ReturnType<typeof checkSystemHealth>>
>;

export type CheckSystemHealthMutationError = ErrorType<
  CheckSystemHealth401 | CheckSystemHealth403 | CheckSystemHealth409 | CheckSystemHealth500
>;

/**
 * @summary Check every category now
 */
export const useCheckSystemHealth = <
  TError = ErrorType<
    CheckSystemHealth401 | CheckSystemHealth403 | CheckSystemHealth409 | CheckSystemHealth500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof checkSystemHealth>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof checkSystemHealth>>, TError, void, TContext> => {
  return useMutation(getCheckSystemHealthMutationOptions(options), queryClient);
};
export const getGetHealthSettingsUrl = () => {
  return `/api/v1/system/health/settings`;
};

/**
 * @summary Get the global Storage levels
 */
export const getHealthSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<HealthSettings> => {
  return apiFetch<HealthSettings>(getGetHealthSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetHealthSettingsQueryKey = () => {
  return [`/api/v1/system/health/settings`] as const;
};

export const getGetHealthSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getHealthSettings>>,
  TError = ErrorType<
    GetHealthSettings401 | GetHealthSettings403 | GetHealthSettings409 | GetHealthSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealthSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetHealthSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getHealthSettings>>> = ({ signal }) =>
    getHealthSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealthSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetHealthSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getHealthSettings>>
>;
export type GetHealthSettingsSuspenseQueryError = ErrorType<
  GetHealthSettings401 | GetHealthSettings403 | GetHealthSettings409 | GetHealthSettings500
>;

export function useGetHealthSettingsSuspense<
  TData = Awaited<ReturnType<typeof getHealthSettings>>,
  TError = ErrorType<
    GetHealthSettings401 | GetHealthSettings403 | GetHealthSettings409 | GetHealthSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealthSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHealthSettingsSuspense<
  TData = Awaited<ReturnType<typeof getHealthSettings>>,
  TError = ErrorType<
    GetHealthSettings401 | GetHealthSettings403 | GetHealthSettings409 | GetHealthSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealthSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHealthSettingsSuspense<
  TData = Awaited<ReturnType<typeof getHealthSettings>>,
  TError = ErrorType<
    GetHealthSettings401 | GetHealthSettings403 | GetHealthSettings409 | GetHealthSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealthSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the global Storage levels
 */

export function useGetHealthSettingsSuspense<
  TData = Awaited<ReturnType<typeof getHealthSettings>>,
  TError = ErrorType<
    GetHealthSettings401 | GetHealthSettings403 | GetHealthSettings409 | GetHealthSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealthSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetHealthSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateHealthSettingsUrl = () => {
  return `/api/v1/system/health/settings`;
};

/**
 * @summary Change the global Storage levels
 */
export const updateHealthSettings = async (
  healthSettingsInputBody: HealthSettingsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<HealthSettings> => {
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
  return apiFetch<HealthSettings>(getUpdateHealthSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(healthSettingsInputBody),
  });
};

export const getUpdateHealthSettingsMutationKey = () => ["updateHealthSettings"] as const;

export const getUpdateHealthSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateHealthSettings401
    | UpdateHealthSettings403
    | UpdateHealthSettings409
    | UpdateHealthSettings422
    | UpdateHealthSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateHealthSettings>>,
    TError,
    UpdateHealthSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateHealthSettings>>,
  TError,
  UpdateHealthSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateHealthSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateHealthSettings>>,
    UpdateHealthSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateHealthSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateHealthSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateHealthSettings>>
>;
export type UpdateHealthSettingsMutationBody = HealthSettingsInputBody;
export type UpdateHealthSettingsMutationError = ErrorType<
  | UpdateHealthSettings401
  | UpdateHealthSettings403
  | UpdateHealthSettings409
  | UpdateHealthSettings422
  | UpdateHealthSettings500
>;
export type UpdateHealthSettingsMutationVariables = { data: HealthSettingsInputBody };

/**
 * @summary Change the global Storage levels
 */
export const useUpdateHealthSettings = <
  TError = ErrorType<
    | UpdateHealthSettings401
    | UpdateHealthSettings403
    | UpdateHealthSettings409
    | UpdateHealthSettings422
    | UpdateHealthSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateHealthSettings>>,
      TError,
      UpdateHealthSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateHealthSettings>>,
  TError,
  UpdateHealthSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateHealthSettingsMutationOptions(options), queryClient);
};
export const getDeleteHealthVolumeUrl = (params: DeleteHealthVolumeParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/system/health/volumes?${stringifiedParams}`
    : `/api/v1/system/health/volumes`;
};

/**
 * @summary Remove a volume's own levels
 */
export const deleteHealthVolume = async (
  params: DeleteHealthVolumeParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeleteHealthVolumeUrl(params), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteHealthVolumeMutationKey = () => ["deleteHealthVolume"] as const;

export const getDeleteHealthVolumeMutationOptions = <
  TError = ErrorType<
    | DeleteHealthVolume401
    | DeleteHealthVolume403
    | DeleteHealthVolume409
    | DeleteHealthVolume422
    | DeleteHealthVolume500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteHealthVolume>>,
    TError,
    DeleteHealthVolumeMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteHealthVolume>>,
  TError,
  DeleteHealthVolumeMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteHealthVolumeMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteHealthVolume>>,
    DeleteHealthVolumeMutationVariables
  > = (props) => {
    const { params } = props ?? {};

    return deleteHealthVolume(params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteHealthVolumeMutationResult = NonNullable<
  Awaited<ReturnType<typeof deleteHealthVolume>>
>;

export type DeleteHealthVolumeMutationError = ErrorType<
  | DeleteHealthVolume401
  | DeleteHealthVolume403
  | DeleteHealthVolume409
  | DeleteHealthVolume422
  | DeleteHealthVolume500
>;
export type DeleteHealthVolumeMutationVariables = { params: DeleteHealthVolumeParams };

/**
 * @summary Remove a volume's own levels
 */
export const useDeleteHealthVolume = <
  TError = ErrorType<
    | DeleteHealthVolume401
    | DeleteHealthVolume403
    | DeleteHealthVolume409
    | DeleteHealthVolume422
    | DeleteHealthVolume500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteHealthVolume>>,
      TError,
      DeleteHealthVolumeMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteHealthVolume>>,
  TError,
  DeleteHealthVolumeMutationVariables,
  TContext
> => {
  return useMutation(getDeleteHealthVolumeMutationOptions(options), queryClient);
};
export const getListHealthVolumesUrl = () => {
  return `/api/v1/system/health/volumes`;
};

/**
 * @summary List every volume with the levels it follows
 */
export const listHealthVolumes = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<HealthVolumeList> => {
  return apiFetch<HealthVolumeList>(getListHealthVolumesUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListHealthVolumesQueryKey = () => {
  return [`/api/v1/system/health/volumes`] as const;
};

export const getListHealthVolumesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listHealthVolumes>>,
  TError = ErrorType<
    ListHealthVolumes401 | ListHealthVolumes403 | ListHealthVolumes409 | ListHealthVolumes500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHealthVolumes>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListHealthVolumesQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listHealthVolumes>>> = ({ signal }) =>
    listHealthVolumes({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHealthVolumes>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListHealthVolumesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listHealthVolumes>>
>;
export type ListHealthVolumesSuspenseQueryError = ErrorType<
  ListHealthVolumes401 | ListHealthVolumes403 | ListHealthVolumes409 | ListHealthVolumes500
>;

export function useListHealthVolumesSuspense<
  TData = Awaited<ReturnType<typeof listHealthVolumes>>,
  TError = ErrorType<
    ListHealthVolumes401 | ListHealthVolumes403 | ListHealthVolumes409 | ListHealthVolumes500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHealthVolumes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListHealthVolumesSuspense<
  TData = Awaited<ReturnType<typeof listHealthVolumes>>,
  TError = ErrorType<
    ListHealthVolumes401 | ListHealthVolumes403 | ListHealthVolumes409 | ListHealthVolumes500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHealthVolumes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListHealthVolumesSuspense<
  TData = Awaited<ReturnType<typeof listHealthVolumes>>,
  TError = ErrorType<
    ListHealthVolumes401 | ListHealthVolumes403 | ListHealthVolumes409 | ListHealthVolumes500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHealthVolumes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List every volume with the levels it follows
 */

export function useListHealthVolumesSuspense<
  TData = Awaited<ReturnType<typeof listHealthVolumes>>,
  TError = ErrorType<
    ListHealthVolumes401 | ListHealthVolumes403 | ListHealthVolumes409 | ListHealthVolumes500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listHealthVolumes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListHealthVolumesSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateHealthVolumeUrl = () => {
  return `/api/v1/system/health/volumes`;
};

/**
 * @summary Set or reset a volume's own levels
 */
export const updateHealthVolume = async (
  healthVolumeInputBody: HealthVolumeInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<HealthVolume> => {
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
  return apiFetch<HealthVolume>(getUpdateHealthVolumeUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(healthVolumeInputBody),
  });
};

export const getUpdateHealthVolumeMutationKey = () => ["updateHealthVolume"] as const;

export const getUpdateHealthVolumeMutationOptions = <
  TError = ErrorType<
    | UpdateHealthVolume401
    | UpdateHealthVolume403
    | UpdateHealthVolume404
    | UpdateHealthVolume409
    | UpdateHealthVolume422
    | UpdateHealthVolume500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateHealthVolume>>,
    TError,
    UpdateHealthVolumeMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateHealthVolume>>,
  TError,
  UpdateHealthVolumeMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateHealthVolumeMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateHealthVolume>>,
    UpdateHealthVolumeMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateHealthVolume(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateHealthVolumeMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateHealthVolume>>
>;
export type UpdateHealthVolumeMutationBody = HealthVolumeInputBody;
export type UpdateHealthVolumeMutationError = ErrorType<
  | UpdateHealthVolume401
  | UpdateHealthVolume403
  | UpdateHealthVolume404
  | UpdateHealthVolume409
  | UpdateHealthVolume422
  | UpdateHealthVolume500
>;
export type UpdateHealthVolumeMutationVariables = { data: HealthVolumeInputBody };

/**
 * @summary Set or reset a volume's own levels
 */
export const useUpdateHealthVolume = <
  TError = ErrorType<
    | UpdateHealthVolume401
    | UpdateHealthVolume403
    | UpdateHealthVolume404
    | UpdateHealthVolume409
    | UpdateHealthVolume422
    | UpdateHealthVolume500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateHealthVolume>>,
      TError,
      UpdateHealthVolumeMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateHealthVolume>>,
  TError,
  UpdateHealthVolumeMutationVariables,
  TContext
> => {
  return useMutation(getUpdateHealthVolumeMutationOptions(options), queryClient);
};
export const getCheckHealthCategoryUrl = (category: string) => {
  return `/api/v1/system/health/${category}/check`;
};

/**
 * @summary Check one category now
 */
export const checkHealthCategory = async (
  category: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<HealthChecked> => {
  return apiFetch<HealthChecked>(getCheckHealthCategoryUrl(category), {
    ...options,
    method: "POST",
  });
};

export const getCheckHealthCategoryMutationKey = () => ["checkHealthCategory"] as const;

export const getCheckHealthCategoryMutationOptions = <
  TError = ErrorType<
    | CheckHealthCategory401
    | CheckHealthCategory403
    | CheckHealthCategory404
    | CheckHealthCategory409
    | CheckHealthCategory422
    | CheckHealthCategory500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof checkHealthCategory>>,
    TError,
    CheckHealthCategoryMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof checkHealthCategory>>,
  TError,
  CheckHealthCategoryMutationVariables,
  TContext
> => {
  const mutationKey = getCheckHealthCategoryMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof checkHealthCategory>>,
    CheckHealthCategoryMutationVariables
  > = (props) => {
    const { category } = props ?? {};

    return checkHealthCategory(category, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CheckHealthCategoryMutationResult = NonNullable<
  Awaited<ReturnType<typeof checkHealthCategory>>
>;

export type CheckHealthCategoryMutationError = ErrorType<
  | CheckHealthCategory401
  | CheckHealthCategory403
  | CheckHealthCategory404
  | CheckHealthCategory409
  | CheckHealthCategory422
  | CheckHealthCategory500
>;
export type CheckHealthCategoryMutationVariables = { category: string };

/**
 * @summary Check one category now
 */
export const useCheckHealthCategory = <
  TError = ErrorType<
    | CheckHealthCategory401
    | CheckHealthCategory403
    | CheckHealthCategory404
    | CheckHealthCategory409
    | CheckHealthCategory422
    | CheckHealthCategory500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof checkHealthCategory>>,
      TError,
      CheckHealthCategoryMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof checkHealthCategory>>,
  TError,
  CheckHealthCategoryMutationVariables,
  TContext
> => {
  return useMutation(getCheckHealthCategoryMutationOptions(options), queryClient);
};
export const getCheckHealthItemUrl = (category: string, key: string) => {
  return `/api/v1/system/health/${category}/items/${key}/check`;
};

/**
 * @summary Check one item now
 */
export const checkHealthItem = async (
  category: string,
  key: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<HealthChecked> => {
  return apiFetch<HealthChecked>(getCheckHealthItemUrl(category, key), {
    ...options,
    method: "POST",
  });
};

export const getCheckHealthItemMutationKey = () => ["checkHealthItem"] as const;

export const getCheckHealthItemMutationOptions = <
  TError = ErrorType<
    | CheckHealthItem401
    | CheckHealthItem403
    | CheckHealthItem404
    | CheckHealthItem409
    | CheckHealthItem422
    | CheckHealthItem500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof checkHealthItem>>,
    TError,
    CheckHealthItemMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof checkHealthItem>>,
  TError,
  CheckHealthItemMutationVariables,
  TContext
> => {
  const mutationKey = getCheckHealthItemMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof checkHealthItem>>,
    CheckHealthItemMutationVariables
  > = (props) => {
    const { category, key } = props ?? {};

    return checkHealthItem(category, key, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CheckHealthItemMutationResult = NonNullable<
  Awaited<ReturnType<typeof checkHealthItem>>
>;

export type CheckHealthItemMutationError = ErrorType<
  | CheckHealthItem401
  | CheckHealthItem403
  | CheckHealthItem404
  | CheckHealthItem409
  | CheckHealthItem422
  | CheckHealthItem500
>;
export type CheckHealthItemMutationVariables = { category: string; key: string };

/**
 * @summary Check one item now
 */
export const useCheckHealthItem = <
  TError = ErrorType<
    | CheckHealthItem401
    | CheckHealthItem403
    | CheckHealthItem404
    | CheckHealthItem409
    | CheckHealthItem422
    | CheckHealthItem500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof checkHealthItem>>,
      TError,
      CheckHealthItemMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof checkHealthItem>>,
  TError,
  CheckHealthItemMutationVariables,
  TContext
> => {
  return useMutation(getCheckHealthItemMutationOptions(options), queryClient);
};
