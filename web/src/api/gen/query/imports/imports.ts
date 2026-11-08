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
  AddPendingImports401,
  AddPendingImports403,
  AddPendingImports409,
  AddPendingImports422,
  AddPendingImports500,
  GetLibraryScanSettings401,
  GetLibraryScanSettings403,
  GetLibraryScanSettings409,
  GetLibraryScanSettings500,
  IgnorePendingImports401,
  IgnorePendingImports403,
  IgnorePendingImports409,
  IgnorePendingImports422,
  IgnorePendingImports500,
  ImportIntoInputBody,
  ImportPendingInto401,
  ImportPendingInto403,
  ImportPendingInto404,
  ImportPendingInto409,
  ImportPendingInto422,
  ImportPendingInto500,
  ImportsAddedList,
  ImportsEntry,
  ImportsImported,
  ImportsList,
  ImportsSettings,
  LibraryScanSettingsInputBody,
  ListPendingImports401,
  ListPendingImports403,
  ListPendingImports409,
  ListPendingImports422,
  ListPendingImports500,
  ListPendingImportsParams,
  MatchPendingImport401,
  MatchPendingImport403,
  MatchPendingImport404,
  MatchPendingImport409,
  MatchPendingImport422,
  MatchPendingImport500,
  MatchPendingImport502,
  MatchPendingInputBody,
  PendingIDsInputBody,
  UnignorePendingImports401,
  UnignorePendingImports403,
  UnignorePendingImports409,
  UnignorePendingImports422,
  UnignorePendingImports500,
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

