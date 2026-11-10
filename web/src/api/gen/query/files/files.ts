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
  CountMoving401,
  CountMoving403,
  CountMoving409,
  CountMoving422,
  CountMoving500,
  CountMovingParams,
  CountUnmatched401,
  CountUnmatched403,
  CountUnmatched409,
  CountUnmatched500,
  FilesCellList,
  FilesFileList,
  FilesRenamePlan,
  FilesRenameResult,
  FilesWorkRenamePlan,
  ForgetFiles401,
  ForgetFiles403,
  ForgetFiles404,
  ForgetFiles409,
  ForgetFiles422,
  ForgetFiles500,
  ForgetInputBody,
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
  MovingOutputBody,
  PlanInputBody,
  PlanRenames401,
  PlanRenames403,
  PlanRenames404,
  PlanRenames409,
  PlanRenames422,
  PlanRenames500,
  PlanWorkRenames401,
  PlanWorkRenames403,
  PlanWorkRenames404,
  PlanWorkRenames409,
  PlanWorkRenames422,
  PlanWorkRenames500,
  RenameFiles401,
  RenameFiles403,
  RenameFiles409,
  RenameFiles422,
  RenameFiles500,
  RenameInputBody,
  UnmatchedOutputBody,
  WorkPlanInputBody,
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

export const getForgetFilesUrl = () => {
  return `/api/v1/files/forget`;
};

/**
 * @summary Delete the records of gone files, all or none
 */
export const forgetFiles = async (
  forgetInputBody: ForgetInputBody,
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
  return apiFetch<void>(getForgetFilesUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(forgetInputBody),
  });
};

export const getForgetFilesMutationKey = () => ["forgetFiles"] as const;

export const getForgetFilesMutationOptions = <
  TError = ErrorType<
    | ForgetFiles401
    | ForgetFiles403
    | ForgetFiles404
    | ForgetFiles409
    | ForgetFiles422
    | ForgetFiles500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof forgetFiles>>,
    TError,
    ForgetFilesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof forgetFiles>>,
  TError,
  ForgetFilesMutationVariables,
  TContext
