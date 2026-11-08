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
  CreateCustomFormat400,
  CreateCustomFormat401,
  CreateCustomFormat403,
  CreateCustomFormat409,
  CreateCustomFormat422,
  CreateCustomFormat500,
  CustomFormatSchema401,
  CustomFormatSchema403,
  CustomFormatSchema409,
  CustomFormatSchema500,
  DeleteCustomFormat401,
  DeleteCustomFormat403,
  DeleteCustomFormat404,
  DeleteCustomFormat409,
  DeleteCustomFormat422,
  DeleteCustomFormat500,
  DeleteCustomFormatParams,
  DuplicateCustomFormat401,
  DuplicateCustomFormat403,
  DuplicateCustomFormat404,
  DuplicateCustomFormat409,
  DuplicateCustomFormat422,
  DuplicateCustomFormat500,
  FormatsCustomFormat,
  FormatsFormatChange,
  FormatsFormatDeletion,
  FormatsFormatList,
  FormatsNewFormat,
  FormatsParse,
  FormatsParseResults,
  FormatsPaste,
  FormatsPasted,
  FormatsSchema,
  GetCustomFormat401,
  GetCustomFormat403,
  GetCustomFormat404,
  GetCustomFormat409,
  GetCustomFormat422,
  GetCustomFormat500,
  ListCustomFormats401,
  ListCustomFormats403,
  ListCustomFormats409,
  ListCustomFormats500,
  ParseReleases400,
  ParseReleases401,
  ParseReleases403,
  ParseReleases404,
  ParseReleases409,
  ParseReleases422,
  ParseReleases500,
  PasteCustomFormats400,
  PasteCustomFormats401,
  PasteCustomFormats403,
  PasteCustomFormats409,
  PasteCustomFormats422,
  PasteCustomFormats500,
  PasteCustomFormatsParams,
  UpdateCustomFormat400,
  UpdateCustomFormat401,
  UpdateCustomFormat403,
  UpdateCustomFormat404,
  UpdateCustomFormat409,
  UpdateCustomFormat422,
  UpdateCustomFormat500,
  UpdateCustomFormatParams,
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

export const getListCustomFormatsUrl = () => {
  return `/api/v1/custom-formats`;
};

/**
 * @summary List the Custom Formats, each with the Targets that score it
 */