export const getListPendingImportsUrl = (params?: ListPendingImportsParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/imports/pending?${stringifiedParams}`
    : `/api/v1/imports/pending`;
};

/**
 * @summary List the pending imports, with what each root's last library scan added
 */
export const listPendingImports = async (
  params?: ListPendingImportsParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsList> => {
  return apiFetch<ImportsList>(getListPendingImportsUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListPendingImportsQueryKey = (params?: ListPendingImportsParams) => {
  return [`/api/v1/imports/pending`, ...(params ? [params] : [])] as const;
};

export const getListPendingImportsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listPendingImports>>,
  TError = ErrorType<
    | ListPendingImports401
    | ListPendingImports403
    | ListPendingImports409
    | ListPendingImports422
    | ListPendingImports500
  >,
>(
  params?: ListPendingImportsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPendingImports>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListPendingImportsQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listPendingImports>>> = ({ signal }) =>
    listPendingImports(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPendingImports>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListPendingImportsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listPendingImports>>
>;
export type ListPendingImportsSuspenseQueryError = ErrorType<
  | ListPendingImports401
  | ListPendingImports403
  | ListPendingImports409
  | ListPendingImports422
  | ListPendingImports500
>;

export function useListPendingImportsSuspense<
  TData = Awaited<ReturnType<typeof listPendingImports>>,
  TError = ErrorType<
    | ListPendingImports401
    | ListPendingImports403
    | ListPendingImports409
    | ListPendingImports422
    | ListPendingImports500
  >,
>(
  params: undefined | ListPendingImportsParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPendingImports>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListPendingImportsSuspense<
  TData = Awaited<ReturnType<typeof listPendingImports>>,
  TError = ErrorType<
    | ListPendingImports401
    | ListPendingImports403
    | ListPendingImports409
    | ListPendingImports422
    | ListPendingImports500
  >,
>(
  params?: ListPendingImportsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPendingImports>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListPendingImportsSuspense<
  TData = Awaited<ReturnType<typeof listPendingImports>>,
  TError = ErrorType<
    | ListPendingImports401
    | ListPendingImports403
    | ListPendingImports409
    | ListPendingImports422
    | ListPendingImports500
  >,
>(
  params?: ListPendingImportsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPendingImports>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the pending imports, with what each root's last library scan added
 */

export function useListPendingImportsSuspense<
  TData = Awaited<ReturnType<typeof listPendingImports>>,
  TError = ErrorType<
    | ListPendingImports401
    | ListPendingImports403
    | ListPendingImports409
    | ListPendingImports422
    | ListPendingImports500
  >,
>(
  params?: ListPendingImportsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPendingImports>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListPendingImportsSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getAddPendingImportsUrl = () => {
  return `/api/v1/imports/pending/add`;
};

/**
 * @summary Add each pending import's suggested Work to its root and attach its files
 */
export const addPendingImports = async (
  pendingIDsInputBody: PendingIDsInputBody,
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
  return apiFetch<ImportsAddedList>(getAddPendingImportsUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(pendingIDsInputBody),
  });
};

export const getAddPendingImportsMutationKey = () => ["addPendingImports"] as const;

export const getAddPendingImportsMutationOptions = <
  TError = ErrorType<
    | AddPendingImports401
    | AddPendingImports403
    | AddPendingImports409
    | AddPendingImports422
    | AddPendingImports500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof addPendingImports>>,
    TError,
    AddPendingImportsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof addPendingImports>>,
  TError,
  AddPendingImportsMutationVariables,
  TContext
> => {
  const mutationKey = getAddPendingImportsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof addPendingImports>>,
    AddPendingImportsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return addPendingImports(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type AddPendingImportsMutationResult = NonNullable<
  Awaited<ReturnType<typeof addPendingImports>>
>;
export type AddPendingImportsMutationBody = PendingIDsInputBody;
export type AddPendingImportsMutationError = ErrorType<
  | AddPendingImports401
  | AddPendingImports403
  | AddPendingImports409
  | AddPendingImports422
  | AddPendingImports500
>;
export type AddPendingImportsMutationVariables = { data: PendingIDsInputBody };

/**
 * @summary Add each pending import's suggested Work to its root and attach its files
 */
export const useAddPendingImports = <
  TError = ErrorType<
    | AddPendingImports401
    | AddPendingImports403
    | AddPendingImports409
    | AddPendingImports422
    | AddPendingImports500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof addPendingImports>>,
      TError,
      AddPendingImportsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof addPendingImports>>,
  TError,
  AddPendingImportsMutationVariables,
  TContext
> => {
  return useMutation(getAddPendingImportsMutationOptions(options), queryClient);
};
export const getIgnorePendingImportsUrl = () => {
  return `/api/v1/imports/pending/ignore`;
};

/**
 * @summary Ignore pending imports until their files change
 */
export const ignorePendingImports = async (
  pendingIDsInputBody: PendingIDsInputBody,
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
  return apiFetch<void>(getIgnorePendingImportsUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(pendingIDsInputBody),
  });
};

export const getIgnorePendingImportsMutationKey = () => ["ignorePendingImports"] as const;

export const getIgnorePendingImportsMutationOptions = <
  TError = ErrorType<
    | IgnorePendingImports401
    | IgnorePendingImports403
    | IgnorePendingImports409
    | IgnorePendingImports422
    | IgnorePendingImports500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof ignorePendingImports>>,
    TError,
    IgnorePendingImportsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof ignorePendingImports>>,
  TError,
  IgnorePendingImportsMutationVariables,
  TContext
> => {
  const mutationKey = getIgnorePendingImportsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof ignorePendingImports>>,
    IgnorePendingImportsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return ignorePendingImports(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type IgnorePendingImportsMutationResult = NonNullable<
  Awaited<ReturnType<typeof ignorePendingImports>>
>;
export type IgnorePendingImportsMutationBody = PendingIDsInputBody;
export type IgnorePendingImportsMutationError = ErrorType<
  | IgnorePendingImports401
  | IgnorePendingImports403
  | IgnorePendingImports409
  | IgnorePendingImports422
  | IgnorePendingImports500
>;
export type IgnorePendingImportsMutationVariables = { data: PendingIDsInputBody };

/**
 * @summary Ignore pending imports until their files change
 */
export const useIgnorePendingImports = <
  TError = ErrorType<
    | IgnorePendingImports401
    | IgnorePendingImports403
    | IgnorePendingImports409
    | IgnorePendingImports422
    | IgnorePendingImports500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof ignorePendingImports>>,
      TError,
      IgnorePendingImportsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof ignorePendingImports>>,
  TError,
  IgnorePendingImportsMutationVariables,
  TContext
> => {
  return useMutation(getIgnorePendingImportsMutationOptions(options), queryClient);
};
export const getUnignorePendingImportsUrl = () => {
  return `/api/v1/imports/pending/unignore`;
};

/**
 * @summary Bring ignored pending imports back to the ones that need the Admin
 */
export const unignorePendingImports = async (
  pendingIDsInputBody: PendingIDsInputBody,
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
  return apiFetch<void>(getUnignorePendingImportsUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(pendingIDsInputBody),
  });
};

export const getUnignorePendingImportsMutationKey = () => ["unignorePendingImports"] as const;

export const getUnignorePendingImportsMutationOptions = <
  TError = ErrorType<
    | UnignorePendingImports401
    | UnignorePendingImports403
    | UnignorePendingImports409
    | UnignorePendingImports422
    | UnignorePendingImports500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof unignorePendingImports>>,
    TError,
    UnignorePendingImportsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof unignorePendingImports>>,
  TError,
  UnignorePendingImportsMutationVariables,
  TContext
> => {
  const mutationKey = getUnignorePendingImportsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof unignorePendingImports>>,
    UnignorePendingImportsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return unignorePendingImports(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UnignorePendingImportsMutationResult = NonNullable<
  Awaited<ReturnType<typeof unignorePendingImports>>
>;
export type UnignorePendingImportsMutationBody = PendingIDsInputBody;
export type UnignorePendingImportsMutationError = ErrorType<
  | UnignorePendingImports401
  | UnignorePendingImports403
  | UnignorePendingImports409
  | UnignorePendingImports422
  | UnignorePendingImports500
>;
export type UnignorePendingImportsMutationVariables = { data: PendingIDsInputBody };

/**
 * @summary Bring ignored pending imports back to the ones that need the Admin
 */
export const useUnignorePendingImports = <
  TError = ErrorType<
    | UnignorePendingImports401
    | UnignorePendingImports403
    | UnignorePendingImports409
    | UnignorePendingImports422
    | UnignorePendingImports500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof unignorePendingImports>>,
      TError,
      UnignorePendingImportsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof unignorePendingImports>>,
  TError,
  UnignorePendingImportsMutationVariables,
  TContext
> => {
  return useMutation(getUnignorePendingImportsMutationOptions(options), queryClient);
};
export const getImportPendingIntoUrl = (id: string) => {
  return `/api/v1/imports/pending/${id}/import`;
};

/**
 * @summary Attach a pending import's chosen files to a Work in its root, at the Items chosen
 */
export const importPendingInto = async (
  id: string,
  importIntoInputBody: ImportIntoInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsImported> => {
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
  return apiFetch<ImportsImported>(getImportPendingIntoUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(importIntoInputBody),
  });
};

export const getImportPendingIntoMutationKey = () => ["importPendingInto"] as const;

export const getImportPendingIntoMutationOptions = <
  TError = ErrorType<
    | ImportPendingInto401
    | ImportPendingInto403
    | ImportPendingInto404
    | ImportPendingInto409
    | ImportPendingInto422
    | ImportPendingInto500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof importPendingInto>>,
    TError,
    ImportPendingIntoMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof importPendingInto>>,
  TError,
  ImportPendingIntoMutationVariables,
  TContext
> => {
  const mutationKey = getImportPendingIntoMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof importPendingInto>>,
    ImportPendingIntoMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return importPendingInto(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ImportPendingIntoMutationResult = NonNullable<
  Awaited<ReturnType<typeof importPendingInto>>
>;
export type ImportPendingIntoMutationBody = ImportIntoInputBody;
export type ImportPendingIntoMutationError = ErrorType<
  | ImportPendingInto401
  | ImportPendingInto403
  | ImportPendingInto404
  | ImportPendingInto409
  | ImportPendingInto422
  | ImportPendingInto500
>;
export type ImportPendingIntoMutationVariables = { id: string; data: ImportIntoInputBody };

/**
 * @summary Attach a pending import's chosen files to a Work in its root, at the Items chosen
 */
export const useImportPendingInto = <
  TError = ErrorType<
    | ImportPendingInto401
    | ImportPendingInto403
    | ImportPendingInto404
    | ImportPendingInto409
    | ImportPendingInto422
    | ImportPendingInto500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof importPendingInto>>,
      TError,
      ImportPendingIntoMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof importPendingInto>>,
  TError,
  ImportPendingIntoMutationVariables,
  TContext
> => {
  return useMutation(getImportPendingIntoMutationOptions(options), queryClient);
};
export const getMatchPendingImportUrl = (id: string) => {
  return `/api/v1/imports/pending/${id}/match`;
};

/**
 * @summary Make a record at the Content Type's source a pending import's suggestion
 */
export const matchPendingImport = async (
  id: string,
  matchPendingInputBody: MatchPendingInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ImportsEntry> => {
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
  return apiFetch<ImportsEntry>(getMatchPendingImportUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(matchPendingInputBody),
  });
};

export const getMatchPendingImportMutationKey = () => ["matchPendingImport"] as const;

export const getMatchPendingImportMutationOptions = <
  TError = ErrorType<
    | MatchPendingImport401
    | MatchPendingImport403
    | MatchPendingImport404
    | MatchPendingImport409
    | MatchPendingImport422
    | MatchPendingImport500
    | MatchPendingImport502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof matchPendingImport>>,
    TError,
    MatchPendingImportMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof matchPendingImport>>,
  TError,
  MatchPendingImportMutationVariables,
  TContext
> => {
  const mutationKey = getMatchPendingImportMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof matchPendingImport>>,
    MatchPendingImportMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return matchPendingImport(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MatchPendingImportMutationResult = NonNullable<
  Awaited<ReturnType<typeof matchPendingImport>>
>;
export type MatchPendingImportMutationBody = MatchPendingInputBody;
export type MatchPendingImportMutationError = ErrorType<
  | MatchPendingImport401
  | MatchPendingImport403
  | MatchPendingImport404
  | MatchPendingImport409
  | MatchPendingImport422
  | MatchPendingImport500
  | MatchPendingImport502
>;
export type MatchPendingImportMutationVariables = { id: string; data: MatchPendingInputBody };

/**
 * @summary Make a record at the Content Type's source a pending import's suggestion
 */
export const useMatchPendingImport = <
  TError = ErrorType<
    | MatchPendingImport401
    | MatchPendingImport403
    | MatchPendingImport404
    | MatchPendingImport409
    | MatchPendingImport422
    | MatchPendingImport500
    | MatchPendingImport502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof matchPendingImport>>,
      TError,
      MatchPendingImportMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof matchPendingImport>>,
  TError,
  MatchPendingImportMutationVariables,
  TContext
> => {
  return useMutation(getMatchPendingImportMutationOptions(options), queryClient);
};
export const getGetLibraryScanSettingsUrl = () => {
  return `/api/v1/imports/settings`;
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
  return [`/api/v1/imports/settings`] as const;
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
  return `/api/v1/imports/settings`;
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
