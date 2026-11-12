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
  ActOnImportFile401,
  ActOnImportFile403,
  ActOnImportFile404,
  ActOnImportFile409,
  ActOnImportFile422,
  ActOnImportFile500,
  ActOnImportFile501,
  ActOnRootImport401,
  ActOnRootImport403,
  ActOnRootImport404,
  ActOnRootImport409,
  ActOnRootImport422,
  ActOnRootImport500,
  ActOnRootImport502,
  AddRootImports401,
  AddRootImports403,
  AddRootImports409,
  AddRootImports422,
  AddRootImports500,
  ConfirmImports401,
  ConfirmImports403,
  ConfirmImports404,
  ConfirmImports409,
  ConfirmImports422,
  ConfirmImports500,
  GetExtensionImpact401,
  GetExtensionImpact403,
  GetExtensionImpact409,
  GetExtensionImpact422,
  GetExtensionImpact500,
  GetExtensionImpactParams,
  GetImport401,
  GetImport403,
  GetImport404,
  GetImport409,
  GetImport422,
  GetImport500,
  GetImportingSettings401,
  GetImportingSettings403,
  GetImportingSettings409,
  GetImportingSettings500,
  GetLibraryScanSettings401,
  GetLibraryScanSettings403,
  GetLibraryScanSettings409,
  GetLibraryScanSettings500,
  IgnoreImports401,
  IgnoreImports403,
  IgnoreImports409,
  IgnoreImports422,
  IgnoreImports500,
  ImportFolder401,
  ImportFolder403,
  ImportFolder409,
  ImportFolder422,
  ImportFolder500,
  ImportFolderInputBody,
  ImportIDsInputBody,
  ImportsAddedList,
  ImportsExtensionImpact,
  ImportsFileAction,
  ImportsImportEntry,
  ImportsImportingSettings,
  ImportsImportsList,
  ImportsList,
  ImportsRootAction,
  ImportsSettings,
  LibraryScanSettingsInputBody,
  ListImports401,
  ListImports403,
  ListImports409,
  ListImports422,
  ListImports500,
  ListImportsParams,
  MoveFilesInputBody,
  SetImportMoveFiles401,
  SetImportMoveFiles403,
  SetImportMoveFiles404,
  SetImportMoveFiles409,
  SetImportMoveFiles422,
  SetImportMoveFiles500,
  UnignoreImports401,
  UnignoreImports403,
  UnignoreImports409,
  UnignoreImports422,
  UnignoreImports500,
  UpdateImportingSettings401,
  UpdateImportingSettings403,
  UpdateImportingSettings409,
  UpdateImportingSettings422,
  UpdateImportingSettings500,
  UpdateLibraryScanSettings401,
  UpdateLibraryScanSettings403,
  UpdateLibraryScanSettings409,
  UpdateLibraryScanSettings422,
  UpdateLibraryScanSettings500,
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

