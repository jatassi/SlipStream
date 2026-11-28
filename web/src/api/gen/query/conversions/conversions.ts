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
  CancelConversion401,
  CancelConversion403,
  CancelConversion404,
  CancelConversion409,
  CancelConversion422,
  CancelConversion500,
  CancelConversionGroup401,
  CancelConversionGroup403,
  CancelConversionGroup404,
  CancelConversionGroup409,
  CancelConversionGroup422,
  CancelConversionGroup500,
  CanceledOutputBody,
  ConversionsConversionList,
  ConversionsConversionRow,
  ConversionsGroupPage,
  ConversionsNodeConverting,
  ConversionsNodeSettings,
  GetConversionGroup401,
  GetConversionGroup403,
  GetConversionGroup404,
  GetConversionGroup409,
  GetConversionGroup422,
  GetConversionGroup500,
  GetConversionGroupParams,
  GetNodeConverting401,
  GetNodeConverting403,
  GetNodeConverting404,
  GetNodeConverting409,
  GetNodeConverting422,
  GetNodeConverting500,
  ListConversions401,
  ListConversions403,
  ListConversions409,
  ListConversions422,
  ListConversions500,
  ListConversionsParams,
  MoveConversion401,
  MoveConversion403,
  MoveConversion404,
  MoveConversion409,
  MoveConversion422,
  MoveConversion500,
  MoveConversionUp401,
  MoveConversionUp403,
  MoveConversionUp404,
  MoveConversionUp409,
  MoveConversionUp422,
  MoveConversionUp500,
  MoveInputBody,
  RerunNodeTests401,
  RerunNodeTests403,
  RerunNodeTests404,
  RerunNodeTests409,
  RerunNodeTests422,
  RerunNodeTests500,
  RetryConversion401,
  RetryConversion403,
  RetryConversion404,
  RetryConversion409,
  RetryConversion422,
  RetryConversion500,
  UpdateNodeConverting401,
  UpdateNodeConverting403,
  UpdateNodeConverting404,
  UpdateNodeConverting409,
  UpdateNodeConverting422,
  UpdateNodeConverting500,
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

export const getListConversionsUrl = (params?: ListConversionsParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/conversions?${stringifiedParams}`
    : `/api/v1/conversions`;
};

/**
 * @summary List the Conversions tab: its counts, the ones that need attention, the ones converting, and a page of Up next by lane
 */
export const listConversions = async (
  params?: ListConversionsParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ConversionsConversionList> => {
  return apiFetch<ConversionsConversionList>(getListConversionsUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListConversionsQueryKey = (params?: ListConversionsParams) => {
  return [`/api/v1/conversions`, ...(params ? [params] : [])] as const;
};

export const getListConversionsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listConversions>>,
  TError = ErrorType<
    | ListConversions401
    | ListConversions403
    | ListConversions409
    | ListConversions422
    | ListConversions500
  >,
>(
  params?: ListConversionsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listConversions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListConversionsQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listConversions>>> = ({ signal }) =>
    listConversions(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listConversions>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListConversionsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listConversions>>
>;
export type ListConversionsSuspenseQueryError = ErrorType<
  | ListConversions401
  | ListConversions403
  | ListConversions409
  | ListConversions422
  | ListConversions500
>;

export function useListConversionsSuspense<
  TData = Awaited<ReturnType<typeof listConversions>>,
  TError = ErrorType<
    | ListConversions401
    | ListConversions403
    | ListConversions409
    | ListConversions422
    | ListConversions500
  >,
>(
  params: undefined | ListConversionsParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listConversions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListConversionsSuspense<
  TData = Awaited<ReturnType<typeof listConversions>>,
  TError = ErrorType<
    | ListConversions401
    | ListConversions403
    | ListConversions409
    | ListConversions422
    | ListConversions500
  >,
>(
  params?: ListConversionsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listConversions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListConversionsSuspense<
  TData = Awaited<ReturnType<typeof listConversions>>,
  TError = ErrorType<
    | ListConversions401
    | ListConversions403
    | ListConversions409
    | ListConversions422
    | ListConversions500
  >,
>(
  params?: ListConversionsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listConversions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Conversions tab: its counts, the ones that need attention, the ones converting, and a page of Up next by lane
 */

export function useListConversionsSuspense<
  TData = Awaited<ReturnType<typeof listConversions>>,
  TError = ErrorType<
    | ListConversions401
    | ListConversions403
    | ListConversions409
    | ListConversions422
    | ListConversions500
  >,
>(
  params?: ListConversionsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listConversions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListConversionsSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetConversionGroupUrl = (groupId: string, params?: GetConversionGroupParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/conversions/groups/${groupId}?${stringifiedParams}`
    : `/api/v1/conversions/groups/${groupId}`;
};