export const listCustomFormats = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<FormatsFormatList> => {
  return apiFetch<FormatsFormatList>(getListCustomFormatsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListCustomFormatsQueryKey = () => {
  return [`/api/v1/custom-formats`] as const;
};

export const getListCustomFormatsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listCustomFormats>>,
  TError = ErrorType<
    ListCustomFormats401 | ListCustomFormats403 | ListCustomFormats409 | ListCustomFormats500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCustomFormats>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListCustomFormatsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listCustomFormats>>> = ({ signal }) =>
    listCustomFormats({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCustomFormats>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListCustomFormatsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listCustomFormats>>
>;
export type ListCustomFormatsSuspenseQueryError = ErrorType<
  ListCustomFormats401 | ListCustomFormats403 | ListCustomFormats409 | ListCustomFormats500
>;

export function useListCustomFormatsSuspense<
  TData = Awaited<ReturnType<typeof listCustomFormats>>,
  TError = ErrorType<
    ListCustomFormats401 | ListCustomFormats403 | ListCustomFormats409 | ListCustomFormats500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCustomFormats>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListCustomFormatsSuspense<
  TData = Awaited<ReturnType<typeof listCustomFormats>>,
  TError = ErrorType<
    ListCustomFormats401 | ListCustomFormats403 | ListCustomFormats409 | ListCustomFormats500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCustomFormats>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListCustomFormatsSuspense<
  TData = Awaited<ReturnType<typeof listCustomFormats>>,
  TError = ErrorType<
    ListCustomFormats401 | ListCustomFormats403 | ListCustomFormats409 | ListCustomFormats500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCustomFormats>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Custom Formats, each with the Targets that score it
 */

export function useListCustomFormatsSuspense<
  TData = Awaited<ReturnType<typeof listCustomFormats>>,
  TError = ErrorType<
    ListCustomFormats401 | ListCustomFormats403 | ListCustomFormats409 | ListCustomFormats500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCustomFormats>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListCustomFormatsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCreateCustomFormatUrl = () => {
  return `/api/v1/custom-formats`;
};

/**
 * @summary Make a Custom Format in SlipStream's own values. No Target scores it until one is given a score for it
 */
export const createCustomFormat = async (
  formatsNewFormat: FormatsNewFormat,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FormatsCustomFormat> => {
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
  return apiFetch<FormatsCustomFormat>(getCreateCustomFormatUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(formatsNewFormat),
  });
};

export const getCreateCustomFormatMutationKey = () => ["createCustomFormat"] as const;

export const getCreateCustomFormatMutationOptions = <
  TError = ErrorType<
    | CreateCustomFormat400
    | CreateCustomFormat401
    | CreateCustomFormat403
    | CreateCustomFormat409
    | CreateCustomFormat422
    | CreateCustomFormat500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof createCustomFormat>>,
    TError,
    CreateCustomFormatMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof createCustomFormat>>,
  TError,
  CreateCustomFormatMutationVariables,
  TContext
> => {
  const mutationKey = getCreateCustomFormatMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof createCustomFormat>>,
    CreateCustomFormatMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return createCustomFormat(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateCustomFormatMutationResult = NonNullable<
  Awaited<ReturnType<typeof createCustomFormat>>
>;
export type CreateCustomFormatMutationBody = FormatsNewFormat;
export type CreateCustomFormatMutationError = ErrorType<
  | CreateCustomFormat400
  | CreateCustomFormat401
  | CreateCustomFormat403
  | CreateCustomFormat409
  | CreateCustomFormat422
  | CreateCustomFormat500
>;
export type CreateCustomFormatMutationVariables = { data: FormatsNewFormat };

/**
 * @summary Make a Custom Format in SlipStream's own values. No Target scores it until one is given a score for it
 */
export const useCreateCustomFormat = <
  TError = ErrorType<
    | CreateCustomFormat400
    | CreateCustomFormat401
    | CreateCustomFormat403
    | CreateCustomFormat409
    | CreateCustomFormat422
    | CreateCustomFormat500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createCustomFormat>>,
      TError,
      CreateCustomFormatMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof createCustomFormat>>,
  TError,
  CreateCustomFormatMutationVariables,
  TContext
> => {
  return useMutation(getCreateCustomFormatMutationOptions(options), queryClient);
};
export const getPasteCustomFormatsUrl = (params?: PasteCustomFormatsParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/custom-formats/paste?${stringifiedParams}`
    : `/api/v1/custom-formats/paste`;
};

/**
 * @summary Add Custom Formats from Radarr, Sonarr or TRaSH Guides JSON, replacing or keeping beside each one of the same name, or preview what the paste adds and replaces
 */
export const pasteCustomFormats = async (
  formatsPaste: FormatsPaste,
  params?: PasteCustomFormatsParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FormatsPasted> => {
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
  return apiFetch<FormatsPasted>(getPasteCustomFormatsUrl(params), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(formatsPaste),
  });
};

export const getPasteCustomFormatsMutationKey = () => ["pasteCustomFormats"] as const;

export const getPasteCustomFormatsMutationOptions = <
  TError = ErrorType<
    | PasteCustomFormats400
    | PasteCustomFormats401
    | PasteCustomFormats403
    | PasteCustomFormats409
    | PasteCustomFormats422
    | PasteCustomFormats500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof pasteCustomFormats>>,
    TError,
    PasteCustomFormatsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof pasteCustomFormats>>,
  TError,
  PasteCustomFormatsMutationVariables,
  TContext
> => {
  const mutationKey = getPasteCustomFormatsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof pasteCustomFormats>>,
    PasteCustomFormatsMutationVariables
  > = (props) => {
    const { data, params } = props ?? {};

    return pasteCustomFormats(data, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PasteCustomFormatsMutationResult = NonNullable<
  Awaited<ReturnType<typeof pasteCustomFormats>>
>;
export type PasteCustomFormatsMutationBody = FormatsPaste;
export type PasteCustomFormatsMutationError = ErrorType<
  | PasteCustomFormats400
  | PasteCustomFormats401
  | PasteCustomFormats403
  | PasteCustomFormats409
  | PasteCustomFormats422
  | PasteCustomFormats500
>;
export type PasteCustomFormatsMutationVariables = {
  data: FormatsPaste;
  params?: PasteCustomFormatsParams;
};

/**
 * @summary Add Custom Formats from Radarr, Sonarr or TRaSH Guides JSON, replacing or keeping beside each one of the same name, or preview what the paste adds and replaces
 */
export const usePasteCustomFormats = <
  TError = ErrorType<
    | PasteCustomFormats400
    | PasteCustomFormats401
    | PasteCustomFormats403
    | PasteCustomFormats409
    | PasteCustomFormats422
    | PasteCustomFormats500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof pasteCustomFormats>>,
      TError,
      PasteCustomFormatsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof pasteCustomFormats>>,
  TError,
  PasteCustomFormatsMutationVariables,
  TContext
> => {
  return useMutation(getPasteCustomFormatsMutationOptions(options), queryClient);
};
export const getCustomFormatSchemaUrl = () => {
  return `/api/v1/custom-formats/schema`;
};

/**
 * @summary List the condition kinds each app's Custom Formats take, with each kind's fields and values in words
 */
export const customFormatSchema = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<FormatsSchema> => {
  return apiFetch<FormatsSchema>(getCustomFormatSchemaUrl(), {
    ...options,
    method: "GET",
  });
};

export const getCustomFormatSchemaQueryKey = () => {
  return [`/api/v1/custom-formats/schema`] as const;
};

export const getCustomFormatSchemaSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof customFormatSchema>>,
  TError = ErrorType<
    CustomFormatSchema401 | CustomFormatSchema403 | CustomFormatSchema409 | CustomFormatSchema500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof customFormatSchema>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getCustomFormatSchemaQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof customFormatSchema>>> = ({ signal }) =>
    customFormatSchema({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof customFormatSchema>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type CustomFormatSchemaSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof customFormatSchema>>
>;
export type CustomFormatSchemaSuspenseQueryError = ErrorType<
  CustomFormatSchema401 | CustomFormatSchema403 | CustomFormatSchema409 | CustomFormatSchema500
>;

export function useCustomFormatSchemaSuspense<
  TData = Awaited<ReturnType<typeof customFormatSchema>>,
  TError = ErrorType<
    CustomFormatSchema401 | CustomFormatSchema403 | CustomFormatSchema409 | CustomFormatSchema500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof customFormatSchema>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useCustomFormatSchemaSuspense<
  TData = Awaited<ReturnType<typeof customFormatSchema>>,
  TError = ErrorType<
    CustomFormatSchema401 | CustomFormatSchema403 | CustomFormatSchema409 | CustomFormatSchema500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof customFormatSchema>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useCustomFormatSchemaSuspense<
  TData = Awaited<ReturnType<typeof customFormatSchema>>,
  TError = ErrorType<
    CustomFormatSchema401 | CustomFormatSchema403 | CustomFormatSchema409 | CustomFormatSchema500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof customFormatSchema>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the condition kinds each app's Custom Formats take, with each kind's fields and values in words
 */

export function useCustomFormatSchemaSuspense<
  TData = Awaited<ReturnType<typeof customFormatSchema>>,
  TError = ErrorType<
    CustomFormatSchema401 | CustomFormatSchema403 | CustomFormatSchema409 | CustomFormatSchema500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof customFormatSchema>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getCustomFormatSchemaSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getDeleteCustomFormatUrl = (formatId: string, params?: DeleteCustomFormatParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/custom-formats/${formatId}?${stringifiedParams}`
    : `/api/v1/custom-formats/${formatId}`;
};

/**
 * @summary Delete a Custom Format and every Target's score for it, refilling the Works of the Targets that scored it, or preview the change's Impact
 */
export const deleteCustomFormat = async (
  formatId: string,
  params?: DeleteCustomFormatParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FormatsFormatDeletion> => {
  return apiFetch<FormatsFormatDeletion>(getDeleteCustomFormatUrl(formatId, params), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteCustomFormatMutationKey = () => ["deleteCustomFormat"] as const;

export const getDeleteCustomFormatMutationOptions = <
  TError = ErrorType<
    | DeleteCustomFormat401
    | DeleteCustomFormat403
    | DeleteCustomFormat404
    | DeleteCustomFormat409
    | DeleteCustomFormat422
    | DeleteCustomFormat500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteCustomFormat>>,
    TError,
    DeleteCustomFormatMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteCustomFormat>>,
  TError,
  DeleteCustomFormatMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteCustomFormatMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteCustomFormat>>,
    DeleteCustomFormatMutationVariables
  > = (props) => {
    const { formatId, params } = props ?? {};

    return deleteCustomFormat(formatId, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteCustomFormatMutationResult = NonNullable<
  Awaited<ReturnType<typeof deleteCustomFormat>>
>;

export type DeleteCustomFormatMutationError = ErrorType<
  | DeleteCustomFormat401
  | DeleteCustomFormat403
  | DeleteCustomFormat404
  | DeleteCustomFormat409
  | DeleteCustomFormat422
  | DeleteCustomFormat500
>;
export type DeleteCustomFormatMutationVariables = {
  formatId: string;
  params?: DeleteCustomFormatParams;
};

/**
 * @summary Delete a Custom Format and every Target's score for it, refilling the Works of the Targets that scored it, or preview the change's Impact
 */
export const useDeleteCustomFormat = <
  TError = ErrorType<
    | DeleteCustomFormat401
    | DeleteCustomFormat403
    | DeleteCustomFormat404
    | DeleteCustomFormat409
    | DeleteCustomFormat422
    | DeleteCustomFormat500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteCustomFormat>>,
      TError,
      DeleteCustomFormatMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteCustomFormat>>,
  TError,
  DeleteCustomFormatMutationVariables,
  TContext
> => {
  return useMutation(getDeleteCustomFormatMutationOptions(options), queryClient);
};
export const getGetCustomFormatUrl = (formatId: string) => {
  return `/api/v1/custom-formats/${formatId}`;
};

/**
 * @summary Get a Custom Format
 */
export const getCustomFormat = async (
  formatId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FormatsCustomFormat> => {
  return apiFetch<FormatsCustomFormat>(getGetCustomFormatUrl(formatId), {
    ...options,
    method: "GET",
  });
};

export const getGetCustomFormatQueryKey = (formatId: string) => {
  return [`/api/v1/custom-formats/${formatId}`] as const;
};

export const getGetCustomFormatSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getCustomFormat>>,
  TError = ErrorType<
    | GetCustomFormat401
    | GetCustomFormat403
    | GetCustomFormat404
    | GetCustomFormat409
    | GetCustomFormat422
    | GetCustomFormat500
  >,
>(
  formatId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCustomFormat>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetCustomFormatQueryKey(formatId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getCustomFormat>>> = ({ signal }) =>
    getCustomFormat(formatId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCustomFormat>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetCustomFormatSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getCustomFormat>>
>;
export type GetCustomFormatSuspenseQueryError = ErrorType<
  | GetCustomFormat401
  | GetCustomFormat403
  | GetCustomFormat404
  | GetCustomFormat409
  | GetCustomFormat422
  | GetCustomFormat500
>;

export function useGetCustomFormatSuspense<
  TData = Awaited<ReturnType<typeof getCustomFormat>>,
  TError = ErrorType<
    | GetCustomFormat401
    | GetCustomFormat403
    | GetCustomFormat404
    | GetCustomFormat409
    | GetCustomFormat422
    | GetCustomFormat500
  >,
>(
  formatId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCustomFormat>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetCustomFormatSuspense<
  TData = Awaited<ReturnType<typeof getCustomFormat>>,
  TError = ErrorType<
    | GetCustomFormat401
    | GetCustomFormat403
    | GetCustomFormat404
    | GetCustomFormat409
    | GetCustomFormat422
    | GetCustomFormat500
  >,
>(
  formatId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCustomFormat>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetCustomFormatSuspense<
  TData = Awaited<ReturnType<typeof getCustomFormat>>,
  TError = ErrorType<
    | GetCustomFormat401
    | GetCustomFormat403
    | GetCustomFormat404
    | GetCustomFormat409
    | GetCustomFormat422
    | GetCustomFormat500
  >,
>(
  formatId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCustomFormat>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Custom Format
 */

export function useGetCustomFormatSuspense<
  TData = Awaited<ReturnType<typeof getCustomFormat>>,
  TError = ErrorType<
    | GetCustomFormat401
    | GetCustomFormat403
    | GetCustomFormat404
    | GetCustomFormat409
    | GetCustomFormat422
    | GetCustomFormat500
  >,
>(
  formatId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCustomFormat>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetCustomFormatSuspenseQueryOptions(formatId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateCustomFormatUrl = (formatId: string, params?: UpdateCustomFormatParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/custom-formats/${formatId}?${stringifiedParams}`
    : `/api/v1/custom-formats/${formatId}`;
};

/**
 * @summary Replace a Custom Format's name, renaming flag and conditions, refilling the Works of the Targets that score it, or preview the change's Impact
 */
export const updateCustomFormat = async (
  formatId: string,
  formatsNewFormat: FormatsNewFormat,
  params?: UpdateCustomFormatParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FormatsFormatChange> => {
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
  return apiFetch<FormatsFormatChange>(getUpdateCustomFormatUrl(formatId, params), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(formatsNewFormat),
  });
};

export const getUpdateCustomFormatMutationKey = () => ["updateCustomFormat"] as const;

export const getUpdateCustomFormatMutationOptions = <
  TError = ErrorType<
    | UpdateCustomFormat400
    | UpdateCustomFormat401
    | UpdateCustomFormat403
    | UpdateCustomFormat404
    | UpdateCustomFormat409
    | UpdateCustomFormat422
    | UpdateCustomFormat500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateCustomFormat>>,
    TError,
    UpdateCustomFormatMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateCustomFormat>>,
  TError,
  UpdateCustomFormatMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateCustomFormatMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateCustomFormat>>,
    UpdateCustomFormatMutationVariables
  > = (props) => {
    const { formatId, data, params } = props ?? {};

    return updateCustomFormat(formatId, data, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateCustomFormatMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateCustomFormat>>
>;
export type UpdateCustomFormatMutationBody = FormatsNewFormat;
export type UpdateCustomFormatMutationError = ErrorType<
  | UpdateCustomFormat400
  | UpdateCustomFormat401
  | UpdateCustomFormat403
  | UpdateCustomFormat404
  | UpdateCustomFormat409
  | UpdateCustomFormat422
  | UpdateCustomFormat500
>;
export type UpdateCustomFormatMutationVariables = {
  formatId: string;
  data: FormatsNewFormat;
  params?: UpdateCustomFormatParams;
};

/**
 * @summary Replace a Custom Format's name, renaming flag and conditions, refilling the Works of the Targets that score it, or preview the change's Impact
 */
export const useUpdateCustomFormat = <
  TError = ErrorType<
    | UpdateCustomFormat400
    | UpdateCustomFormat401
    | UpdateCustomFormat403
    | UpdateCustomFormat404
    | UpdateCustomFormat409
    | UpdateCustomFormat422
    | UpdateCustomFormat500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateCustomFormat>>,
      TError,
      UpdateCustomFormatMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateCustomFormat>>,
  TError,
  UpdateCustomFormatMutationVariables,
  TContext
> => {
  return useMutation(getUpdateCustomFormatMutationOptions(options), queryClient);
};
export const getDuplicateCustomFormatUrl = (formatId: string) => {
  return `/api/v1/custom-formats/${formatId}/duplicate`;
};

/**
 * @summary Copy a Custom Format with its app and conditions as they are, under its name with a number after it. No Target scores the copy until one is given a score for it
 */
export const duplicateCustomFormat = async (
  formatId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FormatsCustomFormat> => {
  return apiFetch<FormatsCustomFormat>(getDuplicateCustomFormatUrl(formatId), {
    ...options,
    method: "POST",
  });
};

export const getDuplicateCustomFormatMutationKey = () => ["duplicateCustomFormat"] as const;

export const getDuplicateCustomFormatMutationOptions = <
  TError = ErrorType<
    | DuplicateCustomFormat401
    | DuplicateCustomFormat403
    | DuplicateCustomFormat404
    | DuplicateCustomFormat409
    | DuplicateCustomFormat422
    | DuplicateCustomFormat500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof duplicateCustomFormat>>,
    TError,
    DuplicateCustomFormatMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof duplicateCustomFormat>>,
  TError,
  DuplicateCustomFormatMutationVariables,
  TContext
> => {
  const mutationKey = getDuplicateCustomFormatMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof duplicateCustomFormat>>,
    DuplicateCustomFormatMutationVariables
  > = (props) => {
    const { formatId } = props ?? {};

    return duplicateCustomFormat(formatId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DuplicateCustomFormatMutationResult = NonNullable<
  Awaited<ReturnType<typeof duplicateCustomFormat>>
>;

export type DuplicateCustomFormatMutationError = ErrorType<
  | DuplicateCustomFormat401
  | DuplicateCustomFormat403
  | DuplicateCustomFormat404
  | DuplicateCustomFormat409
  | DuplicateCustomFormat422
  | DuplicateCustomFormat500
>;
export type DuplicateCustomFormatMutationVariables = { formatId: string };

/**
 * @summary Copy a Custom Format with its app and conditions as they are, under its name with a number after it. No Target scores the copy until one is given a score for it
 */
export const useDuplicateCustomFormat = <
  TError = ErrorType<
    | DuplicateCustomFormat401
    | DuplicateCustomFormat403
    | DuplicateCustomFormat404
    | DuplicateCustomFormat409
    | DuplicateCustomFormat422
    | DuplicateCustomFormat500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof duplicateCustomFormat>>,
      TError,
      DuplicateCustomFormatMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof duplicateCustomFormat>>,
  TError,
  DuplicateCustomFormatMutationVariables,
  TContext
> => {
  return useMutation(getDuplicateCustomFormatMutationOptions(options), queryClient);
};
export const getParseReleasesUrl = () => {
  return `/api/v1/releases/parse`;
};

/**
 * @summary Read release or file names as a search would: the facts, the Work each matches, every Custom Format with each condition's result, and the score and verdict at each Target
 */
export const parseReleases = async (
  formatsParse: FormatsParse,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FormatsParseResults> => {
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
  return apiFetch<FormatsParseResults>(getParseReleasesUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(formatsParse),
  });
};

export const getParseReleasesMutationKey = () => ["parseReleases"] as const;

export const getParseReleasesMutationOptions = <
  TError = ErrorType<
    | ParseReleases400
    | ParseReleases401
    | ParseReleases403
    | ParseReleases404
    | ParseReleases409
    | ParseReleases422
    | ParseReleases500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof parseReleases>>,
    TError,
    ParseReleasesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof parseReleases>>,
  TError,
  ParseReleasesMutationVariables,
  TContext
> => {
  const mutationKey = getParseReleasesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof parseReleases>>,
    ParseReleasesMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return parseReleases(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ParseReleasesMutationResult = NonNullable<Awaited<ReturnType<typeof parseReleases>>>;
export type ParseReleasesMutationBody = FormatsParse;
export type ParseReleasesMutationError = ErrorType<
  | ParseReleases400
  | ParseReleases401
  | ParseReleases403
  | ParseReleases404
  | ParseReleases409
  | ParseReleases422
  | ParseReleases500
>;
export type ParseReleasesMutationVariables = { data: FormatsParse };

/**
 * @summary Read release or file names as a search would: the facts, the Work each matches, every Custom Format with each condition's result, and the score and verdict at each Target
 */
export const useParseReleases = <
  TError = ErrorType<
    | ParseReleases400
    | ParseReleases401
    | ParseReleases403
    | ParseReleases404
    | ParseReleases409
    | ParseReleases422
    | ParseReleases500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof parseReleases>>,
      TError,
      ParseReleasesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof parseReleases>>,
  TError,
  ParseReleasesMutationVariables,
  TContext
> => {
  return useMutation(getParseReleasesMutationOptions(options), queryClient);
};
