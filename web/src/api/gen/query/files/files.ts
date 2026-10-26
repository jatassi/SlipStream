import { queryOptions as queryOptionsBuilder, useSuspenseQuery } from "@tanstack/react-query";
import type {
  DataTag,
  QueryClient,
  QueryFunction,
  QueryKey,
  UseSuspenseQueryOptions,
  UseSuspenseQueryResult,
} from "@tanstack/react-query";

import type {
  FilesCellList,
  FilesFileList,
  GetWorkCells401,
  GetWorkCells403,
  GetWorkCells404,
  GetWorkCells409,
  GetWorkCells422,
  GetWorkCells500,
  ListWorkFiles401,
  ListWorkFiles403,
  ListWorkFiles404,
  ListWorkFiles409,
  ListWorkFiles422,
  ListWorkFiles500,
  ListWorkFiles501,
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

export const getGetWorkCellsUrl = (workId: string) => {
  return `/api/v1/works/${workId}/cells`;
};

/**
 * @summary Get each of a Work's cells with its state and the file filling it, and each Item's status
 */
export const getWorkCells = async (
  workId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FilesCellList> => {
  return apiFetch<FilesCellList>(getGetWorkCellsUrl(workId), {
    ...options,
    method: "GET",
  });
};

export const getGetWorkCellsQueryKey = (workId: string) => {
  return [`/api/v1/works/${workId}/cells`] as const;
};

export const getGetWorkCellsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getWorkCells>>,
  TError = ErrorType<
    | GetWorkCells401
    | GetWorkCells403
    | GetWorkCells404
    | GetWorkCells409
    | GetWorkCells422
    | GetWorkCells500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkCells>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetWorkCellsQueryKey(workId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getWorkCells>>> = ({ signal }) =>
    getWorkCells(workId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkCells>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetWorkCellsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getWorkCells>>>;
export type GetWorkCellsSuspenseQueryError = ErrorType<
  | GetWorkCells401
  | GetWorkCells403
  | GetWorkCells404
  | GetWorkCells409
  | GetWorkCells422
  | GetWorkCells500
>;

export function useGetWorkCellsSuspense<
  TData = Awaited<ReturnType<typeof getWorkCells>>,
  TError = ErrorType<
    | GetWorkCells401
    | GetWorkCells403
    | GetWorkCells404
    | GetWorkCells409
    | GetWorkCells422
    | GetWorkCells500
  >,
>(
  workId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkCells>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkCellsSuspense<
  TData = Awaited<ReturnType<typeof getWorkCells>>,
  TError = ErrorType<
    | GetWorkCells401
    | GetWorkCells403
    | GetWorkCells404
    | GetWorkCells409
    | GetWorkCells422
    | GetWorkCells500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkCells>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkCellsSuspense<
  TData = Awaited<ReturnType<typeof getWorkCells>>,
  TError = ErrorType<
    | GetWorkCells401
    | GetWorkCells403
    | GetWorkCells404
    | GetWorkCells409
    | GetWorkCells422
    | GetWorkCells500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkCells>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get each of a Work's cells with its state and the file filling it, and each Item's status
 */

export function useGetWorkCellsSuspense<
  TData = Awaited<ReturnType<typeof getWorkCells>>,
  TError = ErrorType<
    | GetWorkCells401
    | GetWorkCells403
    | GetWorkCells404
    | GetWorkCells409
    | GetWorkCells422
    | GetWorkCells500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkCells>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetWorkCellsSuspenseQueryOptions(workId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getListWorkFilesUrl = (workId: string) => {
  return `/api/v1/works/${workId}/files`;
};

/**
 * @summary List a Work's library files with the cells each fills, and the folder they sit in
 */
export const listWorkFiles = async (
  workId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FilesFileList> => {
  return apiFetch<FilesFileList>(getListWorkFilesUrl(workId), {
    ...options,
    method: "GET",
  });
};

export const getListWorkFilesQueryKey = (workId: string) => {
  return [`/api/v1/works/${workId}/files`] as const;
};

export const getListWorkFilesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listWorkFiles>>,
  TError = ErrorType<
    | ListWorkFiles401
    | ListWorkFiles403
    | ListWorkFiles404
    | ListWorkFiles409
    | ListWorkFiles422
    | ListWorkFiles500
    | ListWorkFiles501
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorkFiles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListWorkFilesQueryKey(workId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listWorkFiles>>> = ({ signal }) =>
    listWorkFiles(workId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorkFiles>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListWorkFilesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listWorkFiles>>
>;
export type ListWorkFilesSuspenseQueryError = ErrorType<
  | ListWorkFiles401
  | ListWorkFiles403
  | ListWorkFiles404
  | ListWorkFiles409
  | ListWorkFiles422
  | ListWorkFiles500
  | ListWorkFiles501
>;

export function useListWorkFilesSuspense<
  TData = Awaited<ReturnType<typeof listWorkFiles>>,
  TError = ErrorType<
    | ListWorkFiles401
    | ListWorkFiles403
    | ListWorkFiles404
    | ListWorkFiles409
    | ListWorkFiles422
    | ListWorkFiles500
    | ListWorkFiles501
  >,
>(
  workId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorkFiles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListWorkFilesSuspense<
  TData = Awaited<ReturnType<typeof listWorkFiles>>,
  TError = ErrorType<
    | ListWorkFiles401
    | ListWorkFiles403
    | ListWorkFiles404
    | ListWorkFiles409
    | ListWorkFiles422
    | ListWorkFiles500
    | ListWorkFiles501
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorkFiles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListWorkFilesSuspense<
  TData = Awaited<ReturnType<typeof listWorkFiles>>,
  TError = ErrorType<
    | ListWorkFiles401
    | ListWorkFiles403
    | ListWorkFiles404
    | ListWorkFiles409
    | ListWorkFiles422
    | ListWorkFiles500
    | ListWorkFiles501
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorkFiles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List a Work's library files with the cells each fills, and the folder they sit in
 */

export function useListWorkFilesSuspense<
  TData = Awaited<ReturnType<typeof listWorkFiles>>,
  TError = ErrorType<
    | ListWorkFiles401
    | ListWorkFiles403
    | ListWorkFiles404
    | ListWorkFiles409
    | ListWorkFiles422
    | ListWorkFiles500
    | ListWorkFiles501
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listWorkFiles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListWorkFilesSuspenseQueryOptions(workId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
