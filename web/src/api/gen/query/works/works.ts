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
  AddWork401,
  AddWork403,
  AddWork404,
  AddWork409,
  AddWork422,
  AddWork500,
  AddWork502,
  GetContentTypeSettings401,
  GetContentTypeSettings403,
  GetContentTypeSettings404,
  GetContentTypeSettings409,
  GetContentTypeSettings422,
  GetContentTypeSettings500,
  GetWork401,
  GetWork403,
  GetWork404,
  GetWork409,
  GetWork422,
  GetWork500,
  ListContentTypes401,
  ListContentTypes403,
  ListContentTypes409,
  ListContentTypes500,
  ListWorks401,
  ListWorks403,
  ListWorks404,
  ListWorks409,
  ListWorks422,
  ListWorks500,
  ListWorksParams,
  PreviewCatalogRecord401,
  PreviewCatalogRecord403,
  PreviewCatalogRecord404,
  PreviewCatalogRecord409,
  PreviewCatalogRecord422,
  PreviewCatalogRecord500,
  PreviewCatalogRecord502,
  RefreshWork401,
  RefreshWork403,
  RefreshWork404,
  RefreshWork409,
  RefreshWork422,
  RefreshWork500,
  RefreshWork502,
  SearchCatalog401,
  SearchCatalog403,
  SearchCatalog404,
  SearchCatalog409,
  SearchCatalog422,
  SearchCatalog500,
  SearchCatalog502,
  SearchCatalogParams,
  SetContentTypeEnabled401,
  SetContentTypeEnabled403,
  SetContentTypeEnabled404,
  SetContentTypeEnabled409,
  SetContentTypeEnabled422,
  SetContentTypeEnabled500,
  UpdateContentTypeSettings401,
  UpdateContentTypeSettings403,
  UpdateContentTypeSettings404,
  UpdateContentTypeSettings409,
  UpdateContentTypeSettings422,
  UpdateContentTypeSettings500,
  WorksAdded,
  WorksCandidateList,
  WorksContentType,
  WorksContentTypeList,
  WorksEnabled,
  WorksNewWork,
  WorksPreviewed,
  WorksSettingsChange,
  WorksTree,
  WorksTypeSettings,
  WorksWorkList,
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

export const getListContentTypesUrl = () => {
  return `/api/v1/content-types`;
};

/**
 * @summary List the Content Types this install has, in id order, off ones included
 */