/**
 * @summary Get a page of a folded group's Conversions
 */
export const getConversionGroup = async (
  groupId: string,
  params?: GetConversionGroupParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ConversionsGroupPage> => {
  return apiFetch<ConversionsGroupPage>(getGetConversionGroupUrl(groupId, params), {
    ...options,
    method: "GET",
  });
};

export const getGetConversionGroupQueryKey = (
  groupId: string,
  params?: GetConversionGroupParams,
) => {
  return [`/api/v1/conversions/groups/${groupId}`, ...(params ? [params] : [])] as const;
};

export const getGetConversionGroupSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getConversionGroup>>,
  TError = ErrorType<
    | GetConversionGroup401
    | GetConversionGroup403
    | GetConversionGroup404
    | GetConversionGroup409
    | GetConversionGroup422
    | GetConversionGroup500
  >,
>(
  groupId: string,
  params?: GetConversionGroupParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getConversionGroup>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetConversionGroupQueryKey(groupId, params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getConversionGroup>>> = ({ signal }) =>
    getConversionGroup(groupId, params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getConversionGroup>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetConversionGroupSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getConversionGroup>>
>;
export type GetConversionGroupSuspenseQueryError = ErrorType<
  | GetConversionGroup401
  | GetConversionGroup403
  | GetConversionGroup404
  | GetConversionGroup409
  | GetConversionGroup422
  | GetConversionGroup500
>;

export function useGetConversionGroupSuspense<
  TData = Awaited<ReturnType<typeof getConversionGroup>>,
  TError = ErrorType<
    | GetConversionGroup401
    | GetConversionGroup403
    | GetConversionGroup404
    | GetConversionGroup409
    | GetConversionGroup422
    | GetConversionGroup500
  >,
>(
  groupId: string,
  params: undefined | GetConversionGroupParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getConversionGroup>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetConversionGroupSuspense<
  TData = Awaited<ReturnType<typeof getConversionGroup>>,
  TError = ErrorType<
    | GetConversionGroup401
    | GetConversionGroup403
    | GetConversionGroup404
    | GetConversionGroup409
    | GetConversionGroup422
    | GetConversionGroup500
  >,
>(
  groupId: string,
  params?: GetConversionGroupParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getConversionGroup>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetConversionGroupSuspense<
  TData = Awaited<ReturnType<typeof getConversionGroup>>,
  TError = ErrorType<
    | GetConversionGroup401
    | GetConversionGroup403
    | GetConversionGroup404
    | GetConversionGroup409
    | GetConversionGroup422
    | GetConversionGroup500
  >,
>(
  groupId: string,
  params?: GetConversionGroupParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getConversionGroup>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a page of a folded group's Conversions
 */

export function useGetConversionGroupSuspense<
  TData = Awaited<ReturnType<typeof getConversionGroup>>,
  TError = ErrorType<
    | GetConversionGroup401
    | GetConversionGroup403
    | GetConversionGroup404
    | GetConversionGroup409
    | GetConversionGroup422
    | GetConversionGroup500
  >,
>(
  groupId: string,
  params?: GetConversionGroupParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getConversionGroup>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetConversionGroupSuspenseQueryOptions(groupId, params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCancelConversionGroupUrl = (groupId: string) => {
  return `/api/v1/conversions/groups/${groupId}/cancel`;
};

/**
 * @summary Cancel every unfinished Conversion of a folded group
 */
export const cancelConversionGroup = async (
  groupId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<CanceledOutputBody> => {
  return apiFetch<CanceledOutputBody>(getCancelConversionGroupUrl(groupId), {
    ...options,
    method: "POST",
  });
};

export const getCancelConversionGroupMutationKey = () => ["cancelConversionGroup"] as const;

export const getCancelConversionGroupMutationOptions = <
  TError = ErrorType<
    | CancelConversionGroup401
    | CancelConversionGroup403
    | CancelConversionGroup404
    | CancelConversionGroup409
    | CancelConversionGroup422
    | CancelConversionGroup500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof cancelConversionGroup>>,
    TError,
    CancelConversionGroupMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof cancelConversionGroup>>,
  TError,
  CancelConversionGroupMutationVariables,
  TContext
> => {
  const mutationKey = getCancelConversionGroupMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof cancelConversionGroup>>,
    CancelConversionGroupMutationVariables
  > = (props) => {
    const { groupId } = props ?? {};

    return cancelConversionGroup(groupId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CancelConversionGroupMutationResult = NonNullable<
  Awaited<ReturnType<typeof cancelConversionGroup>>
>;

export type CancelConversionGroupMutationError = ErrorType<
  | CancelConversionGroup401
  | CancelConversionGroup403
  | CancelConversionGroup404
  | CancelConversionGroup409
  | CancelConversionGroup422
  | CancelConversionGroup500
>;
export type CancelConversionGroupMutationVariables = { groupId: string };

/**
 * @summary Cancel every unfinished Conversion of a folded group
 */
export const useCancelConversionGroup = <
  TError = ErrorType<
    | CancelConversionGroup401
    | CancelConversionGroup403
    | CancelConversionGroup404
    | CancelConversionGroup409
    | CancelConversionGroup422
    | CancelConversionGroup500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof cancelConversionGroup>>,
      TError,
      CancelConversionGroupMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof cancelConversionGroup>>,
  TError,
  CancelConversionGroupMutationVariables,
  TContext
> => {
  return useMutation(getCancelConversionGroupMutationOptions(options), queryClient);
};
export const getCancelConversionUrl = (id: string) => {
  return `/api/v1/conversions/${id}/cancel`;
};

/**
 * @summary Cancel an unfinished Conversion. SlipStream doesn't queue it again until its source file changes or an Admin asks for it
 */
export const cancelConversion = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ConversionsConversionRow> => {
  return apiFetch<ConversionsConversionRow>(getCancelConversionUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getCancelConversionMutationKey = () => ["cancelConversion"] as const;

export const getCancelConversionMutationOptions = <
  TError = ErrorType<
    | CancelConversion401
    | CancelConversion403
    | CancelConversion404
    | CancelConversion409
    | CancelConversion422
    | CancelConversion500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof cancelConversion>>,
    TError,
    CancelConversionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof cancelConversion>>,
  TError,
  CancelConversionMutationVariables,
  TContext
> => {
  const mutationKey = getCancelConversionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof cancelConversion>>,
    CancelConversionMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return cancelConversion(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CancelConversionMutationResult = NonNullable<
  Awaited<ReturnType<typeof cancelConversion>>
>;

export type CancelConversionMutationError = ErrorType<
  | CancelConversion401
  | CancelConversion403
  | CancelConversion404
  | CancelConversion409
  | CancelConversion422
  | CancelConversion500
>;
export type CancelConversionMutationVariables = { id: string };

/**
 * @summary Cancel an unfinished Conversion. SlipStream doesn't queue it again until its source file changes or an Admin asks for it
 */
export const useCancelConversion = <
  TError = ErrorType<
    | CancelConversion401
    | CancelConversion403
    | CancelConversion404
    | CancelConversion409
    | CancelConversion422
    | CancelConversion500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof cancelConversion>>,
      TError,
      CancelConversionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof cancelConversion>>,
  TError,
  CancelConversionMutationVariables,
  TContext
> => {
  return useMutation(getCancelConversionMutationOptions(options), queryClient);
};
export const getMoveConversionUrl = (id: string) => {
  return `/api/v1/conversions/${id}/move`;
};

/**
 * @summary Put a waiting Conversion right after another in its lane
 */
export const moveConversion = async (
  id: string,
  moveInputBody: MoveInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ConversionsConversionRow> => {
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
  return apiFetch<ConversionsConversionRow>(getMoveConversionUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(moveInputBody),
  });
};

export const getMoveConversionMutationKey = () => ["moveConversion"] as const;

export const getMoveConversionMutationOptions = <
  TError = ErrorType<
    | MoveConversion401
    | MoveConversion403
    | MoveConversion404
    | MoveConversion409
    | MoveConversion422
    | MoveConversion500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof moveConversion>>,
    TError,
    MoveConversionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof moveConversion>>,
  TError,
  MoveConversionMutationVariables,
  TContext
> => {
  const mutationKey = getMoveConversionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof moveConversion>>,
    MoveConversionMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return moveConversion(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MoveConversionMutationResult = NonNullable<Awaited<ReturnType<typeof moveConversion>>>;
export type MoveConversionMutationBody = MoveInputBody;
export type MoveConversionMutationError = ErrorType<
  | MoveConversion401
  | MoveConversion403
  | MoveConversion404
  | MoveConversion409
  | MoveConversion422
  | MoveConversion500
>;
export type MoveConversionMutationVariables = { id: string; data: MoveInputBody };

/**
 * @summary Put a waiting Conversion right after another in its lane
 */
export const useMoveConversion = <
  TError = ErrorType<
    | MoveConversion401
    | MoveConversion403
    | MoveConversion404
    | MoveConversion409
    | MoveConversion422
    | MoveConversion500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof moveConversion>>,
      TError,
      MoveConversionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof moveConversion>>,
  TError,
  MoveConversionMutationVariables,
  TContext
> => {
  return useMutation(getMoveConversionMutationOptions(options), queryClient);
};
export const getMoveConversionUpUrl = (id: string) => {
  return `/api/v1/conversions/${id}/move-up`;
};

/**
 * @summary Put a waiting Conversion at the head of Moved up
 */
export const moveConversionUp = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ConversionsConversionRow> => {
  return apiFetch<ConversionsConversionRow>(getMoveConversionUpUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getMoveConversionUpMutationKey = () => ["moveConversionUp"] as const;

export const getMoveConversionUpMutationOptions = <
  TError = ErrorType<
    | MoveConversionUp401
    | MoveConversionUp403
    | MoveConversionUp404
    | MoveConversionUp409
    | MoveConversionUp422
    | MoveConversionUp500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof moveConversionUp>>,
    TError,
    MoveConversionUpMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof moveConversionUp>>,
  TError,
  MoveConversionUpMutationVariables,
  TContext
> => {
  const mutationKey = getMoveConversionUpMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof moveConversionUp>>,
    MoveConversionUpMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return moveConversionUp(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MoveConversionUpMutationResult = NonNullable<
  Awaited<ReturnType<typeof moveConversionUp>>
>;

export type MoveConversionUpMutationError = ErrorType<
  | MoveConversionUp401
  | MoveConversionUp403
  | MoveConversionUp404
  | MoveConversionUp409
  | MoveConversionUp422
  | MoveConversionUp500
>;
export type MoveConversionUpMutationVariables = { id: string };

/**
 * @summary Put a waiting Conversion at the head of Moved up
 */
export const useMoveConversionUp = <
  TError = ErrorType<
    | MoveConversionUp401
    | MoveConversionUp403
    | MoveConversionUp404
    | MoveConversionUp409
    | MoveConversionUp422
    | MoveConversionUp500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof moveConversionUp>>,
      TError,
      MoveConversionUpMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof moveConversionUp>>,
  TError,
  MoveConversionUpMutationVariables,
  TContext
> => {
  return useMutation(getMoveConversionUpMutationOptions(options), queryClient);
};
export const getRetryConversionUrl = (id: string) => {
  return `/api/v1/conversions/${id}/retry`;
};

/**
 * @summary Queue a failed Conversion again at the head of its lane, with a fresh set of attempts
 */
export const retryConversion = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ConversionsConversionRow> => {
  return apiFetch<ConversionsConversionRow>(getRetryConversionUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getRetryConversionMutationKey = () => ["retryConversion"] as const;

export const getRetryConversionMutationOptions = <
  TError = ErrorType<
    | RetryConversion401
    | RetryConversion403
    | RetryConversion404
    | RetryConversion409
    | RetryConversion422
    | RetryConversion500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof retryConversion>>,
    TError,
    RetryConversionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof retryConversion>>,
  TError,
  RetryConversionMutationVariables,
  TContext
> => {
  const mutationKey = getRetryConversionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof retryConversion>>,
    RetryConversionMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return retryConversion(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RetryConversionMutationResult = NonNullable<
  Awaited<ReturnType<typeof retryConversion>>
>;

export type RetryConversionMutationError = ErrorType<
  | RetryConversion401
  | RetryConversion403
  | RetryConversion404
  | RetryConversion409
  | RetryConversion422
  | RetryConversion500
>;
export type RetryConversionMutationVariables = { id: string };

/**
 * @summary Queue a failed Conversion again at the head of its lane, with a fresh set of attempts
 */
export const useRetryConversion = <
  TError = ErrorType<
    | RetryConversion401
    | RetryConversion403
    | RetryConversion404
    | RetryConversion409
    | RetryConversion422
    | RetryConversion500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof retryConversion>>,
      TError,
      RetryConversionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof retryConversion>>,
  TError,
  RetryConversionMutationVariables,
  TContext
> => {
  return useMutation(getRetryConversionMutationOptions(options), queryClient);
};
export const getGetNodeConvertingUrl = (id: string) => {
  return `/api/v1/nodes/${id}/converting`;
};

/**
 * @summary Get a Node's Converting section: its settings, what it converts with, and its test runs
 */
export const getNodeConverting = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ConversionsNodeConverting> => {
  return apiFetch<ConversionsNodeConverting>(getGetNodeConvertingUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetNodeConvertingQueryKey = (id: string) => {
  return [`/api/v1/nodes/${id}/converting`] as const;
};

export const getGetNodeConvertingSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getNodeConverting>>,
  TError = ErrorType<
    | GetNodeConverting401
    | GetNodeConverting403
    | GetNodeConverting404
    | GetNodeConverting409
    | GetNodeConverting422
    | GetNodeConverting500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConverting>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetNodeConvertingQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getNodeConverting>>> = ({ signal }) =>
    getNodeConverting(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConverting>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetNodeConvertingSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getNodeConverting>>
>;
export type GetNodeConvertingSuspenseQueryError = ErrorType<
  | GetNodeConverting401
  | GetNodeConverting403
  | GetNodeConverting404
  | GetNodeConverting409
  | GetNodeConverting422
  | GetNodeConverting500
>;

export function useGetNodeConvertingSuspense<
  TData = Awaited<ReturnType<typeof getNodeConverting>>,
  TError = ErrorType<
    | GetNodeConverting401
    | GetNodeConverting403
    | GetNodeConverting404
    | GetNodeConverting409
    | GetNodeConverting422
    | GetNodeConverting500
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConverting>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeConvertingSuspense<
  TData = Awaited<ReturnType<typeof getNodeConverting>>,
  TError = ErrorType<
    | GetNodeConverting401
    | GetNodeConverting403
    | GetNodeConverting404
    | GetNodeConverting409
    | GetNodeConverting422
    | GetNodeConverting500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConverting>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeConvertingSuspense<
  TData = Awaited<ReturnType<typeof getNodeConverting>>,
  TError = ErrorType<
    | GetNodeConverting401
    | GetNodeConverting403
    | GetNodeConverting404
    | GetNodeConverting409
    | GetNodeConverting422
    | GetNodeConverting500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConverting>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Node's Converting section: its settings, what it converts with, and its test runs
 */

export function useGetNodeConvertingSuspense<
  TData = Awaited<ReturnType<typeof getNodeConverting>>,
  TError = ErrorType<
    | GetNodeConverting401
    | GetNodeConverting403
    | GetNodeConverting404
    | GetNodeConverting409
    | GetNodeConverting422
    | GetNodeConverting500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConverting>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetNodeConvertingSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateNodeConvertingUrl = (id: string) => {
  return `/api/v1/nodes/${id}/converting`;
};

/**
 * @summary Change a Node's Conversions at once, hours and processor switch
 */
export const updateNodeConverting = async (
  id: string,
  conversionsNodeSettings: ConversionsNodeSettings,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ConversionsNodeConverting> => {
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
  return apiFetch<ConversionsNodeConverting>(getUpdateNodeConvertingUrl(id), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(conversionsNodeSettings),
  });
};

export const getUpdateNodeConvertingMutationKey = () => ["updateNodeConverting"] as const;

export const getUpdateNodeConvertingMutationOptions = <
  TError = ErrorType<
    | UpdateNodeConverting401
    | UpdateNodeConverting403
    | UpdateNodeConverting404
    | UpdateNodeConverting409
    | UpdateNodeConverting422
    | UpdateNodeConverting500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateNodeConverting>>,
    TError,
    UpdateNodeConvertingMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateNodeConverting>>,
  TError,
  UpdateNodeConvertingMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateNodeConvertingMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateNodeConverting>>,
    UpdateNodeConvertingMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return updateNodeConverting(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateNodeConvertingMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateNodeConverting>>
>;
export type UpdateNodeConvertingMutationBody = ConversionsNodeSettings;
export type UpdateNodeConvertingMutationError = ErrorType<
  | UpdateNodeConverting401
  | UpdateNodeConverting403
  | UpdateNodeConverting404
  | UpdateNodeConverting409
  | UpdateNodeConverting422
  | UpdateNodeConverting500
>;
export type UpdateNodeConvertingMutationVariables = { id: string; data: ConversionsNodeSettings };

/**
 * @summary Change a Node's Conversions at once, hours and processor switch
 */
export const useUpdateNodeConverting = <
  TError = ErrorType<
    | UpdateNodeConverting401
    | UpdateNodeConverting403
    | UpdateNodeConverting404
    | UpdateNodeConverting409
    | UpdateNodeConverting422
    | UpdateNodeConverting500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateNodeConverting>>,
      TError,
      UpdateNodeConvertingMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateNodeConverting>>,
  TError,
  UpdateNodeConvertingMutationVariables,
  TContext
> => {
  return useMutation(getUpdateNodeConvertingMutationOptions(options), queryClient);
};
export const getRerunNodeTestsUrl = (id: string) => {
  return `/api/v1/nodes/${id}/converting/rerun`;
};

/**
 * @summary Run a Node's test runs again
 */
export const rerunNodeTests = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getRerunNodeTestsUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getRerunNodeTestsMutationKey = () => ["rerunNodeTests"] as const;

export const getRerunNodeTestsMutationOptions = <
  TError = ErrorType<
    | RerunNodeTests401
    | RerunNodeTests403
    | RerunNodeTests404
    | RerunNodeTests409
    | RerunNodeTests422
    | RerunNodeTests500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof rerunNodeTests>>,
    TError,
    RerunNodeTestsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof rerunNodeTests>>,
  TError,
  RerunNodeTestsMutationVariables,
  TContext
> => {
  const mutationKey = getRerunNodeTestsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof rerunNodeTests>>,
    RerunNodeTestsMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return rerunNodeTests(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RerunNodeTestsMutationResult = NonNullable<Awaited<ReturnType<typeof rerunNodeTests>>>;

export type RerunNodeTestsMutationError = ErrorType<
  | RerunNodeTests401
  | RerunNodeTests403
  | RerunNodeTests404
  | RerunNodeTests409
  | RerunNodeTests422
  | RerunNodeTests500
>;
export type RerunNodeTestsMutationVariables = { id: string };

/**
 * @summary Run a Node's test runs again
 */
export const useRerunNodeTests = <
  TError = ErrorType<
    | RerunNodeTests401
    | RerunNodeTests403
    | RerunNodeTests404
    | RerunNodeTests409
    | RerunNodeTests422
    | RerunNodeTests500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof rerunNodeTests>>,
      TError,
      RerunNodeTestsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof rerunNodeTests>>,
  TError,
  RerunNodeTestsMutationVariables,
  TContext
> => {
  return useMutation(getRerunNodeTestsMutationOptions(options), queryClient);
};