export const getListImportsUrl = (params?: ListImportsParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/imports?${stringifiedParams}` : `/api/v1/imports`;
};

/**
 * @summary List the unfinished Imports, with what each root's last library scan added
 */
export const listImports = async (
  params?: ListImportsParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsList> => {
  return apiFetch<ImportsList>(getListImportsUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListImportsQueryKey = (params?: ListImportsParams) => {
  return [`/api/v1/imports`, ...(params ? [params] : [])] as const;
};

export const getListImportsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listImports>>,
  TError = ErrorType<
    ListImports401 | ListImports403 | ListImports409 | ListImports422 | ListImports500
  >,
>(
  params?: ListImportsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listImports>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListImportsQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listImports>>> = ({ signal }) =>
    listImports(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listImports>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListImportsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listImports>>>;
export type ListImportsSuspenseQueryError = ErrorType<
  ListImports401 | ListImports403 | ListImports409 | ListImports422 | ListImports500
>;

export function useListImportsSuspense<
  TData = Awaited<ReturnType<typeof listImports>>,
  TError = ErrorType<
    ListImports401 | ListImports403 | ListImports409 | ListImports422 | ListImports500
  >,
>(
  params: undefined | ListImportsParams,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listImports>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListImportsSuspense<
  TData = Awaited<ReturnType<typeof listImports>>,
  TError = ErrorType<
    ListImports401 | ListImports403 | ListImports409 | ListImports422 | ListImports500
  >,
>(
  params?: ListImportsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listImports>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListImportsSuspense<
  TData = Awaited<ReturnType<typeof listImports>>,
  TError = ErrorType<
    ListImports401 | ListImports403 | ListImports409 | ListImports422 | ListImports500
  >,
>(
  params?: ListImportsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listImports>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the unfinished Imports, with what each root's last library scan added
 */

export function useListImportsSuspense<
  TData = Awaited<ReturnType<typeof listImports>>,
  TError = ErrorType<
    ListImports401 | ListImports403 | ListImports409 | ListImports422 | ListImports500
  >,
>(
  params?: ListImportsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listImports>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListImportsSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getImportFolderUrl = () => {
  return `/api/v1/imports`;
};

/**
 * @summary Open the Imports of a folder on a Node, one per Work and Edition its files match
 */
export const importFolder = async (
  importFolderInputBody: ImportFolderInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsImportsList> => {
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
  return apiFetch<ImportsImportsList>(getImportFolderUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(importFolderInputBody),
  });
};

export const getImportFolderMutationKey = () => ["importFolder"] as const;

export const getImportFolderMutationOptions = <
  TError = ErrorType<
    ImportFolder401 | ImportFolder403 | ImportFolder409 | ImportFolder422 | ImportFolder500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof importFolder>>,
    TError,
    ImportFolderMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof importFolder>>,
  TError,
  ImportFolderMutationVariables,
  TContext
> => {
  const mutationKey = getImportFolderMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof importFolder>>,
    ImportFolderMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return importFolder(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ImportFolderMutationResult = NonNullable<Awaited<ReturnType<typeof importFolder>>>;
export type ImportFolderMutationBody = ImportFolderInputBody;
export type ImportFolderMutationError = ErrorType<
  ImportFolder401 | ImportFolder403 | ImportFolder409 | ImportFolder422 | ImportFolder500
>;
export type ImportFolderMutationVariables = { data: ImportFolderInputBody };

/**
 * @summary Open the Imports of a folder on a Node, one per Work and Edition its files match
 */
export const useImportFolder = <
  TError = ErrorType<
    ImportFolder401 | ImportFolder403 | ImportFolder409 | ImportFolder422 | ImportFolder500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof importFolder>>,
      TError,
      ImportFolderMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof importFolder>>,
  TError,
  ImportFolderMutationVariables,
  TContext
> => {
  return useMutation(getImportFolderMutationOptions(options), queryClient);
};
export const getAddRootImportsUrl = () => {
  return `/api/v1/imports/add`;
};

/**
 * @summary Add each root Import's suggested Work to its root and attach its files
 */
export const addRootImports = async (
  importIDsInputBody: ImportIDsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsAddedList> => {
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
  return apiFetch<ImportsAddedList>(getAddRootImportsUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(importIDsInputBody),
  });
};

export const getAddRootImportsMutationKey = () => ["addRootImports"] as const;

export const getAddRootImportsMutationOptions = <
  TError = ErrorType<
    | AddRootImports401
    | AddRootImports403
    | AddRootImports409
    | AddRootImports422
    | AddRootImports500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof addRootImports>>,
    TError,
    AddRootImportsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof addRootImports>>,
  TError,
  AddRootImportsMutationVariables,
  TContext
> => {
  const mutationKey = getAddRootImportsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof addRootImports>>,
    AddRootImportsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return addRootImports(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type AddRootImportsMutationResult = NonNullable<Awaited<ReturnType<typeof addRootImports>>>;
export type AddRootImportsMutationBody = ImportIDsInputBody;
export type AddRootImportsMutationError = ErrorType<
  AddRootImports401 | AddRootImports403 | AddRootImports409 | AddRootImports422 | AddRootImports500
>;
export type AddRootImportsMutationVariables = { data: ImportIDsInputBody };

/**
 * @summary Add each root Import's suggested Work to its root and attach its files
 */
export const useAddRootImports = <
  TError = ErrorType<
    | AddRootImports401
    | AddRootImports403
    | AddRootImports409
    | AddRootImports422
    | AddRootImports500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof addRootImports>>,
      TError,
      AddRootImportsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof addRootImports>>,
  TError,
  AddRootImportsMutationVariables,
  TContext
> => {
  return useMutation(getAddRootImportsMutationOptions(options), queryClient);
};
export const getConfirmImportsUrl = () => {
  return `/api/v1/imports/confirm`;
};

/**
 * @summary Import every file of these Imports that waits for confirmation
 */
export const confirmImports = async (
  importIDsInputBody: ImportIDsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsImportsList> => {
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
  return apiFetch<ImportsImportsList>(getConfirmImportsUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(importIDsInputBody),
  });
};

export const getConfirmImportsMutationKey = () => ["confirmImports"] as const;

export const getConfirmImportsMutationOptions = <
  TError = ErrorType<
    | ConfirmImports401
    | ConfirmImports403
    | ConfirmImports404
    | ConfirmImports409
    | ConfirmImports422
    | ConfirmImports500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof confirmImports>>,
    TError,
    ConfirmImportsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof confirmImports>>,
  TError,
  ConfirmImportsMutationVariables,
  TContext
> => {
  const mutationKey = getConfirmImportsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof confirmImports>>,
    ConfirmImportsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return confirmImports(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ConfirmImportsMutationResult = NonNullable<Awaited<ReturnType<typeof confirmImports>>>;
export type ConfirmImportsMutationBody = ImportIDsInputBody;
export type ConfirmImportsMutationError = ErrorType<
  | ConfirmImports401
  | ConfirmImports403
  | ConfirmImports404
  | ConfirmImports409
  | ConfirmImports422
  | ConfirmImports500
>;
export type ConfirmImportsMutationVariables = { data: ImportIDsInputBody };

/**
 * @summary Import every file of these Imports that waits for confirmation
 */
export const useConfirmImports = <
  TError = ErrorType<
    | ConfirmImports401
    | ConfirmImports403
    | ConfirmImports404
    | ConfirmImports409
    | ConfirmImports422
    | ConfirmImports500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof confirmImports>>,
      TError,
      ConfirmImportsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof confirmImports>>,
  TError,
  ConfirmImportsMutationVariables,
  TContext
> => {
  return useMutation(getConfirmImportsMutationOptions(options), queryClient);
};
export const getIgnoreImportsUrl = () => {
  return `/api/v1/imports/ignore`;
};

/**
 * @summary Ignore Imports until their files change
 */
export const ignoreImports = async (
  importIDsInputBody: ImportIDsInputBody,
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
  return apiFetch<void>(getIgnoreImportsUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(importIDsInputBody),
  });
};

export const getIgnoreImportsMutationKey = () => ["ignoreImports"] as const;

export const getIgnoreImportsMutationOptions = <
  TError = ErrorType<
    IgnoreImports401 | IgnoreImports403 | IgnoreImports409 | IgnoreImports422 | IgnoreImports500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof ignoreImports>>,
    TError,
    IgnoreImportsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof ignoreImports>>,
  TError,
  IgnoreImportsMutationVariables,
  TContext
> => {
  const mutationKey = getIgnoreImportsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof ignoreImports>>,
    IgnoreImportsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return ignoreImports(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type IgnoreImportsMutationResult = NonNullable<Awaited<ReturnType<typeof ignoreImports>>>;
export type IgnoreImportsMutationBody = ImportIDsInputBody;
export type IgnoreImportsMutationError = ErrorType<
  IgnoreImports401 | IgnoreImports403 | IgnoreImports409 | IgnoreImports422 | IgnoreImports500
>;
export type IgnoreImportsMutationVariables = { data: ImportIDsInputBody };

/**
 * @summary Ignore Imports until their files change
 */
export const useIgnoreImports = <
  TError = ErrorType<
    IgnoreImports401 | IgnoreImports403 | IgnoreImports409 | IgnoreImports422 | IgnoreImports500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof ignoreImports>>,
      TError,
      IgnoreImportsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof ignoreImports>>,
  TError,
  IgnoreImportsMutationVariables,
  TContext
> => {
  return useMutation(getIgnoreImportsMutationOptions(options), queryClient);
};
export const getGetImportingSettingsUrl = () => {
  return `/api/v1/imports/importing-settings`;
};

/**
 * @summary Get the Importing settings
 */
export const getImportingSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsImportingSettings> => {
  return apiFetch<ImportsImportingSettings>(getGetImportingSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetImportingSettingsQueryKey = () => {
  return [`/api/v1/imports/importing-settings`] as const;
};

export const getGetImportingSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getImportingSettings>>,
  TError = ErrorType<
    | GetImportingSettings401
    | GetImportingSettings403
    | GetImportingSettings409
    | GetImportingSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImportingSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetImportingSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getImportingSettings>>> = ({ signal }) =>
    getImportingSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImportingSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetImportingSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getImportingSettings>>
>;
export type GetImportingSettingsSuspenseQueryError = ErrorType<
  | GetImportingSettings401
  | GetImportingSettings403
  | GetImportingSettings409
  | GetImportingSettings500
>;

export function useGetImportingSettingsSuspense<
  TData = Awaited<ReturnType<typeof getImportingSettings>>,
  TError = ErrorType<
    | GetImportingSettings401
    | GetImportingSettings403
    | GetImportingSettings409
    | GetImportingSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImportingSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetImportingSettingsSuspense<
  TData = Awaited<ReturnType<typeof getImportingSettings>>,
  TError = ErrorType<
    | GetImportingSettings401
    | GetImportingSettings403
    | GetImportingSettings409
    | GetImportingSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImportingSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetImportingSettingsSuspense<
  TData = Awaited<ReturnType<typeof getImportingSettings>>,
  TError = ErrorType<
    | GetImportingSettings401
    | GetImportingSettings403
    | GetImportingSettings409
    | GetImportingSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImportingSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Importing settings
 */

export function useGetImportingSettingsSuspense<
  TData = Awaited<ReturnType<typeof getImportingSettings>>,
  TError = ErrorType<
    | GetImportingSettings401
    | GetImportingSettings403
    | GetImportingSettings409
    | GetImportingSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImportingSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetImportingSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateImportingSettingsUrl = () => {
  return `/api/v1/imports/importing-settings`;
};

/**
 * @summary Change the Importing settings
 */
export const updateImportingSettings = async (
  libraryScanSettingsInputBody: LibraryScanSettingsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsImportingSettings> => {
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
  return apiFetch<ImportsImportingSettings>(getUpdateImportingSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(libraryScanSettingsInputBody),
  });
};

export const getUpdateImportingSettingsMutationKey = () => ["updateImportingSettings"] as const;

export const getUpdateImportingSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateImportingSettings401
    | UpdateImportingSettings403
    | UpdateImportingSettings409
    | UpdateImportingSettings422
    | UpdateImportingSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateImportingSettings>>,
    TError,
    UpdateImportingSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateImportingSettings>>,
  TError,
  UpdateImportingSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateImportingSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateImportingSettings>>,
    UpdateImportingSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateImportingSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateImportingSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateImportingSettings>>
>;
export type UpdateImportingSettingsMutationBody = LibraryScanSettingsInputBody;
export type UpdateImportingSettingsMutationError = ErrorType<
  | UpdateImportingSettings401
  | UpdateImportingSettings403
  | UpdateImportingSettings409
  | UpdateImportingSettings422
  | UpdateImportingSettings500
>;
export type UpdateImportingSettingsMutationVariables = { data: LibraryScanSettingsInputBody };

/**
 * @summary Change the Importing settings
 */
export const useUpdateImportingSettings = <
  TError = ErrorType<
    | UpdateImportingSettings401
    | UpdateImportingSettings403
    | UpdateImportingSettings409
    | UpdateImportingSettings422
    | UpdateImportingSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateImportingSettings>>,
      TError,
      UpdateImportingSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateImportingSettings>>,
  TError,
  UpdateImportingSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateImportingSettingsMutationOptions(options), queryClient);
};
export const getGetExtensionImpactUrl = (params?: GetExtensionImpactParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/imports/importing-settings/extension-impact?${stringifiedParams}`
    : `/api/v1/imports/importing-settings/extension-impact`;
};

