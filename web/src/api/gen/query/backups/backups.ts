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
  AddBackupDestination401,
  AddBackupDestination403,
  AddBackupDestination404,
  AddBackupDestination409,
  AddBackupDestination422,
  AddBackupDestination500,
  AddBackupDestination503,
  BackUpNow401,
  BackUpNow403,
  BackUpNow409,
  BackUpNow500,
  BackupSettingsInputBody,
  BackupsAdded,
  BackupsBackupList,
  BackupsBackupSettings,
  BackupsChecked,
  BackupsDestinationList,
  BackupsNewDestination,
  BackupsPassphrase,
  CheckBackupPassphrase401,
  CheckBackupPassphrase403,
  CheckBackupPassphrase409,
  CheckBackupPassphrase422,
  CheckBackupPassphrase500,
  CheckInputBody,
  DeleteBackup401,
  DeleteBackup403,
  DeleteBackup404,
  DeleteBackup409,
  DeleteBackup422,
  DeleteBackup500,
  GetBackupPassphrase401,
  GetBackupPassphrase403,
  GetBackupPassphrase409,
  GetBackupPassphrase500,
  GetBackupSettings401,
  GetBackupSettings403,
  GetBackupSettings409,
  GetBackupSettings500,
  ListBackupDestinations401,
  ListBackupDestinations403,
  ListBackupDestinations409,
  ListBackupDestinations500,
  ListBackups401,
  ListBackups403,
  ListBackups409,
  ListBackups500,
  PrepareBackupDownload401,
  PrepareBackupDownload403,
  PrepareBackupDownload404,
  PrepareBackupDownload409,
  PrepareBackupDownload422,
  PrepareBackupDownload500,
  PrepareRecoverySheet401,
  PrepareRecoverySheet403,
  PrepareRecoverySheet409,
  PrepareRecoverySheet500,
  PullBackup401,
  PullBackup403,
  PullBackup404,
  PullBackup409,
  PullBackup422,
  PullBackup500,
  PullInputBody,
  RemoveBackupDestination401,
  RemoveBackupDestination403,
  RemoveBackupDestination404,
  RemoveBackupDestination409,
  RemoveBackupDestination422,
  RemoveBackupDestination500,
  ReplaceBackupPassphrase401,
  ReplaceBackupPassphrase403,
  ReplaceBackupPassphrase409,
  ReplaceBackupPassphrase422,
  ReplaceBackupPassphrase500,
  ReplaceInputBody,
  RetireController401,
  RetireController403,
  RetireController409,
  RetireController500,
  SaveBackupPassphrase401,
  SaveBackupPassphrase403,
  SaveBackupPassphrase409,
  SaveBackupPassphrase500,
  UpdateBackupSettings401,
  UpdateBackupSettings403,
  UpdateBackupSettings409,
  UpdateBackupSettings422,
  UpdateBackupSettings500,
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

export const getListBackupDestinationsUrl = () => {
  return `/api/v1/backup-destinations`;
};

/**
 * @summary List the Backup destinations and their copies
 */