> => {
  const mutationKey = getForgetFilesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof forgetFiles>>,
    ForgetFilesMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return forgetFiles(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ForgetFilesMutationResult = NonNullable<Awaited<ReturnType<typeof forgetFiles>>>;
export type ForgetFilesMutationBody = ForgetInputBody;
export type ForgetFilesMutationError = ErrorType<
  | ForgetFiles401
  | ForgetFiles403
  | ForgetFiles404
  | ForgetFiles409
  | ForgetFiles422
  | ForgetFiles500
>;
export type ForgetFilesMutationVariables = { data: ForgetInputBody };

/**
 * @summary Delete the records of gone files, all or none
 */
export const useForgetFiles = <
  TError = ErrorType<
    | ForgetFiles401
    | ForgetFiles403
    | ForgetFiles404
    | ForgetFiles409
    | ForgetFiles422
    | ForgetFiles500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof forgetFiles>>,
      TError,
      ForgetFilesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof forgetFiles>>,
  TError,
  ForgetFilesMutationVariables,
  TContext
> => {
  return useMutation(getForgetFilesMutationOptions(options), queryClient);
};
export const getRenameFilesUrl = () => {
  return `/api/v1/renames`;
};

/**
 * @summary Rename the files of each Work whose plan still has the digest shown, and list the Works left because their plans differ now
 */
export const renameFiles = async (
  renameInputBody: RenameInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FilesRenameResult> => {
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
  return apiFetch<FilesRenameResult>(getRenameFilesUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(renameInputBody),
  });
};

export const getRenameFilesMutationKey = () => ["renameFiles"] as const;

export const getRenameFilesMutationOptions = <
  TError = ErrorType<
    RenameFiles401 | RenameFiles403 | RenameFiles409 | RenameFiles422 | RenameFiles500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof renameFiles>>,
    TError,
    RenameFilesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof renameFiles>>,
  TError,
  RenameFilesMutationVariables,
  TContext
> => {
  const mutationKey = getRenameFilesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof renameFiles>>,
    RenameFilesMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return renameFiles(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RenameFilesMutationResult = NonNullable<Awaited<ReturnType<typeof renameFiles>>>;
export type RenameFilesMutationBody = RenameInputBody;
export type RenameFilesMutationError = ErrorType<
  RenameFiles401 | RenameFiles403 | RenameFiles409 | RenameFiles422 | RenameFiles500
>;
export type RenameFilesMutationVariables = { data: RenameInputBody };

/**
 * @summary Rename the files of each Work whose plan still has the digest shown, and list the Works left because their plans differ now
 */
export const useRenameFiles = <
  TError = ErrorType<
    RenameFiles401 | RenameFiles403 | RenameFiles409 | RenameFiles422 | RenameFiles500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof renameFiles>>,
      TError,
      RenameFilesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof renameFiles>>,
  TError,
  RenameFilesMutationVariables,
  TContext
> => {
  return useMutation(getRenameFilesMutationOptions(options), queryClient);
};
export const getCountMovingUrl = (params?: CountMovingParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/renames/moving?${stringifiedParams}`
    : `/api/v1/renames/moving`;
};

/**
 * @summary Count the files the disk queue still moves, in a Work, a Content Type or the library
 */
export const countMoving = async (
  params?: CountMovingParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovingOutputBody> => {
  return apiFetch<MovingOutputBody>(getCountMovingUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getCountMovingQueryKey = (params?: CountMovingParams) => {
  return [`/api/v1/renames/moving`, ...(params ? [params] : [])] as const;
};

export const getCountMovingSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof countMoving>>,
  TError = ErrorType<
    CountMoving401 | CountMoving403 | CountMoving409 | CountMoving422 | CountMoving500
  >,
>(
  params?: CountMovingParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof countMoving>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getCountMovingQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof countMoving>>> = ({ signal }) =>
    countMoving(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof countMoving>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type CountMovingSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof countMoving>>>;
export type CountMovingSuspenseQueryError = ErrorType<
  CountMoving401 | CountMoving403 | CountMoving409 | CountMoving422 | CountMoving500
>;

export function useCountMovingSuspense<
  TData = Awaited<ReturnType<typeof countMoving>>,
  TError = ErrorType<
    CountMoving401 | CountMoving403 | CountMoving409 | CountMoving422 | CountMoving500
  >,
>(
  params: undefined | CountMovingParams,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof countMoving>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useCountMovingSuspense<
  TData = Awaited<ReturnType<typeof countMoving>>,
  TError = ErrorType<
    CountMoving401 | CountMoving403 | CountMoving409 | CountMoving422 | CountMoving500
  >,
>(
  params?: CountMovingParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof countMoving>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useCountMovingSuspense<
  TData = Awaited<ReturnType<typeof countMoving>>,
  TError = ErrorType<
    CountMoving401 | CountMoving403 | CountMoving409 | CountMoving422 | CountMoving500
  >,
>(
  params?: CountMovingParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof countMoving>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Count the files the disk queue still moves, in a Work, a Content Type or the library
 */

export function useCountMovingSuspense<
  TData = Awaited<ReturnType<typeof countMoving>>,
  TError = ErrorType<
    CountMoving401 | CountMoving403 | CountMoving409 | CountMoving422 | CountMoving500
  >,
>(
  params?: CountMovingParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof countMoving>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getCountMovingSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getPlanRenamesUrl = () => {
  return `/api/v1/renames/plan`;
};

/**
 * @summary Plan the renames of a Work, a Content Type or the library, each Work's counts and digest, under the stored naming settings or a draft
 */
export const planRenames = async (
  planInputBody: PlanInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FilesRenamePlan> => {
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
  return apiFetch<FilesRenamePlan>(getPlanRenamesUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(planInputBody),
  });
};

export const getPlanRenamesMutationKey = () => ["planRenames"] as const;

export const getPlanRenamesMutationOptions = <
  TError = ErrorType<
    | PlanRenames401
    | PlanRenames403
    | PlanRenames404
    | PlanRenames409
    | PlanRenames422
    | PlanRenames500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof planRenames>>,
    TError,
    PlanRenamesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof planRenames>>,
  TError,
  PlanRenamesMutationVariables,
  TContext
> => {
  const mutationKey = getPlanRenamesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof planRenames>>,
    PlanRenamesMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return planRenames(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PlanRenamesMutationResult = NonNullable<Awaited<ReturnType<typeof planRenames>>>;
export type PlanRenamesMutationBody = PlanInputBody;
export type PlanRenamesMutationError = ErrorType<
  | PlanRenames401
  | PlanRenames403
  | PlanRenames404
  | PlanRenames409
  | PlanRenames422
  | PlanRenames500
>;
export type PlanRenamesMutationVariables = { data: PlanInputBody };

/**
 * @summary Plan the renames of a Work, a Content Type or the library, each Work's counts and digest, under the stored naming settings or a draft
 */
export const usePlanRenames = <
  TError = ErrorType<
    | PlanRenames401
    | PlanRenames403
    | PlanRenames404
    | PlanRenames409
    | PlanRenames422
    | PlanRenames500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof planRenames>>,
      TError,
      PlanRenamesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof planRenames>>,
  TError,
  PlanRenamesMutationVariables,
  TContext
> => {
  return useMutation(getPlanRenamesMutationOptions(options), queryClient);
};
export const getPlanWorkRenamesUrl = (workId: string) => {
  return `/api/v1/renames/plan/${workId}`;
};

/**
 * @summary Plan the renames of one Work, each file's row with its companions, under the stored naming settings or a draft
 */
export const planWorkRenames = async (
  workId: string,
  workPlanInputBody?: WorkPlanInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FilesWorkRenamePlan> => {
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
  return apiFetch<FilesWorkRenamePlan>(getPlanWorkRenamesUrl(workId), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(workPlanInputBody),
  });
};

export const getPlanWorkRenamesMutationKey = () => ["planWorkRenames"] as const;

export const getPlanWorkRenamesMutationOptions = <
  TError = ErrorType<
    | PlanWorkRenames401
    | PlanWorkRenames403
    | PlanWorkRenames404
    | PlanWorkRenames409
    | PlanWorkRenames422
    | PlanWorkRenames500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof planWorkRenames>>,
    TError,
    PlanWorkRenamesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof planWorkRenames>>,
  TError,
  PlanWorkRenamesMutationVariables,
  TContext
> => {
  const mutationKey = getPlanWorkRenamesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof planWorkRenames>>,
    PlanWorkRenamesMutationVariables
  > = (props) => {
    const { workId, data } = props ?? {};

    return planWorkRenames(workId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PlanWorkRenamesMutationResult = NonNullable<
  Awaited<ReturnType<typeof planWorkRenames>>
>;
export type PlanWorkRenamesMutationBody = WorkPlanInputBody | undefined;
export type PlanWorkRenamesMutationError = ErrorType<
  | PlanWorkRenames401
  | PlanWorkRenames403
  | PlanWorkRenames404
  | PlanWorkRenames409
  | PlanWorkRenames422
  | PlanWorkRenames500
>;
export type PlanWorkRenamesMutationVariables = { workId: string; data?: WorkPlanInputBody };

/**
 * @summary Plan the renames of one Work, each file's row with its companions, under the stored naming settings or a draft
 */
export const usePlanWorkRenames = <
  TError = ErrorType<
    | PlanWorkRenames401
    | PlanWorkRenames403
    | PlanWorkRenames404
    | PlanWorkRenames409
    | PlanWorkRenames422
    | PlanWorkRenames500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof planWorkRenames>>,
      TError,
      PlanWorkRenamesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof planWorkRenames>>,
  TError,
  PlanWorkRenamesMutationVariables,
  TContext
> => {
  return useMutation(getPlanWorkRenamesMutationOptions(options), queryClient);
};
export const getCountUnmatchedUrl = () => {
  return `/api/v1/renames/unmatched`;
};

/**
 * @summary Count, per Content Type, the files whose path differs from what the naming templates make
 */
export const countUnmatched = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<UnmatchedOutputBody> => {
  return apiFetch<UnmatchedOutputBody>(getCountUnmatchedUrl(), {
    ...options,
    method: "GET",
  });
};

export const getCountUnmatchedQueryKey = () => {
  return [`/api/v1/renames/unmatched`] as const;
};

export const getCountUnmatchedSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof countUnmatched>>,
  TError = ErrorType<CountUnmatched401 | CountUnmatched403 | CountUnmatched409 | CountUnmatched500>,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof countUnmatched>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getCountUnmatchedQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof countUnmatched>>> = ({ signal }) =>
    countUnmatched({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof countUnmatched>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type CountUnmatchedSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof countUnmatched>>
>;
export type CountUnmatchedSuspenseQueryError = ErrorType<
  CountUnmatched401 | CountUnmatched403 | CountUnmatched409 | CountUnmatched500
>;

export function useCountUnmatchedSuspense<
  TData = Awaited<ReturnType<typeof countUnmatched>>,
  TError = ErrorType<CountUnmatched401 | CountUnmatched403 | CountUnmatched409 | CountUnmatched500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof countUnmatched>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useCountUnmatchedSuspense<
  TData = Awaited<ReturnType<typeof countUnmatched>>,
  TError = ErrorType<CountUnmatched401 | CountUnmatched403 | CountUnmatched409 | CountUnmatched500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof countUnmatched>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useCountUnmatchedSuspense<
  TData = Awaited<ReturnType<typeof countUnmatched>>,
  TError = ErrorType<CountUnmatched401 | CountUnmatched403 | CountUnmatched409 | CountUnmatched500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof countUnmatched>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Count, per Content Type, the files whose path differs from what the naming templates make
 */

export function useCountUnmatchedSuspense<
  TData = Awaited<ReturnType<typeof countUnmatched>>,
  TError = ErrorType<CountUnmatched401 | CountUnmatched403 | CountUnmatched409 | CountUnmatched500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof countUnmatched>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getCountUnmatchedSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

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