export const listContentTypes = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<WorksContentTypeList> => {
  return apiFetch<WorksContentTypeList>(getListContentTypesUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListContentTypesQueryKey = () => {
  return [`/api/v1/content-types`] as const;
};

export const getListContentTypesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listContentTypes>>,
  TError = ErrorType<
    ListContentTypes401 | ListContentTypes403 | ListContentTypes409 | ListContentTypes500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listContentTypes>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListContentTypesQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listContentTypes>>> = ({ signal }) =>
    listContentTypes({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listContentTypes>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListContentTypesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listContentTypes>>
>;
export type ListContentTypesSuspenseQueryError = ErrorType<
  ListContentTypes401 | ListContentTypes403 | ListContentTypes409 | ListContentTypes500
>;

export function useListContentTypesSuspense<
  TData = Awaited<ReturnType<typeof listContentTypes>>,
  TError = ErrorType<
    ListContentTypes401 | ListContentTypes403 | ListContentTypes409 | ListContentTypes500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listContentTypes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListContentTypesSuspense<
  TData = Awaited<ReturnType<typeof listContentTypes>>,
  TError = ErrorType<
    ListContentTypes401 | ListContentTypes403 | ListContentTypes409 | ListContentTypes500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listContentTypes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListContentTypesSuspense<
  TData = Awaited<ReturnType<typeof listContentTypes>>,
  TError = ErrorType<
    ListContentTypes401 | ListContentTypes403 | ListContentTypes409 | ListContentTypes500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listContentTypes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Content Types this install has, in id order, off ones included
 */

export function useListContentTypesSuspense<
  TData = Awaited<ReturnType<typeof listContentTypes>>,
  TError = ErrorType<
    ListContentTypes401 | ListContentTypes403 | ListContentTypes409 | ListContentTypes500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listContentTypes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListContentTypesSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSearchCatalogUrl = (contentType: string, params: SearchCatalogParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/content-types/${contentType}/catalog?${stringifiedParams}`
    : `/api/v1/content-types/${contentType}/catalog`;
};

/**
 * @summary Search a Content Type's metadata source
 */
export const searchCatalog = async (
  contentType: string,
  params: SearchCatalogParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<WorksCandidateList> => {
  return apiFetch<WorksCandidateList>(getSearchCatalogUrl(contentType, params), {
    ...options,
    method: "GET",
  });
};

export const getSearchCatalogQueryKey = (contentType: string, params?: SearchCatalogParams) => {
  return [`/api/v1/content-types/${contentType}/catalog`, ...(params ? [params] : [])] as const;
};

export const getSearchCatalogSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof searchCatalog>>,
  TError = ErrorType<
    | SearchCatalog401
    | SearchCatalog403
    | SearchCatalog404
    | SearchCatalog409
    | SearchCatalog422
    | SearchCatalog500
    | SearchCatalog502
  >,
>(
  contentType: string,
  params: SearchCatalogParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof searchCatalog>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getSearchCatalogQueryKey(contentType, params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof searchCatalog>>> = ({ signal }) =>
    searchCatalog(contentType, params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof searchCatalog>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type SearchCatalogSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof searchCatalog>>
>;
export type SearchCatalogSuspenseQueryError = ErrorType<
  | SearchCatalog401
  | SearchCatalog403
  | SearchCatalog404
  | SearchCatalog409
  | SearchCatalog422
  | SearchCatalog500
  | SearchCatalog502
>;

export function useSearchCatalogSuspense<
  TData = Awaited<ReturnType<typeof searchCatalog>>,
  TError = ErrorType<
    | SearchCatalog401
    | SearchCatalog403
    | SearchCatalog404
    | SearchCatalog409
    | SearchCatalog422
    | SearchCatalog500
    | SearchCatalog502
  >,
>(
  contentType: string,
  params: SearchCatalogParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof searchCatalog>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useSearchCatalogSuspense<
  TData = Awaited<ReturnType<typeof searchCatalog>>,
  TError = ErrorType<
    | SearchCatalog401
    | SearchCatalog403
    | SearchCatalog404
    | SearchCatalog409
    | SearchCatalog422
    | SearchCatalog500
    | SearchCatalog502
  >,
>(
  contentType: string,
  params: SearchCatalogParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof searchCatalog>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useSearchCatalogSuspense<
  TData = Awaited<ReturnType<typeof searchCatalog>>,
  TError = ErrorType<
    | SearchCatalog401
    | SearchCatalog403
    | SearchCatalog404
    | SearchCatalog409
    | SearchCatalog422
    | SearchCatalog500
    | SearchCatalog502
  >,
>(
  contentType: string,
  params: SearchCatalogParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof searchCatalog>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Search a Content Type's metadata source
 */

export function useSearchCatalogSuspense<
  TData = Awaited<ReturnType<typeof searchCatalog>>,
  TError = ErrorType<
    | SearchCatalog401
    | SearchCatalog403
    | SearchCatalog404
    | SearchCatalog409
    | SearchCatalog422
    | SearchCatalog500
    | SearchCatalog502
  >,
>(
  contentType: string,
  params: SearchCatalogParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof searchCatalog>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getSearchCatalogSuspenseQueryOptions(contentType, params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getPreviewCatalogRecordUrl = (contentType: string, key: string) => {
  return `/api/v1/content-types/${contentType}/catalog/${key}`;
};

/**
 * @summary Read a record of a Content Type's metadata source as the Work adding it would make, without adding it or reading ratings
 */
export const previewCatalogRecord = async (
  contentType: string,
  key: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<WorksPreviewed> => {
  return apiFetch<WorksPreviewed>(getPreviewCatalogRecordUrl(contentType, key), {
    ...options,
    method: "GET",
  });
};

export const getPreviewCatalogRecordQueryKey = (contentType: string, key: string) => {
  return [`/api/v1/content-types/${contentType}/catalog/${key}`] as const;
};

export const getPreviewCatalogRecordSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof previewCatalogRecord>>,
  TError = ErrorType<
    | PreviewCatalogRecord401
    | PreviewCatalogRecord403
    | PreviewCatalogRecord404
    | PreviewCatalogRecord409
    | PreviewCatalogRecord422
    | PreviewCatalogRecord500
    | PreviewCatalogRecord502
  >,
>(
  contentType: string,
  key: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewCatalogRecord>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getPreviewCatalogRecordQueryKey(contentType, key);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof previewCatalogRecord>>> = ({ signal }) =>
    previewCatalogRecord(contentType, key, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewCatalogRecord>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type PreviewCatalogRecordSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof previewCatalogRecord>>
>;
export type PreviewCatalogRecordSuspenseQueryError = ErrorType<
  | PreviewCatalogRecord401
  | PreviewCatalogRecord403
  | PreviewCatalogRecord404
  | PreviewCatalogRecord409
  | PreviewCatalogRecord422
  | PreviewCatalogRecord500
  | PreviewCatalogRecord502
>;

export function usePreviewCatalogRecordSuspense<
  TData = Awaited<ReturnType<typeof previewCatalogRecord>>,
  TError = ErrorType<
    | PreviewCatalogRecord401
    | PreviewCatalogRecord403
    | PreviewCatalogRecord404
    | PreviewCatalogRecord409
    | PreviewCatalogRecord422
    | PreviewCatalogRecord500
    | PreviewCatalogRecord502
  >,
>(
  contentType: string,
  key: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewCatalogRecord>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function usePreviewCatalogRecordSuspense<
  TData = Awaited<ReturnType<typeof previewCatalogRecord>>,
  TError = ErrorType<
    | PreviewCatalogRecord401
    | PreviewCatalogRecord403
    | PreviewCatalogRecord404
    | PreviewCatalogRecord409
    | PreviewCatalogRecord422
    | PreviewCatalogRecord500
    | PreviewCatalogRecord502
  >,
>(
  contentType: string,
  key: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewCatalogRecord>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function usePreviewCatalogRecordSuspense<
  TData = Awaited<ReturnType<typeof previewCatalogRecord>>,
  TError = ErrorType<
    | PreviewCatalogRecord401
    | PreviewCatalogRecord403
    | PreviewCatalogRecord404
    | PreviewCatalogRecord409
    | PreviewCatalogRecord422
    | PreviewCatalogRecord500
    | PreviewCatalogRecord502
  >,
>(
  contentType: string,
  key: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewCatalogRecord>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Read a record of a Content Type's metadata source as the Work adding it would make, without adding it or reading ratings
 */

export function usePreviewCatalogRecordSuspense<
  TData = Awaited<ReturnType<typeof previewCatalogRecord>>,
  TError = ErrorType<
    | PreviewCatalogRecord401
    | PreviewCatalogRecord403
    | PreviewCatalogRecord404
    | PreviewCatalogRecord409
    | PreviewCatalogRecord422
    | PreviewCatalogRecord500
    | PreviewCatalogRecord502
  >,
>(
  contentType: string,
  key: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewCatalogRecord>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getPreviewCatalogRecordSuspenseQueryOptions(contentType, key, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetContentTypeEnabledUrl = (contentType: string) => {
  return `/api/v1/content-types/${contentType}/enabled`;
};

/**
 * @summary Turn a Content Type on or off. It can be off only while it holds no Works
 */
export const setContentTypeEnabled = async (
  contentType: string,
  worksEnabled: WorksEnabled,
  options?: Parameters<typeof apiFetch>[1],
): Promise<WorksContentType> => {
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
  return apiFetch<WorksContentType>(getSetContentTypeEnabledUrl(contentType), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(worksEnabled),
  });
};

export const getSetContentTypeEnabledMutationKey = () => ["setContentTypeEnabled"] as const;

export const getSetContentTypeEnabledMutationOptions = <
  TError = ErrorType<
    | SetContentTypeEnabled401
    | SetContentTypeEnabled403
    | SetContentTypeEnabled404
    | SetContentTypeEnabled409
    | SetContentTypeEnabled422
    | SetContentTypeEnabled500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setContentTypeEnabled>>,
    TError,
    SetContentTypeEnabledMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setContentTypeEnabled>>,
  TError,
  SetContentTypeEnabledMutationVariables,
  TContext
> => {
  const mutationKey = getSetContentTypeEnabledMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setContentTypeEnabled>>,
    SetContentTypeEnabledMutationVariables
  > = (props) => {
    const { contentType, data } = props ?? {};

    return setContentTypeEnabled(contentType, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetContentTypeEnabledMutationResult = NonNullable<
  Awaited<ReturnType<typeof setContentTypeEnabled>>
>;
export type SetContentTypeEnabledMutationBody = WorksEnabled;
export type SetContentTypeEnabledMutationError = ErrorType<
  | SetContentTypeEnabled401
  | SetContentTypeEnabled403
  | SetContentTypeEnabled404
  | SetContentTypeEnabled409
  | SetContentTypeEnabled422
  | SetContentTypeEnabled500
>;
export type SetContentTypeEnabledMutationVariables = { contentType: string; data: WorksEnabled };

/**
 * @summary Turn a Content Type on or off. It can be off only while it holds no Works
 */
export const useSetContentTypeEnabled = <
  TError = ErrorType<
    | SetContentTypeEnabled401
    | SetContentTypeEnabled403
    | SetContentTypeEnabled404
    | SetContentTypeEnabled409
    | SetContentTypeEnabled422
    | SetContentTypeEnabled500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setContentTypeEnabled>>,
      TError,
      SetContentTypeEnabledMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setContentTypeEnabled>>,
  TError,
  SetContentTypeEnabledMutationVariables,
  TContext
> => {
  return useMutation(getSetContentTypeEnabledMutationOptions(options), queryClient);
};
export const getGetContentTypeSettingsUrl = (contentType: string) => {
  return `/api/v1/content-types/${contentType}/settings`;
};

/**
 * @summary Get a Content Type's own settings
 */
export const getContentTypeSettings = async (
  contentType: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<WorksTypeSettings> => {
  return apiFetch<WorksTypeSettings>(getGetContentTypeSettingsUrl(contentType), {
    ...options,
    method: "GET",
  });
};

export const getGetContentTypeSettingsQueryKey = (contentType: string) => {
  return [`/api/v1/content-types/${contentType}/settings`] as const;
};

export const getGetContentTypeSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getContentTypeSettings>>,
  TError = ErrorType<
    | GetContentTypeSettings401
    | GetContentTypeSettings403
    | GetContentTypeSettings404
    | GetContentTypeSettings409
    | GetContentTypeSettings422
    | GetContentTypeSettings500
  >,
>(
  contentType: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getContentTypeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetContentTypeSettingsQueryKey(contentType);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getContentTypeSettings>>> = ({ signal }) =>
    getContentTypeSettings(contentType, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof getContentTypeSettings>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetContentTypeSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getContentTypeSettings>>
>;
export type GetContentTypeSettingsSuspenseQueryError = ErrorType<
  | GetContentTypeSettings401
  | GetContentTypeSettings403
  | GetContentTypeSettings404
  | GetContentTypeSettings409
  | GetContentTypeSettings422
  | GetContentTypeSettings500
>;

export function useGetContentTypeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getContentTypeSettings>>,
  TError = ErrorType<
    | GetContentTypeSettings401
    | GetContentTypeSettings403
    | GetContentTypeSettings404
    | GetContentTypeSettings409
    | GetContentTypeSettings422
    | GetContentTypeSettings500
  >,
>(
  contentType: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getContentTypeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetContentTypeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getContentTypeSettings>>,
  TError = ErrorType<
    | GetContentTypeSettings401
    | GetContentTypeSettings403
    | GetContentTypeSettings404
    | GetContentTypeSettings409
    | GetContentTypeSettings422
    | GetContentTypeSettings500
  >,
>(
  contentType: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getContentTypeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetContentTypeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getContentTypeSettings>>,
  TError = ErrorType<
    | GetContentTypeSettings401
    | GetContentTypeSettings403
    | GetContentTypeSettings404
    | GetContentTypeSettings409
    | GetContentTypeSettings422
    | GetContentTypeSettings500
  >,
>(
  contentType: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getContentTypeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Content Type's own settings
 */

export function useGetContentTypeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getContentTypeSettings>>,
  TError = ErrorType<
    | GetContentTypeSettings401
    | GetContentTypeSettings403
    | GetContentTypeSettings404
    | GetContentTypeSettings409
    | GetContentTypeSettings422
    | GetContentTypeSettings500
  >,
>(
  contentType: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getContentTypeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetContentTypeSettingsSuspenseQueryOptions(contentType, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateContentTypeSettingsUrl = (contentType: string) => {
  return `/api/v1/content-types/${contentType}/settings`;
};

/**
 * @summary Change a Content Type's own settings by key, all or none. Every Work of the type may then read differently
 */
export const updateContentTypeSettings = async (
  contentType: string,
  worksSettingsChange: WorksSettingsChange,
  options?: Parameters<typeof apiFetch>[1],
): Promise<WorksTypeSettings> => {
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
  return apiFetch<WorksTypeSettings>(getUpdateContentTypeSettingsUrl(contentType), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(worksSettingsChange),
  });
};

export const getUpdateContentTypeSettingsMutationKey = () => ["updateContentTypeSettings"] as const;

export const getUpdateContentTypeSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateContentTypeSettings401
    | UpdateContentTypeSettings403
    | UpdateContentTypeSettings404
    | UpdateContentTypeSettings409
    | UpdateContentTypeSettings422
    | UpdateContentTypeSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateContentTypeSettings>>,
    TError,
    UpdateContentTypeSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateContentTypeSettings>>,
  TError,
  UpdateContentTypeSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateContentTypeSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateContentTypeSettings>>,
    UpdateContentTypeSettingsMutationVariables
  > = (props) => {
    const { contentType, data } = props ?? {};

    return updateContentTypeSettings(contentType, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateContentTypeSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateContentTypeSettings>>
>;
export type UpdateContentTypeSettingsMutationBody = WorksSettingsChange;
export type UpdateContentTypeSettingsMutationError = ErrorType<
  | UpdateContentTypeSettings401
  | UpdateContentTypeSettings403
  | UpdateContentTypeSettings404
  | UpdateContentTypeSettings409
  | UpdateContentTypeSettings422
  | UpdateContentTypeSettings500
>;
export type UpdateContentTypeSettingsMutationVariables = {
  contentType: string;
  data: WorksSettingsChange;
};

/**
 * @summary Change a Content Type's own settings by key, all or none. Every Work of the type may then read differently
 */
export const useUpdateContentTypeSettings = <
  TError = ErrorType<
    | UpdateContentTypeSettings401
    | UpdateContentTypeSettings403
    | UpdateContentTypeSettings404
    | UpdateContentTypeSettings409
    | UpdateContentTypeSettings422
    | UpdateContentTypeSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateContentTypeSettings>>,
      TError,
      UpdateContentTypeSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateContentTypeSettings>>,
  TError,
  UpdateContentTypeSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateContentTypeSettingsMutationOptions(options), queryClient);
};
export const getListWorksUrl = (params?: ListWorksParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/works?${stringifiedParams}` : `/api/v1/works`;
};

/**
 * @summary List the Works, or one Content Type's
 */
export const listWorks = async (
  params?: ListWorksParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<WorksWorkList> => {
  return apiFetch<WorksWorkList>(getListWorksUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListWorksQueryKey = (params?: ListWorksParams) => {
  return [`/api/v1/works`, ...(params ? [params] : [])] as const;
};

export const getListWorksSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listWorks>>,
  TError = ErrorType<
    ListWorks401 | ListWorks403 | ListWorks404 | ListWorks409 | ListWorks422 | ListWorks500
  >,
>(
  params?: ListWorksParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorks>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListWorksQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listWorks>>> = ({ signal }) =>
    listWorks(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorks>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListWorksSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listWorks>>>;
export type ListWorksSuspenseQueryError = ErrorType<
  ListWorks401 | ListWorks403 | ListWorks404 | ListWorks409 | ListWorks422 | ListWorks500
>;

export function useListWorksSuspense<
  TData = Awaited<ReturnType<typeof listWorks>>,
  TError = ErrorType<
    ListWorks401 | ListWorks403 | ListWorks404 | ListWorks409 | ListWorks422 | ListWorks500
  >,
>(
  params: undefined | ListWorksParams,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorks>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListWorksSuspense<
  TData = Awaited<ReturnType<typeof listWorks>>,
  TError = ErrorType<
    ListWorks401 | ListWorks403 | ListWorks404 | ListWorks409 | ListWorks422 | ListWorks500
  >,
>(
  params?: ListWorksParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorks>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListWorksSuspense<
  TData = Awaited<ReturnType<typeof listWorks>>,
  TError = ErrorType<
    ListWorks401 | ListWorks403 | ListWorks404 | ListWorks409 | ListWorks422 | ListWorks500
  >,
>(
  params?: ListWorksParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorks>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Works, or one Content Type's
 */

export function useListWorksSuspense<
  TData = Awaited<ReturnType<typeof listWorks>>,
  TError = ErrorType<
    ListWorks401 | ListWorks403 | ListWorks404 | ListWorks409 | ListWorks422 | ListWorks500
  >,
>(
  params?: ListWorksParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorks>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListWorksSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getAddWorkUrl = () => {
  return `/api/v1/works`;
};

/**
 * @summary Add a Work by its key with its whole tree, or get the Work already added
 */
export const addWork = async (
  worksNewWork: WorksNewWork,
  options?: Parameters<typeof apiFetch>[1],
): Promise<WorksAdded> => {
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
  return apiFetch<WorksAdded>(getAddWorkUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(worksNewWork),
  });
};

export const getAddWorkMutationKey = () => ["addWork"] as const;

export const getAddWorkMutationOptions = <
  TError = ErrorType<
    AddWork401 | AddWork403 | AddWork404 | AddWork409 | AddWork422 | AddWork500 | AddWork502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof addWork>>,
    TError,
    AddWorkMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof addWork>>,
  TError,
  AddWorkMutationVariables,
  TContext
> => {
  const mutationKey = getAddWorkMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof addWork>>,
    AddWorkMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return addWork(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type AddWorkMutationResult = NonNullable<Awaited<ReturnType<typeof addWork>>>;
export type AddWorkMutationBody = WorksNewWork;
export type AddWorkMutationError = ErrorType<
  AddWork401 | AddWork403 | AddWork404 | AddWork409 | AddWork422 | AddWork500 | AddWork502
>;
export type AddWorkMutationVariables = { data: WorksNewWork };

/**
 * @summary Add a Work by its key with its whole tree, or get the Work already added
 */
export const useAddWork = <
  TError = ErrorType<
    AddWork401 | AddWork403 | AddWork404 | AddWork409 | AddWork422 | AddWork500 | AddWork502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof addWork>>,
      TError,
      AddWorkMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof addWork>>,
  TError,
  AddWorkMutationVariables,
  TContext
> => {
  return useMutation(getAddWorkMutationOptions(options), queryClient);
};
export const getGetWorkUrl = (workId: string) => {
  return `/api/v1/works/${workId}`;
};

/**
 * @summary Get a Work's tree with each node's fields
 */
export const getWork = async (
  workId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<WorksTree> => {
  return apiFetch<WorksTree>(getGetWorkUrl(workId), {
    ...options,
    method: "GET",
  });
};

export const getGetWorkQueryKey = (workId: string) => {
  return [`/api/v1/works/${workId}`] as const;
};

export const getGetWorkSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getWork>>,
  TError = ErrorType<GetWork401 | GetWork403 | GetWork404 | GetWork409 | GetWork422 | GetWork500>,
>(
  workId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWork>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetWorkQueryKey(workId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getWork>>> = ({ signal }) =>
    getWork(workId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWork>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetWorkSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getWork>>>;
export type GetWorkSuspenseQueryError = ErrorType<
  GetWork401 | GetWork403 | GetWork404 | GetWork409 | GetWork422 | GetWork500
>;

export function useGetWorkSuspense<
  TData = Awaited<ReturnType<typeof getWork>>,
  TError = ErrorType<GetWork401 | GetWork403 | GetWork404 | GetWork409 | GetWork422 | GetWork500>,
>(
  workId: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWork>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkSuspense<
  TData = Awaited<ReturnType<typeof getWork>>,
  TError = ErrorType<GetWork401 | GetWork403 | GetWork404 | GetWork409 | GetWork422 | GetWork500>,
>(
  workId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWork>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkSuspense<
  TData = Awaited<ReturnType<typeof getWork>>,
  TError = ErrorType<GetWork401 | GetWork403 | GetWork404 | GetWork409 | GetWork422 | GetWork500>,
>(
  workId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWork>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Work's tree with each node's fields
 */

export function useGetWorkSuspense<
  TData = Awaited<ReturnType<typeof getWork>>,
  TError = ErrorType<GetWork401 | GetWork403 | GetWork404 | GetWork409 | GetWork422 | GetWork500>,
>(
  workId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWork>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetWorkSuspenseQueryOptions(workId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getRefreshWorkUrl = (workId: string) => {
  return `/api/v1/works/${workId}/refresh`;
};

/**
 * @summary Refresh a Work from its metadata source, keeping every id
 */
export const refreshWork = async (
  workId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<WorksTree> => {
  return apiFetch<WorksTree>(getRefreshWorkUrl(workId), {
    ...options,
    method: "POST",
  });
};

export const getRefreshWorkMutationKey = () => ["refreshWork"] as const;

export const getRefreshWorkMutationOptions = <
  TError = ErrorType<
    | RefreshWork401
    | RefreshWork403
    | RefreshWork404
    | RefreshWork409
    | RefreshWork422
    | RefreshWork500
    | RefreshWork502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof refreshWork>>,
    TError,
    RefreshWorkMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof refreshWork>>,
  TError,
  RefreshWorkMutationVariables,
  TContext
> => {
  const mutationKey = getRefreshWorkMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof refreshWork>>,
    RefreshWorkMutationVariables
  > = (props) => {
    const { workId } = props ?? {};

    return refreshWork(workId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RefreshWorkMutationResult = NonNullable<Awaited<ReturnType<typeof refreshWork>>>;

export type RefreshWorkMutationError = ErrorType<
  | RefreshWork401
  | RefreshWork403
  | RefreshWork404
  | RefreshWork409
  | RefreshWork422
  | RefreshWork500
  | RefreshWork502
>;
export type RefreshWorkMutationVariables = { workId: string };

/**
 * @summary Refresh a Work from its metadata source, keeping every id
 */
export const useRefreshWork = <
  TError = ErrorType<
    | RefreshWork401
    | RefreshWork403
    | RefreshWork404
    | RefreshWork409
    | RefreshWork422
    | RefreshWork500
    | RefreshWork502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof refreshWork>>,
      TError,
      RefreshWorkMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof refreshWork>>,
  TError,
  RefreshWorkMutationVariables,
  TContext
> => {
  return useMutation(getRefreshWorkMutationOptions(options), queryClient);
};