export const listBackupDestinations = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<BackupsDestinationList> => {
  return apiFetch<BackupsDestinationList>(getListBackupDestinationsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListBackupDestinationsQueryKey = () => {
  return [`/api/v1/backup-destinations`] as const;
};

export const getListBackupDestinationsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listBackupDestinations>>,
  TError = ErrorType<
    | ListBackupDestinations401
    | ListBackupDestinations403
    | ListBackupDestinations409
    | ListBackupDestinations500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackupDestinations>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListBackupDestinationsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listBackupDestinations>>> = ({ signal }) =>
    listBackupDestinations({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listBackupDestinations>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListBackupDestinationsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listBackupDestinations>>
>;
export type ListBackupDestinationsSuspenseQueryError = ErrorType<
  | ListBackupDestinations401
  | ListBackupDestinations403
  | ListBackupDestinations409
  | ListBackupDestinations500
>;

export function useListBackupDestinationsSuspense<
  TData = Awaited<ReturnType<typeof listBackupDestinations>>,
  TError = ErrorType<
    | ListBackupDestinations401
    | ListBackupDestinations403
    | ListBackupDestinations409
    | ListBackupDestinations500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackupDestinations>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListBackupDestinationsSuspense<
  TData = Awaited<ReturnType<typeof listBackupDestinations>>,
  TError = ErrorType<
    | ListBackupDestinations401
    | ListBackupDestinations403
    | ListBackupDestinations409
    | ListBackupDestinations500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackupDestinations>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListBackupDestinationsSuspense<
  TData = Awaited<ReturnType<typeof listBackupDestinations>>,
  TError = ErrorType<
    | ListBackupDestinations401
    | ListBackupDestinations403
    | ListBackupDestinations409
    | ListBackupDestinations500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackupDestinations>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Backup destinations and their copies
 */

export function useListBackupDestinationsSuspense<
  TData = Awaited<ReturnType<typeof listBackupDestinations>>,
  TError = ErrorType<
    | ListBackupDestinations401
    | ListBackupDestinations403
    | ListBackupDestinations409
    | ListBackupDestinations500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackupDestinations>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListBackupDestinationsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getAddBackupDestinationUrl = () => {
  return `/api/v1/backup-destinations`;
};

/**
 * @summary Make a folder on a Node a Backup destination, and copy every Backup there
 */
export const addBackupDestination = async (
  backupsNewDestination: BackupsNewDestination,
  options?: Parameters<typeof apiFetch>[1],
): Promise<BackupsAdded> => {
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
  return apiFetch<BackupsAdded>(getAddBackupDestinationUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(backupsNewDestination),
  });
};

export const getAddBackupDestinationMutationKey = () => ["addBackupDestination"] as const;

export const getAddBackupDestinationMutationOptions = <
  TError = ErrorType<
    | AddBackupDestination401
    | AddBackupDestination403
    | AddBackupDestination404
    | AddBackupDestination409
    | AddBackupDestination422
    | AddBackupDestination500
    | AddBackupDestination503
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof addBackupDestination>>,
    TError,
    AddBackupDestinationMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof addBackupDestination>>,
  TError,
  AddBackupDestinationMutationVariables,
  TContext
> => {
  const mutationKey = getAddBackupDestinationMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof addBackupDestination>>,
    AddBackupDestinationMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return addBackupDestination(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type AddBackupDestinationMutationResult = NonNullable<
  Awaited<ReturnType<typeof addBackupDestination>>
>;
export type AddBackupDestinationMutationBody = BackupsNewDestination;
export type AddBackupDestinationMutationError = ErrorType<
  | AddBackupDestination401
  | AddBackupDestination403
  | AddBackupDestination404
  | AddBackupDestination409
  | AddBackupDestination422
  | AddBackupDestination500
  | AddBackupDestination503
>;
export type AddBackupDestinationMutationVariables = { data: BackupsNewDestination };

/**
 * @summary Make a folder on a Node a Backup destination, and copy every Backup there
 */
export const useAddBackupDestination = <
  TError = ErrorType<
    | AddBackupDestination401
    | AddBackupDestination403
    | AddBackupDestination404
    | AddBackupDestination409
    | AddBackupDestination422
    | AddBackupDestination500
    | AddBackupDestination503
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof addBackupDestination>>,
      TError,
      AddBackupDestinationMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof addBackupDestination>>,
  TError,
  AddBackupDestinationMutationVariables,
  TContext
> => {
  return useMutation(getAddBackupDestinationMutationOptions(options), queryClient);
};
export const getRemoveBackupDestinationUrl = (destinationId: string) => {
  return `/api/v1/backup-destinations/${destinationId}`;
};

/**
 * @summary Stop copying Backups to a destination, leaving its copies in its folder
 */
export const removeBackupDestination = async (
  destinationId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getRemoveBackupDestinationUrl(destinationId), {
    ...options,
    method: "DELETE",
  });
};

export const getRemoveBackupDestinationMutationKey = () => ["removeBackupDestination"] as const;

export const getRemoveBackupDestinationMutationOptions = <
  TError = ErrorType<
    | RemoveBackupDestination401
    | RemoveBackupDestination403
    | RemoveBackupDestination404
    | RemoveBackupDestination409
    | RemoveBackupDestination422
    | RemoveBackupDestination500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof removeBackupDestination>>,
    TError,
    RemoveBackupDestinationMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof removeBackupDestination>>,
  TError,
  RemoveBackupDestinationMutationVariables,
  TContext
> => {
  const mutationKey = getRemoveBackupDestinationMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof removeBackupDestination>>,
    RemoveBackupDestinationMutationVariables
  > = (props) => {
    const { destinationId } = props ?? {};

    return removeBackupDestination(destinationId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RemoveBackupDestinationMutationResult = NonNullable<
  Awaited<ReturnType<typeof removeBackupDestination>>
>;

export type RemoveBackupDestinationMutationError = ErrorType<
  | RemoveBackupDestination401
  | RemoveBackupDestination403
  | RemoveBackupDestination404
  | RemoveBackupDestination409
  | RemoveBackupDestination422
  | RemoveBackupDestination500
>;
export type RemoveBackupDestinationMutationVariables = { destinationId: string };

/**
 * @summary Stop copying Backups to a destination, leaving its copies in its folder
 */
export const useRemoveBackupDestination = <
  TError = ErrorType<
    | RemoveBackupDestination401
    | RemoveBackupDestination403
    | RemoveBackupDestination404
    | RemoveBackupDestination409
    | RemoveBackupDestination422
    | RemoveBackupDestination500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof removeBackupDestination>>,
      TError,
      RemoveBackupDestinationMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof removeBackupDestination>>,
  TError,
  RemoveBackupDestinationMutationVariables,
  TContext
> => {
  return useMutation(getRemoveBackupDestinationMutationOptions(options), queryClient);
};
export const getGetBackupSettingsUrl = () => {
  return `/api/v1/backup-settings`;
};

/**
 * @summary Get the Backups settings
 */
export const getBackupSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<BackupsBackupSettings> => {
  return apiFetch<BackupsBackupSettings>(getGetBackupSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetBackupSettingsQueryKey = () => {
  return [`/api/v1/backup-settings`] as const;
};

export const getGetBackupSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getBackupSettings>>,
  TError = ErrorType<
    GetBackupSettings401 | GetBackupSettings403 | GetBackupSettings409 | GetBackupSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetBackupSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getBackupSettings>>> = ({ signal }) =>
    getBackupSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetBackupSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getBackupSettings>>
>;
export type GetBackupSettingsSuspenseQueryError = ErrorType<
  GetBackupSettings401 | GetBackupSettings403 | GetBackupSettings409 | GetBackupSettings500
>;

export function useGetBackupSettingsSuspense<
  TData = Awaited<ReturnType<typeof getBackupSettings>>,
  TError = ErrorType<
    GetBackupSettings401 | GetBackupSettings403 | GetBackupSettings409 | GetBackupSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetBackupSettingsSuspense<
  TData = Awaited<ReturnType<typeof getBackupSettings>>,
  TError = ErrorType<
    GetBackupSettings401 | GetBackupSettings403 | GetBackupSettings409 | GetBackupSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetBackupSettingsSuspense<
  TData = Awaited<ReturnType<typeof getBackupSettings>>,
  TError = ErrorType<
    GetBackupSettings401 | GetBackupSettings403 | GetBackupSettings409 | GetBackupSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Backups settings
 */

export function useGetBackupSettingsSuspense<
  TData = Awaited<ReturnType<typeof getBackupSettings>>,
  TError = ErrorType<
    GetBackupSettings401 | GetBackupSettings403 | GetBackupSettings409 | GetBackupSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetBackupSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateBackupSettingsUrl = () => {
  return `/api/v1/backup-settings`;
};

/**
 * @summary Change Backups settings by key, all or none
 */
export const updateBackupSettings = async (
  backupSettingsInputBody: BackupSettingsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<BackupsBackupSettings> => {
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
  return apiFetch<BackupsBackupSettings>(getUpdateBackupSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(backupSettingsInputBody),
  });
};

export const getUpdateBackupSettingsMutationKey = () => ["updateBackupSettings"] as const;

export const getUpdateBackupSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateBackupSettings401
    | UpdateBackupSettings403
    | UpdateBackupSettings409
    | UpdateBackupSettings422
    | UpdateBackupSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateBackupSettings>>,
    TError,
    UpdateBackupSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateBackupSettings>>,
  TError,
  UpdateBackupSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateBackupSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateBackupSettings>>,
    UpdateBackupSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateBackupSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateBackupSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateBackupSettings>>
>;
export type UpdateBackupSettingsMutationBody = BackupSettingsInputBody;
export type UpdateBackupSettingsMutationError = ErrorType<
  | UpdateBackupSettings401
  | UpdateBackupSettings403
  | UpdateBackupSettings409
  | UpdateBackupSettings422
  | UpdateBackupSettings500
>;
export type UpdateBackupSettingsMutationVariables = { data: BackupSettingsInputBody };

/**
 * @summary Change Backups settings by key, all or none
 */
export const useUpdateBackupSettings = <
  TError = ErrorType<
    | UpdateBackupSettings401
    | UpdateBackupSettings403
    | UpdateBackupSettings409
    | UpdateBackupSettings422
    | UpdateBackupSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateBackupSettings>>,
      TError,
      UpdateBackupSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateBackupSettings>>,
  TError,
  UpdateBackupSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateBackupSettingsMutationOptions(options), queryClient);
};
export const getListBackupsUrl = () => {
  return `/api/v1/backups`;
};

/**
 * @summary List the Backups, newest first, with their copies and the passphrase's state
 */
export const listBackups = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<BackupsBackupList> => {
  return apiFetch<BackupsBackupList>(getListBackupsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListBackupsQueryKey = () => {
  return [`/api/v1/backups`] as const;
};

export const getListBackupsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listBackups>>,
  TError = ErrorType<ListBackups401 | ListBackups403 | ListBackups409 | ListBackups500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackups>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListBackupsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listBackups>>> = ({ signal }) =>
    listBackups({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackups>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListBackupsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listBackups>>>;
export type ListBackupsSuspenseQueryError = ErrorType<
  ListBackups401 | ListBackups403 | ListBackups409 | ListBackups500
>;

export function useListBackupsSuspense<
  TData = Awaited<ReturnType<typeof listBackups>>,
  TError = ErrorType<ListBackups401 | ListBackups403 | ListBackups409 | ListBackups500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackups>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListBackupsSuspense<
  TData = Awaited<ReturnType<typeof listBackups>>,
  TError = ErrorType<ListBackups401 | ListBackups403 | ListBackups409 | ListBackups500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackups>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListBackupsSuspense<
  TData = Awaited<ReturnType<typeof listBackups>>,
  TError = ErrorType<ListBackups401 | ListBackups403 | ListBackups409 | ListBackups500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackups>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Backups, newest first, with their copies and the passphrase's state
 */

export function useListBackupsSuspense<
  TData = Awaited<ReturnType<typeof listBackups>>,
  TError = ErrorType<ListBackups401 | ListBackups403 | ListBackups409 | ListBackups500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackups>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListBackupsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getBackUpNowUrl = () => {
  return `/api/v1/backups`;
};

/**
 * @summary Start a Backup now. The Backup task's run shows how it went
 */
export const backUpNow = async (options?: Parameters<typeof apiFetch>[1]): Promise<void> => {
  return apiFetch<void>(getBackUpNowUrl(), {
    ...options,
    method: "POST",
  });
};

export const getBackUpNowMutationKey = () => ["backUpNow"] as const;

export const getBackUpNowMutationOptions = <
  TError = ErrorType<BackUpNow401 | BackUpNow403 | BackUpNow409 | BackUpNow500>,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<Awaited<ReturnType<typeof backUpNow>>, TError, void, TContext>;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof backUpNow>>, TError, void, TContext> => {
  const mutationKey = getBackUpNowMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof backUpNow>>, void> = () => {
    return backUpNow(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type BackUpNowMutationResult = NonNullable<Awaited<ReturnType<typeof backUpNow>>>;

export type BackUpNowMutationError = ErrorType<
  BackUpNow401 | BackUpNow403 | BackUpNow409 | BackUpNow500
>;

/**
 * @summary Start a Backup now. The Backup task's run shows how it went
 */
export const useBackUpNow = <
  TError = ErrorType<BackUpNow401 | BackUpNow403 | BackUpNow409 | BackUpNow500>,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<Awaited<ReturnType<typeof backUpNow>>, TError, void, TContext>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof backUpNow>>, TError, void, TContext> => {
  return useMutation(getBackUpNowMutationOptions(options), queryClient);
};
export const getGetBackupPassphraseUrl = () => {
  return `/api/v1/backups/passphrase`;
};

/**
 * @summary Show the passphrase that opens every new Backup
 */
export const getBackupPassphrase = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<BackupsPassphrase> => {
  return apiFetch<BackupsPassphrase>(getGetBackupPassphraseUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetBackupPassphraseQueryKey = () => {
  return [`/api/v1/backups/passphrase`] as const;
};

export const getGetBackupPassphraseSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getBackupPassphrase>>,
  TError = ErrorType<
    | GetBackupPassphrase401
    | GetBackupPassphrase403
    | GetBackupPassphrase409
    | GetBackupPassphrase500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupPassphrase>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetBackupPassphraseQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getBackupPassphrase>>> = ({ signal }) =>
    getBackupPassphrase({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupPassphrase>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetBackupPassphraseSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getBackupPassphrase>>
>;
export type GetBackupPassphraseSuspenseQueryError = ErrorType<
  GetBackupPassphrase401 | GetBackupPassphrase403 | GetBackupPassphrase409 | GetBackupPassphrase500
>;

export function useGetBackupPassphraseSuspense<
  TData = Awaited<ReturnType<typeof getBackupPassphrase>>,
  TError = ErrorType<
    | GetBackupPassphrase401
    | GetBackupPassphrase403
    | GetBackupPassphrase409
    | GetBackupPassphrase500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupPassphrase>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetBackupPassphraseSuspense<
  TData = Awaited<ReturnType<typeof getBackupPassphrase>>,
  TError = ErrorType<
    | GetBackupPassphrase401
    | GetBackupPassphrase403
    | GetBackupPassphrase409
    | GetBackupPassphrase500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupPassphrase>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetBackupPassphraseSuspense<
  TData = Awaited<ReturnType<typeof getBackupPassphrase>>,
  TError = ErrorType<
    | GetBackupPassphrase401
    | GetBackupPassphrase403
    | GetBackupPassphrase409
    | GetBackupPassphrase500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupPassphrase>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Show the passphrase that opens every new Backup
 */

export function useGetBackupPassphraseSuspense<
  TData = Awaited<ReturnType<typeof getBackupPassphrase>>,
  TError = ErrorType<
    | GetBackupPassphrase401
    | GetBackupPassphrase403
    | GetBackupPassphrase409
    | GetBackupPassphrase500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getBackupPassphrase>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetBackupPassphraseSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getReplaceBackupPassphraseUrl = () => {
  return `/api/v1/backups/passphrase`;
};

/**
 * @summary Make a new Backup passphrase and start a Backup with it
 */
export const replaceBackupPassphrase = async (
  replaceInputBody: ReplaceInputBody,
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
  return apiFetch<void>(getReplaceBackupPassphraseUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(replaceInputBody),
  });
};

export const getReplaceBackupPassphraseMutationKey = () => ["replaceBackupPassphrase"] as const;

export const getReplaceBackupPassphraseMutationOptions = <
  TError = ErrorType<
    | ReplaceBackupPassphrase401
    | ReplaceBackupPassphrase403
    | ReplaceBackupPassphrase409
    | ReplaceBackupPassphrase422
    | ReplaceBackupPassphrase500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof replaceBackupPassphrase>>,
    TError,
    ReplaceBackupPassphraseMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof replaceBackupPassphrase>>,
  TError,
  ReplaceBackupPassphraseMutationVariables,
  TContext
> => {
  const mutationKey = getReplaceBackupPassphraseMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof replaceBackupPassphrase>>,
    ReplaceBackupPassphraseMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return replaceBackupPassphrase(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ReplaceBackupPassphraseMutationResult = NonNullable<
  Awaited<ReturnType<typeof replaceBackupPassphrase>>
>;
export type ReplaceBackupPassphraseMutationBody = ReplaceInputBody;
export type ReplaceBackupPassphraseMutationError = ErrorType<
  | ReplaceBackupPassphrase401
  | ReplaceBackupPassphrase403
  | ReplaceBackupPassphrase409
  | ReplaceBackupPassphrase422
  | ReplaceBackupPassphrase500
>;
export type ReplaceBackupPassphraseMutationVariables = { data: ReplaceInputBody };

/**
 * @summary Make a new Backup passphrase and start a Backup with it
 */
export const useReplaceBackupPassphrase = <
  TError = ErrorType<
    | ReplaceBackupPassphrase401
    | ReplaceBackupPassphrase403
    | ReplaceBackupPassphrase409
    | ReplaceBackupPassphrase422
    | ReplaceBackupPassphrase500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof replaceBackupPassphrase>>,
      TError,
      ReplaceBackupPassphraseMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof replaceBackupPassphrase>>,
  TError,
  ReplaceBackupPassphraseMutationVariables,
  TContext
> => {
  return useMutation(getReplaceBackupPassphraseMutationOptions(options), queryClient);
};
export const getCheckBackupPassphraseUrl = () => {
  return `/api/v1/backups/passphrase/check`;
};

/**
 * @summary Check a saved copy against the current Backup passphrase
 */
export const checkBackupPassphrase = async (
  checkInputBody: CheckInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<BackupsChecked> => {
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
  return apiFetch<BackupsChecked>(getCheckBackupPassphraseUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(checkInputBody),
  });
};

export const getCheckBackupPassphraseMutationKey = () => ["checkBackupPassphrase"] as const;

export const getCheckBackupPassphraseMutationOptions = <
  TError = ErrorType<
    | CheckBackupPassphrase401
    | CheckBackupPassphrase403
    | CheckBackupPassphrase409
    | CheckBackupPassphrase422
    | CheckBackupPassphrase500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof checkBackupPassphrase>>,
    TError,
    CheckBackupPassphraseMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof checkBackupPassphrase>>,
  TError,
  CheckBackupPassphraseMutationVariables,
  TContext
> => {
  const mutationKey = getCheckBackupPassphraseMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof checkBackupPassphrase>>,
    CheckBackupPassphraseMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return checkBackupPassphrase(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CheckBackupPassphraseMutationResult = NonNullable<
  Awaited<ReturnType<typeof checkBackupPassphrase>>
>;
export type CheckBackupPassphraseMutationBody = CheckInputBody;
export type CheckBackupPassphraseMutationError = ErrorType<
  | CheckBackupPassphrase401
  | CheckBackupPassphrase403
  | CheckBackupPassphrase409
  | CheckBackupPassphrase422
  | CheckBackupPassphrase500
>;
export type CheckBackupPassphraseMutationVariables = { data: CheckInputBody };

/**
 * @summary Check a saved copy against the current Backup passphrase
 */
export const useCheckBackupPassphrase = <
  TError = ErrorType<
    | CheckBackupPassphrase401
    | CheckBackupPassphrase403
    | CheckBackupPassphrase409
    | CheckBackupPassphrase422
    | CheckBackupPassphrase500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof checkBackupPassphrase>>,
      TError,
      CheckBackupPassphraseMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof checkBackupPassphrase>>,
  TError,
  CheckBackupPassphraseMutationVariables,
  TContext
> => {
  return useMutation(getCheckBackupPassphraseMutationOptions(options), queryClient);
};
export const getSaveBackupPassphraseUrl = () => {
  return `/api/v1/backups/passphrase/saved`;
};

/**
 * @summary Record that you saved the current Backup passphrase
 */
export const saveBackupPassphrase = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getSaveBackupPassphraseUrl(), {
    ...options,
    method: "POST",
  });
};

export const getSaveBackupPassphraseMutationKey = () => ["saveBackupPassphrase"] as const;

export const getSaveBackupPassphraseMutationOptions = <
  TError = ErrorType<
    | SaveBackupPassphrase401
    | SaveBackupPassphrase403
    | SaveBackupPassphrase409
    | SaveBackupPassphrase500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof saveBackupPassphrase>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof saveBackupPassphrase>>,
  TError,
  void,
  TContext
> => {
  const mutationKey = getSaveBackupPassphraseMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof saveBackupPassphrase>>,
    void
  > = () => {
    return saveBackupPassphrase(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SaveBackupPassphraseMutationResult = NonNullable<
  Awaited<ReturnType<typeof saveBackupPassphrase>>
>;

export type SaveBackupPassphraseMutationError = ErrorType<
  | SaveBackupPassphrase401
  | SaveBackupPassphrase403
  | SaveBackupPassphrase409
  | SaveBackupPassphrase500
>;

/**
 * @summary Record that you saved the current Backup passphrase
 */
export const useSaveBackupPassphrase = <
  TError = ErrorType<
    | SaveBackupPassphrase401
    | SaveBackupPassphrase403
    | SaveBackupPassphrase409
    | SaveBackupPassphrase500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof saveBackupPassphrase>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof saveBackupPassphrase>>, TError, void, TContext> => {
  return useMutation(getSaveBackupPassphraseMutationOptions(options), queryClient);
};
export const getPrepareRecoverySheetUrl = () => {
  return `/api/v1/backups/recovery-sheet/ticket`;
};

/**
 * @summary Confirm it's you before downloading the recovery sheet
 */
export const prepareRecoverySheet = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getPrepareRecoverySheetUrl(), {
    ...options,
    method: "POST",
  });
};

export const getPrepareRecoverySheetMutationKey = () => ["prepareRecoverySheet"] as const;

export const getPrepareRecoverySheetMutationOptions = <
  TError = ErrorType<
    | PrepareRecoverySheet401
    | PrepareRecoverySheet403
    | PrepareRecoverySheet409
    | PrepareRecoverySheet500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof prepareRecoverySheet>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof prepareRecoverySheet>>,
  TError,
  void,
  TContext
> => {
  const mutationKey = getPrepareRecoverySheetMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof prepareRecoverySheet>>,
    void
  > = () => {
    return prepareRecoverySheet(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PrepareRecoverySheetMutationResult = NonNullable<
  Awaited<ReturnType<typeof prepareRecoverySheet>>
>;

export type PrepareRecoverySheetMutationError = ErrorType<
  | PrepareRecoverySheet401
  | PrepareRecoverySheet403
  | PrepareRecoverySheet409
  | PrepareRecoverySheet500
>;

/**
 * @summary Confirm it's you before downloading the recovery sheet
 */
export const usePrepareRecoverySheet = <
  TError = ErrorType<
    | PrepareRecoverySheet401
    | PrepareRecoverySheet403
    | PrepareRecoverySheet409
    | PrepareRecoverySheet500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof prepareRecoverySheet>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof prepareRecoverySheet>>, TError, void, TContext> => {
  return useMutation(getPrepareRecoverySheetMutationOptions(options), queryClient);
};
export const getRetireControllerUrl = () => {
  return `/api/v1/backups/retire`;
};

/**
 * @summary Retire this Controller after its library was restored on another machine
 */
export const retireController = async (options?: Parameters<typeof apiFetch>[1]): Promise<void> => {
  return apiFetch<void>(getRetireControllerUrl(), {
    ...options,
    method: "POST",
  });
};

export const getRetireControllerMutationKey = () => ["retireController"] as const;

export const getRetireControllerMutationOptions = <
  TError = ErrorType<
    RetireController401 | RetireController403 | RetireController409 | RetireController500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof retireController>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof retireController>>, TError, void, TContext> => {
  const mutationKey = getRetireControllerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof retireController>>, void> = () => {
    return retireController(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RetireControllerMutationResult = NonNullable<
  Awaited<ReturnType<typeof retireController>>
>;

export type RetireControllerMutationError = ErrorType<
  RetireController401 | RetireController403 | RetireController409 | RetireController500
>;

/**
 * @summary Retire this Controller after its library was restored on another machine
 */
export const useRetireController = <
  TError = ErrorType<
    RetireController401 | RetireController403 | RetireController409 | RetireController500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof retireController>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof retireController>>, TError, void, TContext> => {
  return useMutation(getRetireControllerMutationOptions(options), queryClient);
};
export const getDeleteBackupUrl = (backupId: string) => {
  return `/api/v1/backups/${backupId}`;
};

/**
 * @summary Delete a Backup and its copies, and list the Backups left
 */
export const deleteBackup = async (
  backupId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<BackupsBackupList> => {
  return apiFetch<BackupsBackupList>(getDeleteBackupUrl(backupId), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteBackupMutationKey = () => ["deleteBackup"] as const;

export const getDeleteBackupMutationOptions = <
  TError = ErrorType<
    | DeleteBackup401
    | DeleteBackup403
    | DeleteBackup404
    | DeleteBackup409
    | DeleteBackup422
    | DeleteBackup500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteBackup>>,
    TError,
    DeleteBackupMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteBackup>>,
  TError,
  DeleteBackupMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteBackupMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteBackup>>,
    DeleteBackupMutationVariables
  > = (props) => {
    const { backupId } = props ?? {};

    return deleteBackup(backupId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteBackupMutationResult = NonNullable<Awaited<ReturnType<typeof deleteBackup>>>;

export type DeleteBackupMutationError = ErrorType<
  | DeleteBackup401
  | DeleteBackup403
  | DeleteBackup404
  | DeleteBackup409
  | DeleteBackup422
  | DeleteBackup500
>;
export type DeleteBackupMutationVariables = { backupId: string };

/**
 * @summary Delete a Backup and its copies, and list the Backups left
 */
export const useDeleteBackup = <
  TError = ErrorType<
    | DeleteBackup401
    | DeleteBackup403
    | DeleteBackup404
    | DeleteBackup409
    | DeleteBackup422
    | DeleteBackup500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteBackup>>,
      TError,
      DeleteBackupMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteBackup>>,
  TError,
  DeleteBackupMutationVariables,
  TContext
> => {
  return useMutation(getDeleteBackupMutationOptions(options), queryClient);
};
export const getPrepareBackupDownloadUrl = (backupId: string) => {
  return `/api/v1/backups/${backupId}/file/ticket`;
};

/**
 * @summary Confirm it's you before downloading a Backup's file
 */
export const prepareBackupDownload = async (
  backupId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getPrepareBackupDownloadUrl(backupId), {
    ...options,
    method: "POST",
  });
};

export const getPrepareBackupDownloadMutationKey = () => ["prepareBackupDownload"] as const;

export const getPrepareBackupDownloadMutationOptions = <
  TError = ErrorType<
    | PrepareBackupDownload401
    | PrepareBackupDownload403
    | PrepareBackupDownload404
    | PrepareBackupDownload409
    | PrepareBackupDownload422
    | PrepareBackupDownload500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof prepareBackupDownload>>,
    TError,
    PrepareBackupDownloadMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof prepareBackupDownload>>,
  TError,
  PrepareBackupDownloadMutationVariables,
  TContext
> => {
  const mutationKey = getPrepareBackupDownloadMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof prepareBackupDownload>>,
    PrepareBackupDownloadMutationVariables
  > = (props) => {
    const { backupId } = props ?? {};

    return prepareBackupDownload(backupId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PrepareBackupDownloadMutationResult = NonNullable<
  Awaited<ReturnType<typeof prepareBackupDownload>>
>;

export type PrepareBackupDownloadMutationError = ErrorType<
  | PrepareBackupDownload401
  | PrepareBackupDownload403
  | PrepareBackupDownload404
  | PrepareBackupDownload409
  | PrepareBackupDownload422
  | PrepareBackupDownload500
>;
export type PrepareBackupDownloadMutationVariables = { backupId: string };

/**
 * @summary Confirm it's you before downloading a Backup's file
 */
export const usePrepareBackupDownload = <
  TError = ErrorType<
    | PrepareBackupDownload401
    | PrepareBackupDownload403
    | PrepareBackupDownload404
    | PrepareBackupDownload409
    | PrepareBackupDownload422
    | PrepareBackupDownload500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof prepareBackupDownload>>,
      TError,
      PrepareBackupDownloadMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof prepareBackupDownload>>,
  TError,
  PrepareBackupDownloadMutationVariables,
  TContext
> => {
  return useMutation(getPrepareBackupDownloadMutationOptions(options), queryClient);
};
export const getPullBackupUrl = (backupId: string) => {
  return `/api/v1/backups/${backupId}/pull`;
};

/**
 * @summary Copy a Backup back from a destination into this machine's Backups folder
 */
export const pullBackup = async (
  backupId: string,
  pullInputBody: PullInputBody,
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
  return apiFetch<void>(getPullBackupUrl(backupId), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(pullInputBody),
  });
};

export const getPullBackupMutationKey = () => ["pullBackup"] as const;

export const getPullBackupMutationOptions = <
  TError = ErrorType<
    PullBackup401 | PullBackup403 | PullBackup404 | PullBackup409 | PullBackup422 | PullBackup500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof pullBackup>>,
    TError,
    PullBackupMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof pullBackup>>,
  TError,
  PullBackupMutationVariables,
  TContext
> => {
  const mutationKey = getPullBackupMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof pullBackup>>,
    PullBackupMutationVariables
  > = (props) => {
    const { backupId, data } = props ?? {};

    return pullBackup(backupId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PullBackupMutationResult = NonNullable<Awaited<ReturnType<typeof pullBackup>>>;
export type PullBackupMutationBody = PullInputBody;
export type PullBackupMutationError = ErrorType<
  PullBackup401 | PullBackup403 | PullBackup404 | PullBackup409 | PullBackup422 | PullBackup500
>;
export type PullBackupMutationVariables = { backupId: string; data: PullInputBody };

/**
 * @summary Copy a Backup back from a destination into this machine's Backups folder
 */
export const usePullBackup = <
  TError = ErrorType<
    PullBackup401 | PullBackup403 | PullBackup404 | PullBackup409 | PullBackup422 | PullBackup500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof pullBackup>>,
      TError,
      PullBackupMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof pullBackup>>,
  TError,
  PullBackupMutationVariables,
  TContext
> => {
  return useMutation(getPullBackupMutationOptions(options), queryClient);
};