/**
 * @summary Count the library and pending files a video extension list stops counting as video
 */
export const getExtensionImpact = async (
  params?: GetExtensionImpactParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsExtensionImpact> => {
  return apiFetch<ImportsExtensionImpact>(getGetExtensionImpactUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getGetExtensionImpactQueryKey = (params?: GetExtensionImpactParams) => {
  return [
    `/api/v1/imports/importing-settings/extension-impact`,
    ...(params ? [params] : []),
  ] as const;
};

export const getGetExtensionImpactSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getExtensionImpact>>,
  TError = ErrorType<
    | GetExtensionImpact401
    | GetExtensionImpact403
    | GetExtensionImpact409
    | GetExtensionImpact422
    | GetExtensionImpact500
  >,
>(
  params?: GetExtensionImpactParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getExtensionImpact>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetExtensionImpactQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getExtensionImpact>>> = ({ signal }) =>
    getExtensionImpact(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getExtensionImpact>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetExtensionImpactSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getExtensionImpact>>
>;
export type GetExtensionImpactSuspenseQueryError = ErrorType<
  | GetExtensionImpact401
  | GetExtensionImpact403
  | GetExtensionImpact409
  | GetExtensionImpact422
  | GetExtensionImpact500
>;

export function useGetExtensionImpactSuspense<
  TData = Awaited<ReturnType<typeof getExtensionImpact>>,
  TError = ErrorType<
    | GetExtensionImpact401
    | GetExtensionImpact403
    | GetExtensionImpact409
    | GetExtensionImpact422
    | GetExtensionImpact500
  >,
>(
  params: undefined | GetExtensionImpactParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getExtensionImpact>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetExtensionImpactSuspense<
  TData = Awaited<ReturnType<typeof getExtensionImpact>>,
  TError = ErrorType<
    | GetExtensionImpact401
    | GetExtensionImpact403
    | GetExtensionImpact409
    | GetExtensionImpact422
    | GetExtensionImpact500
  >,
>(
  params?: GetExtensionImpactParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getExtensionImpact>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetExtensionImpactSuspense<
  TData = Awaited<ReturnType<typeof getExtensionImpact>>,
  TError = ErrorType<
    | GetExtensionImpact401
    | GetExtensionImpact403
    | GetExtensionImpact409
    | GetExtensionImpact422
    | GetExtensionImpact500
  >,
>(
  params?: GetExtensionImpactParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getExtensionImpact>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Count the library and pending files a video extension list stops counting as video
 */

export function useGetExtensionImpactSuspense<
  TData = Awaited<ReturnType<typeof getExtensionImpact>>,
  TError = ErrorType<
    | GetExtensionImpact401
    | GetExtensionImpact403
    | GetExtensionImpact409
    | GetExtensionImpact422
    | GetExtensionImpact500
  >,
>(
  params?: GetExtensionImpactParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getExtensionImpact>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetExtensionImpactSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetImportMoveFilesUrl = () => {
  return `/api/v1/imports/move-files`;
};

/**
 * @summary Turn Move files on or off for folder Imports
 */
export const setImportMoveFiles = async (
  moveFilesInputBody: MoveFilesInputBody,
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
  return apiFetch<void>(getSetImportMoveFilesUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(moveFilesInputBody),
  });
};

export const getSetImportMoveFilesMutationKey = () => ["setImportMoveFiles"] as const;

export const getSetImportMoveFilesMutationOptions = <
  TError = ErrorType<
    | SetImportMoveFiles401
    | SetImportMoveFiles403
    | SetImportMoveFiles404
    | SetImportMoveFiles409
    | SetImportMoveFiles422
    | SetImportMoveFiles500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setImportMoveFiles>>,
    TError,
    SetImportMoveFilesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setImportMoveFiles>>,
  TError,
  SetImportMoveFilesMutationVariables,
  TContext
> => {
  const mutationKey = getSetImportMoveFilesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setImportMoveFiles>>,
    SetImportMoveFilesMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return setImportMoveFiles(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetImportMoveFilesMutationResult = NonNullable<
  Awaited<ReturnType<typeof setImportMoveFiles>>
>;
export type SetImportMoveFilesMutationBody = MoveFilesInputBody;
export type SetImportMoveFilesMutationError = ErrorType<
  | SetImportMoveFiles401
  | SetImportMoveFiles403
  | SetImportMoveFiles404
  | SetImportMoveFiles409
  | SetImportMoveFiles422
  | SetImportMoveFiles500
>;
export type SetImportMoveFilesMutationVariables = { data: MoveFilesInputBody };

/**
 * @summary Turn Move files on or off for folder Imports
 */
export const useSetImportMoveFiles = <
  TError = ErrorType<
    | SetImportMoveFiles401
    | SetImportMoveFiles403
    | SetImportMoveFiles404
    | SetImportMoveFiles409
    | SetImportMoveFiles422
    | SetImportMoveFiles500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setImportMoveFiles>>,
      TError,
      SetImportMoveFilesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setImportMoveFiles>>,
  TError,
  SetImportMoveFilesMutationVariables,
  TContext
> => {
  return useMutation(getSetImportMoveFilesMutationOptions(options), queryClient);
};
export const getGetLibraryScanSettingsUrl = () => {
  return `/api/v1/imports/scan-settings`;
};

/**
 * @summary Get the Library scan settings
 */
export const getLibraryScanSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsSettings> => {
  return apiFetch<ImportsSettings>(getGetLibraryScanSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetLibraryScanSettingsQueryKey = () => {
  return [`/api/v1/imports/scan-settings`] as const;
};

export const getGetLibraryScanSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getLibraryScanSettings>>,
  TError = ErrorType<
    | GetLibraryScanSettings401
    | GetLibraryScanSettings403
    | GetLibraryScanSettings409
    | GetLibraryScanSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getLibraryScanSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetLibraryScanSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getLibraryScanSettings>>> = ({ signal }) =>
    getLibraryScanSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof getLibraryScanSettings>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetLibraryScanSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getLibraryScanSettings>>
>;
export type GetLibraryScanSettingsSuspenseQueryError = ErrorType<
  | GetLibraryScanSettings401
  | GetLibraryScanSettings403
  | GetLibraryScanSettings409
  | GetLibraryScanSettings500
>;

export function useGetLibraryScanSettingsSuspense<
  TData = Awaited<ReturnType<typeof getLibraryScanSettings>>,
  TError = ErrorType<
    | GetLibraryScanSettings401
    | GetLibraryScanSettings403
    | GetLibraryScanSettings409
    | GetLibraryScanSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getLibraryScanSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetLibraryScanSettingsSuspense<
  TData = Awaited<ReturnType<typeof getLibraryScanSettings>>,
  TError = ErrorType<
    | GetLibraryScanSettings401
    | GetLibraryScanSettings403
    | GetLibraryScanSettings409
    | GetLibraryScanSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getLibraryScanSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetLibraryScanSettingsSuspense<
  TData = Awaited<ReturnType<typeof getLibraryScanSettings>>,
  TError = ErrorType<
    | GetLibraryScanSettings401
    | GetLibraryScanSettings403
    | GetLibraryScanSettings409
    | GetLibraryScanSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getLibraryScanSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Library scan settings
 */

export function useGetLibraryScanSettingsSuspense<
  TData = Awaited<ReturnType<typeof getLibraryScanSettings>>,
  TError = ErrorType<
    | GetLibraryScanSettings401
    | GetLibraryScanSettings403
    | GetLibraryScanSettings409
    | GetLibraryScanSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getLibraryScanSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetLibraryScanSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateLibraryScanSettingsUrl = () => {
  return `/api/v1/imports/scan-settings`;
};

/**
 * @summary Change the Library scan settings
 */
export const updateLibraryScanSettings = async (
  libraryScanSettingsInputBody: LibraryScanSettingsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsSettings> => {
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
  return apiFetch<ImportsSettings>(getUpdateLibraryScanSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(libraryScanSettingsInputBody),
  });
};

export const getUpdateLibraryScanSettingsMutationKey = () => ["updateLibraryScanSettings"] as const;

export const getUpdateLibraryScanSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateLibraryScanSettings401
    | UpdateLibraryScanSettings403
    | UpdateLibraryScanSettings409
    | UpdateLibraryScanSettings422
    | UpdateLibraryScanSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateLibraryScanSettings>>,
    TError,
    UpdateLibraryScanSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateLibraryScanSettings>>,
  TError,
  UpdateLibraryScanSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateLibraryScanSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateLibraryScanSettings>>,
    UpdateLibraryScanSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateLibraryScanSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateLibraryScanSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateLibraryScanSettings>>
>;
export type UpdateLibraryScanSettingsMutationBody = LibraryScanSettingsInputBody;
export type UpdateLibraryScanSettingsMutationError = ErrorType<
  | UpdateLibraryScanSettings401
  | UpdateLibraryScanSettings403
  | UpdateLibraryScanSettings409
  | UpdateLibraryScanSettings422
  | UpdateLibraryScanSettings500
>;
export type UpdateLibraryScanSettingsMutationVariables = { data: LibraryScanSettingsInputBody };

/**
 * @summary Change the Library scan settings
 */
export const useUpdateLibraryScanSettings = <
  TError = ErrorType<
    | UpdateLibraryScanSettings401
    | UpdateLibraryScanSettings403
    | UpdateLibraryScanSettings409
    | UpdateLibraryScanSettings422
    | UpdateLibraryScanSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateLibraryScanSettings>>,
      TError,
      UpdateLibraryScanSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateLibraryScanSettings>>,
  TError,
  UpdateLibraryScanSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateLibraryScanSettingsMutationOptions(options), queryClient);
};
export const getUnignoreImportsUrl = () => {
  return `/api/v1/imports/unignore`;
};

/**
 * @summary Bring ignored Imports back to the ones that need the Admin
 */
export const unignoreImports = async (
  importIDsInputBody: ImportIDsInputBody,
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
  return apiFetch<void>(getUnignoreImportsUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(importIDsInputBody),
  });
};

export const getUnignoreImportsMutationKey = () => ["unignoreImports"] as const;

export const getUnignoreImportsMutationOptions = <
  TError = ErrorType<
    | UnignoreImports401
    | UnignoreImports403
    | UnignoreImports409
    | UnignoreImports422
    | UnignoreImports500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof unignoreImports>>,
    TError,
    UnignoreImportsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof unignoreImports>>,
  TError,
  UnignoreImportsMutationVariables,
  TContext
> => {
  const mutationKey = getUnignoreImportsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof unignoreImports>>,
    UnignoreImportsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return unignoreImports(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UnignoreImportsMutationResult = NonNullable<
  Awaited<ReturnType<typeof unignoreImports>>
>;
export type UnignoreImportsMutationBody = ImportIDsInputBody;
export type UnignoreImportsMutationError = ErrorType<
  | UnignoreImports401
  | UnignoreImports403
  | UnignoreImports409
  | UnignoreImports422
  | UnignoreImports500
>;
export type UnignoreImportsMutationVariables = { data: ImportIDsInputBody };

/**
 * @summary Bring ignored Imports back to the ones that need the Admin
 */
export const useUnignoreImports = <
  TError = ErrorType<
    | UnignoreImports401
    | UnignoreImports403
    | UnignoreImports409
    | UnignoreImports422
    | UnignoreImports500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof unignoreImports>>,
      TError,
      UnignoreImportsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof unignoreImports>>,
  TError,
  UnignoreImportsMutationVariables,
  TContext
> => {
  return useMutation(getUnignoreImportsMutationOptions(options), queryClient);
};
export const getGetImportUrl = (id: string) => {
  return `/api/v1/imports/${id}`;
};

/**
 * @summary Get an Import with its files
 */
export const getImport = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsImportEntry> => {
  return apiFetch<ImportsImportEntry>(getGetImportUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetImportQueryKey = (id: string) => {
  return [`/api/v1/imports/${id}`] as const;
};

export const getGetImportSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getImport>>,
  TError = ErrorType<
    GetImport401 | GetImport403 | GetImport404 | GetImport409 | GetImport422 | GetImport500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImport>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetImportQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getImport>>> = ({ signal }) =>
    getImport(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImport>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetImportSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getImport>>>;
export type GetImportSuspenseQueryError = ErrorType<
  GetImport401 | GetImport403 | GetImport404 | GetImport409 | GetImport422 | GetImport500
>;

export function useGetImportSuspense<
  TData = Awaited<ReturnType<typeof getImport>>,
  TError = ErrorType<
    GetImport401 | GetImport403 | GetImport404 | GetImport409 | GetImport422 | GetImport500
  >,
>(
  id: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImport>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetImportSuspense<
  TData = Awaited<ReturnType<typeof getImport>>,
  TError = ErrorType<
    GetImport401 | GetImport403 | GetImport404 | GetImport409 | GetImport422 | GetImport500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImport>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetImportSuspense<
  TData = Awaited<ReturnType<typeof getImport>>,
  TError = ErrorType<
    GetImport401 | GetImport403 | GetImport404 | GetImport409 | GetImport422 | GetImport500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImport>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get an Import with its files
 */

export function useGetImportSuspense<
  TData = Awaited<ReturnType<typeof getImport>>,
  TError = ErrorType<
    GetImport401 | GetImport403 | GetImport404 | GetImport409 | GetImport422 | GetImport500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getImport>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetImportSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getActOnRootImportUrl = (id: string) => {
  return `/api/v1/imports/${id}`;
};

/**
 * @summary Match a root Import to a record at the Content Type's source, or import its chosen files into a Work in its root
 */
export const actOnRootImport = async (
  id: string,
  importsRootAction: ImportsRootAction,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsImportEntry> => {
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
  return apiFetch<ImportsImportEntry>(getActOnRootImportUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(importsRootAction),
  });
};

export const getActOnRootImportMutationKey = () => ["actOnRootImport"] as const;

export const getActOnRootImportMutationOptions = <
  TError = ErrorType<
    | ActOnRootImport401
    | ActOnRootImport403
    | ActOnRootImport404
    | ActOnRootImport409
    | ActOnRootImport422
    | ActOnRootImport500
    | ActOnRootImport502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof actOnRootImport>>,
    TError,
    ActOnRootImportMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof actOnRootImport>>,
  TError,
  ActOnRootImportMutationVariables,
  TContext
> => {
  const mutationKey = getActOnRootImportMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof actOnRootImport>>,
    ActOnRootImportMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return actOnRootImport(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ActOnRootImportMutationResult = NonNullable<
  Awaited<ReturnType<typeof actOnRootImport>>
>;
export type ActOnRootImportMutationBody = ImportsRootAction;
export type ActOnRootImportMutationError = ErrorType<
  | ActOnRootImport401
  | ActOnRootImport403
  | ActOnRootImport404
  | ActOnRootImport409
  | ActOnRootImport422
  | ActOnRootImport500
  | ActOnRootImport502
>;
export type ActOnRootImportMutationVariables = { id: string; data: ImportsRootAction };

/**
 * @summary Match a root Import to a record at the Content Type's source, or import its chosen files into a Work in its root
 */
export const useActOnRootImport = <
  TError = ErrorType<
    | ActOnRootImport401
    | ActOnRootImport403
    | ActOnRootImport404
    | ActOnRootImport409
    | ActOnRootImport422
    | ActOnRootImport500
    | ActOnRootImport502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof actOnRootImport>>,
      TError,
      ActOnRootImportMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof actOnRootImport>>,
  TError,
  ActOnRootImportMutationVariables,
  TContext
> => {
  return useMutation(getActOnRootImportMutationOptions(options), queryClient);
};
export const getActOnImportFileUrl = (id: string, position: number) => {
  return `/api/v1/imports/${id}/files/${position}`;
};

/**
 * @summary Confirm, retry, decline, force or match one file of an Import
 */
export const actOnImportFile = async (
  id: string,
  position: number,
  importsFileAction: ImportsFileAction,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsImportEntry> => {
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
  return apiFetch<ImportsImportEntry>(getActOnImportFileUrl(id, position), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(importsFileAction),
  });
};

export const getActOnImportFileMutationKey = () => ["actOnImportFile"] as const;

export const getActOnImportFileMutationOptions = <
  TError = ErrorType<
    | ActOnImportFile401
    | ActOnImportFile403
    | ActOnImportFile404
    | ActOnImportFile409
    | ActOnImportFile422
    | ActOnImportFile500
    | ActOnImportFile501
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof actOnImportFile>>,
    TError,
    ActOnImportFileMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof actOnImportFile>>,
  TError,
  ActOnImportFileMutationVariables,
  TContext
> => {
  const mutationKey = getActOnImportFileMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof actOnImportFile>>,
    ActOnImportFileMutationVariables
  > = (props) => {
    const { id, position, data } = props ?? {};

    return actOnImportFile(id, position, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ActOnImportFileMutationResult = NonNullable<
  Awaited<ReturnType<typeof actOnImportFile>>
>;
export type ActOnImportFileMutationBody = ImportsFileAction;
export type ActOnImportFileMutationError = ErrorType<
  | ActOnImportFile401
  | ActOnImportFile403
  | ActOnImportFile404
  | ActOnImportFile409
  | ActOnImportFile422
  | ActOnImportFile500
  | ActOnImportFile501
>;
export type ActOnImportFileMutationVariables = {
  id: string;
  position: number;
  data: ImportsFileAction;
};

/**
 * @summary Confirm, retry, decline, force or match one file of an Import
 */
export const useActOnImportFile = <
  TError = ErrorType<
    | ActOnImportFile401
    | ActOnImportFile403
    | ActOnImportFile404
    | ActOnImportFile409
    | ActOnImportFile422
    | ActOnImportFile500
    | ActOnImportFile501
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof actOnImportFile>>,
      TError,
      ActOnImportFileMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof actOnImportFile>>,
  TError,
  ActOnImportFileMutationVariables,
  TContext
> => {
  return useMutation(getActOnImportFileMutationOptions(options), queryClient);
};
