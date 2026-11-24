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
  CancelMoveUnit401,
  CancelMoveUnit403,
  CancelMoveUnit404,
  CancelMoveUnit409,
  CancelMoveUnit422,
  CancelMoveUnit500,
  CancelMoves401,
  CancelMoves403,
  CancelMoves404,
  CancelMoves409,
  CancelMoves422,
  CancelMoves500,
  GetMove401,
  GetMove403,
  GetMove404,
  GetMove409,
  GetMove422,
  GetMove500,
  GetMoveParams,
  GetMoveSettings401,
  GetMoveSettings403,
  GetMoveSettings409,
  GetMoveSettings500,
  ListMoves401,
  ListMoves403,
  ListMoves404,
  ListMoves409,
  ListMoves422,
  ListMoves500,
  ListMovesParams,
  MovesBatchList,
  MovesBatchPage,
  MovesCanceledMoves,
  MovesMoveAsk,
  MovesMovePreview,
  MovesMoveSettings,
  MovesMoveSettingsValues,
  MovesMoveUnit,
  MovesStartAsk,
  MovesStartedMoves,
  PatchMoveSettings401,
  PatchMoveSettings403,
  PatchMoveSettings409,
  PatchMoveSettings422,
  PatchMoveSettings500,
  PreviewMoveBack401,
  PreviewMoveBack403,
  PreviewMoveBack404,
  PreviewMoveBack409,
  PreviewMoveBack422,
  PreviewMoveBack500,
  PreviewMoves401,
  PreviewMoves403,
  PreviewMoves404,
  PreviewMoves409,
  PreviewMoves422,
  PreviewMoves500,
  RetryMoveUnit401,
  RetryMoveUnit403,
  RetryMoveUnit404,
  RetryMoveUnit409,
  RetryMoveUnit422,
  RetryMoveUnit500,
  StartMoves401,
  StartMoves403,
  StartMoves409,
  StartMoves410,
  StartMoves422,
  StartMoves500,
  StartMovesParams,
  SwitchMoveUnitNow401,
  SwitchMoveUnitNow403,
  SwitchMoveUnitNow404,
  SwitchMoveUnitNow409,
  SwitchMoveUnitNow422,
  SwitchMoveUnitNow500,
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

export const getListMovesUrl = (params?: ListMovesParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/moves?${stringifiedParams}` : `/api/v1/moves`;
};

/**
 * @summary List the Moves' batches, open ones first
 */
export const listMoves = async (
  params?: ListMovesParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesBatchList> => {
  return apiFetch<MovesBatchList>(getListMovesUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListMovesQueryKey = (params?: ListMovesParams) => {
  return [`/api/v1/moves`, ...(params ? [params] : [])] as const;
};

export const getListMovesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMoves>>,
  TError = ErrorType<
    ListMoves401 | ListMoves403 | ListMoves404 | ListMoves409 | ListMoves422 | ListMoves500
  >,
>(
  params?: ListMovesParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMoves>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMovesQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMoves>>> = ({ signal }) =>
    listMoves(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMoves>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMovesSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listMoves>>>;
export type ListMovesSuspenseQueryError = ErrorType<
  ListMoves401 | ListMoves403 | ListMoves404 | ListMoves409 | ListMoves422 | ListMoves500
>;

export function useListMovesSuspense<
  TData = Awaited<ReturnType<typeof listMoves>>,
  TError = ErrorType<
    ListMoves401 | ListMoves403 | ListMoves404 | ListMoves409 | ListMoves422 | ListMoves500
  >,
>(
  params: undefined | ListMovesParams,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMoves>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMovesSuspense<
  TData = Awaited<ReturnType<typeof listMoves>>,
  TError = ErrorType<
    ListMoves401 | ListMoves403 | ListMoves404 | ListMoves409 | ListMoves422 | ListMoves500
  >,
>(
  params?: ListMovesParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMoves>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMovesSuspense<
  TData = Awaited<ReturnType<typeof listMoves>>,
  TError = ErrorType<
    ListMoves401 | ListMoves403 | ListMoves404 | ListMoves409 | ListMoves422 | ListMoves500
  >,
>(
  params?: ListMovesParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMoves>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Moves' batches, open ones first
 */

export function useListMovesSuspense<
  TData = Awaited<ReturnType<typeof listMoves>>,
  TError = ErrorType<
    ListMoves401 | ListMoves403 | ListMoves404 | ListMoves409 | ListMoves422 | ListMoves500
  >,
>(
  params?: ListMovesParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMoves>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMovesSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getStartMovesUrl = (params: StartMovesParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/v1/moves?${stringifiedParams}` : `/api/v1/moves`;
};

/**
 * @summary Start a previewed Move. A Start sent again returns the same batch
 */
export const startMoves = async (
  movesStartAsk: MovesStartAsk,
  params: StartMovesParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesStartedMoves> => {
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
  return apiFetch<MovesStartedMoves>(getStartMovesUrl(params), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(movesStartAsk),
  });
};

export const getStartMovesMutationKey = () => ["startMoves"] as const;

export const getStartMovesMutationOptions = <
  TError = ErrorType<
    StartMoves401 | StartMoves403 | StartMoves409 | StartMoves410 | StartMoves422 | StartMoves500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof startMoves>>,
    TError,
    StartMovesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof startMoves>>,
  TError,
  StartMovesMutationVariables,
  TContext
> => {
  const mutationKey = getStartMovesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof startMoves>>,
    StartMovesMutationVariables
  > = (props) => {
    const { data, params } = props ?? {};

    return startMoves(data, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type StartMovesMutationResult = NonNullable<Awaited<ReturnType<typeof startMoves>>>;
export type StartMovesMutationBody = MovesStartAsk;
export type StartMovesMutationError = ErrorType<
  StartMoves401 | StartMoves403 | StartMoves409 | StartMoves410 | StartMoves422 | StartMoves500
>;
export type StartMovesMutationVariables = { data: MovesStartAsk; params: StartMovesParams };

/**
 * @summary Start a previewed Move. A Start sent again returns the same batch
 */
export const useStartMoves = <
  TError = ErrorType<
    StartMoves401 | StartMoves403 | StartMoves409 | StartMoves410 | StartMoves422 | StartMoves500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof startMoves>>,
      TError,
      StartMovesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof startMoves>>,
  TError,
  StartMovesMutationVariables,
  TContext
> => {
  return useMutation(getStartMovesMutationOptions(options), queryClient);
};
export const getPreviewMovesUrl = () => {
  return `/api/v1/moves/preview`;
};

/**
 * @summary Preview moving Works to other library roots, or torrents to other Download Clients: a verdict for each, and a token to start it with
 */
export const previewMoves = async (
  movesMoveAsk: MovesMoveAsk,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesMovePreview> => {
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
  return apiFetch<MovesMovePreview>(getPreviewMovesUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(movesMoveAsk),
  });
};

export const getPreviewMovesMutationKey = () => ["previewMoves"] as const;

export const getPreviewMovesMutationOptions = <
  TError = ErrorType<
    | PreviewMoves401
    | PreviewMoves403
    | PreviewMoves404
    | PreviewMoves409
    | PreviewMoves422
    | PreviewMoves500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof previewMoves>>,
    TError,
    PreviewMovesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof previewMoves>>,
  TError,
  PreviewMovesMutationVariables,
  TContext
> => {
  const mutationKey = getPreviewMovesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof previewMoves>>,
    PreviewMovesMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return previewMoves(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PreviewMovesMutationResult = NonNullable<Awaited<ReturnType<typeof previewMoves>>>;
export type PreviewMovesMutationBody = MovesMoveAsk;
export type PreviewMovesMutationError = ErrorType<
  | PreviewMoves401
  | PreviewMoves403
  | PreviewMoves404
  | PreviewMoves409
  | PreviewMoves422
  | PreviewMoves500
>;
export type PreviewMovesMutationVariables = { data: MovesMoveAsk };

/**
 * @summary Preview moving Works to other library roots, or torrents to other Download Clients: a verdict for each, and a token to start it with
 */
export const usePreviewMoves = <
  TError = ErrorType<
    | PreviewMoves401
    | PreviewMoves403
    | PreviewMoves404
    | PreviewMoves409
    | PreviewMoves422
    | PreviewMoves500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof previewMoves>>,
      TError,
      PreviewMovesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof previewMoves>>,
  TError,
  PreviewMovesMutationVariables,
  TContext
> => {
  return useMutation(getPreviewMovesMutationOptions(options), queryClient);
};
export const getGetMoveSettingsUrl = () => {
  return `/api/v1/moves/settings`;
};

/**
 * @summary Get the Moves settings of Settings > Library roots
 */
export const getMoveSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesMoveSettings> => {
  return apiFetch<MovesMoveSettings>(getGetMoveSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetMoveSettingsQueryKey = () => {
  return [`/api/v1/moves/settings`] as const;
};

export const getGetMoveSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getMoveSettings>>,
  TError = ErrorType<
    GetMoveSettings401 | GetMoveSettings403 | GetMoveSettings409 | GetMoveSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMoveSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetMoveSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getMoveSettings>>> = ({ signal }) =>
    getMoveSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMoveSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetMoveSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getMoveSettings>>
>;
export type GetMoveSettingsSuspenseQueryError = ErrorType<
  GetMoveSettings401 | GetMoveSettings403 | GetMoveSettings409 | GetMoveSettings500
>;

export function useGetMoveSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMoveSettings>>,
  TError = ErrorType<
    GetMoveSettings401 | GetMoveSettings403 | GetMoveSettings409 | GetMoveSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMoveSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMoveSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMoveSettings>>,
  TError = ErrorType<
    GetMoveSettings401 | GetMoveSettings403 | GetMoveSettings409 | GetMoveSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMoveSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMoveSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMoveSettings>>,
  TError = ErrorType<
    GetMoveSettings401 | GetMoveSettings403 | GetMoveSettings409 | GetMoveSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMoveSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Moves settings of Settings > Library roots
 */

export function useGetMoveSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMoveSettings>>,
  TError = ErrorType<
    GetMoveSettings401 | GetMoveSettings403 | GetMoveSettings409 | GetMoveSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMoveSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetMoveSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getPatchMoveSettingsUrl = () => {
  return `/api/v1/moves/settings`;
};

/**
 * @summary Change the Moves settings by key, all or none
 */
export const patchMoveSettings = async (
  movesMoveSettingsValues: MovesMoveSettingsValues,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesMoveSettings> => {
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
  return apiFetch<MovesMoveSettings>(getPatchMoveSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(movesMoveSettingsValues),
  });
};

export const getPatchMoveSettingsMutationKey = () => ["patchMoveSettings"] as const;

export const getPatchMoveSettingsMutationOptions = <
  TError = ErrorType<
    | PatchMoveSettings401
    | PatchMoveSettings403
    | PatchMoveSettings409
    | PatchMoveSettings422
    | PatchMoveSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof patchMoveSettings>>,
    TError,
    PatchMoveSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof patchMoveSettings>>,
  TError,
  PatchMoveSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getPatchMoveSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof patchMoveSettings>>,
    PatchMoveSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return patchMoveSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PatchMoveSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof patchMoveSettings>>
>;
export type PatchMoveSettingsMutationBody = MovesMoveSettingsValues;
export type PatchMoveSettingsMutationError = ErrorType<
  | PatchMoveSettings401
  | PatchMoveSettings403
  | PatchMoveSettings409
  | PatchMoveSettings422
  | PatchMoveSettings500
>;
export type PatchMoveSettingsMutationVariables = { data: MovesMoveSettingsValues };

/**
 * @summary Change the Moves settings by key, all or none
 */
export const usePatchMoveSettings = <
  TError = ErrorType<
    | PatchMoveSettings401
    | PatchMoveSettings403
    | PatchMoveSettings409
    | PatchMoveSettings422
    | PatchMoveSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof patchMoveSettings>>,
      TError,
      PatchMoveSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof patchMoveSettings>>,
  TError,
  PatchMoveSettingsMutationVariables,
  TContext
> => {
  return useMutation(getPatchMoveSettingsMutationOptions(options), queryClient);
};
export const getCancelMoveUnitUrl = (unitId: string) => {
  return `/api/v1/moves/units/${unitId}/cancel`;
};

/**
 * @summary Cancel a unit of a Move before its switch. It removes what it placed
 */
export const cancelMoveUnit = async (
  unitId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesMoveUnit> => {
  return apiFetch<MovesMoveUnit>(getCancelMoveUnitUrl(unitId), {
    ...options,
    method: "POST",
  });
};

export const getCancelMoveUnitMutationKey = () => ["cancelMoveUnit"] as const;

export const getCancelMoveUnitMutationOptions = <
  TError = ErrorType<
    | CancelMoveUnit401
    | CancelMoveUnit403
    | CancelMoveUnit404
    | CancelMoveUnit409
    | CancelMoveUnit422
    | CancelMoveUnit500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof cancelMoveUnit>>,
    TError,
    CancelMoveUnitMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof cancelMoveUnit>>,
  TError,
  CancelMoveUnitMutationVariables,
  TContext
> => {
  const mutationKey = getCancelMoveUnitMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof cancelMoveUnit>>,
    CancelMoveUnitMutationVariables
  > = (props) => {
    const { unitId } = props ?? {};

    return cancelMoveUnit(unitId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CancelMoveUnitMutationResult = NonNullable<Awaited<ReturnType<typeof cancelMoveUnit>>>;

export type CancelMoveUnitMutationError = ErrorType<
  | CancelMoveUnit401
  | CancelMoveUnit403
  | CancelMoveUnit404
  | CancelMoveUnit409
  | CancelMoveUnit422
  | CancelMoveUnit500
>;
export type CancelMoveUnitMutationVariables = { unitId: string };

/**
 * @summary Cancel a unit of a Move before its switch. It removes what it placed
 */
export const useCancelMoveUnit = <
  TError = ErrorType<
    | CancelMoveUnit401
    | CancelMoveUnit403
    | CancelMoveUnit404
    | CancelMoveUnit409
    | CancelMoveUnit422
    | CancelMoveUnit500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof cancelMoveUnit>>,
      TError,
      CancelMoveUnitMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof cancelMoveUnit>>,
  TError,
  CancelMoveUnitMutationVariables,
  TContext
> => {
  return useMutation(getCancelMoveUnitMutationOptions(options), queryClient);
};
export const getRetryMoveUnitUrl = (unitId: string) => {
  return `/api/v1/moves/units/${unitId}/retry`;
};

/**
 * @summary Queue a failed unit of a Move again
 */
export const retryMoveUnit = async (
  unitId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesMoveUnit> => {
  return apiFetch<MovesMoveUnit>(getRetryMoveUnitUrl(unitId), {
    ...options,
    method: "POST",
  });
};

export const getRetryMoveUnitMutationKey = () => ["retryMoveUnit"] as const;

export const getRetryMoveUnitMutationOptions = <
  TError = ErrorType<
    | RetryMoveUnit401
    | RetryMoveUnit403
    | RetryMoveUnit404
    | RetryMoveUnit409
    | RetryMoveUnit422
    | RetryMoveUnit500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof retryMoveUnit>>,
    TError,
    RetryMoveUnitMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof retryMoveUnit>>,
  TError,
  RetryMoveUnitMutationVariables,
  TContext
> => {
  const mutationKey = getRetryMoveUnitMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof retryMoveUnit>>,
    RetryMoveUnitMutationVariables
  > = (props) => {
    const { unitId } = props ?? {};

    return retryMoveUnit(unitId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RetryMoveUnitMutationResult = NonNullable<Awaited<ReturnType<typeof retryMoveUnit>>>;

export type RetryMoveUnitMutationError = ErrorType<
  | RetryMoveUnit401
  | RetryMoveUnit403
  | RetryMoveUnit404
  | RetryMoveUnit409
  | RetryMoveUnit422
  | RetryMoveUnit500
>;
export type RetryMoveUnitMutationVariables = { unitId: string };

/**
 * @summary Queue a failed unit of a Move again
 */
export const useRetryMoveUnit = <
  TError = ErrorType<
    | RetryMoveUnit401
    | RetryMoveUnit403
    | RetryMoveUnit404
    | RetryMoveUnit409
    | RetryMoveUnit422
    | RetryMoveUnit500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof retryMoveUnit>>,
      TError,
      RetryMoveUnitMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof retryMoveUnit>>,
  TError,
  RetryMoveUnitMutationVariables,
  TContext
> => {
  return useMutation(getRetryMoveUnitMutationOptions(options), queryClient);
};
export const getSwitchMoveUnitNowUrl = (unitId: string) => {
  return `/api/v1/moves/units/${unitId}/switch-now`;
};

/**
 * @summary Switch a unit waiting for playback or a Media Server without waiting longer
 */
export const switchMoveUnitNow = async (
  unitId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesMoveUnit> => {
  return apiFetch<MovesMoveUnit>(getSwitchMoveUnitNowUrl(unitId), {
    ...options,
    method: "POST",
  });
};

export const getSwitchMoveUnitNowMutationKey = () => ["switchMoveUnitNow"] as const;

export const getSwitchMoveUnitNowMutationOptions = <
  TError = ErrorType<
    | SwitchMoveUnitNow401
    | SwitchMoveUnitNow403
    | SwitchMoveUnitNow404
    | SwitchMoveUnitNow409
    | SwitchMoveUnitNow422
    | SwitchMoveUnitNow500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof switchMoveUnitNow>>,
    TError,
    SwitchMoveUnitNowMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof switchMoveUnitNow>>,
  TError,
  SwitchMoveUnitNowMutationVariables,
  TContext
> => {
  const mutationKey = getSwitchMoveUnitNowMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof switchMoveUnitNow>>,
    SwitchMoveUnitNowMutationVariables
  > = (props) => {
    const { unitId } = props ?? {};

    return switchMoveUnitNow(unitId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SwitchMoveUnitNowMutationResult = NonNullable<
  Awaited<ReturnType<typeof switchMoveUnitNow>>
>;

export type SwitchMoveUnitNowMutationError = ErrorType<
  | SwitchMoveUnitNow401
  | SwitchMoveUnitNow403
  | SwitchMoveUnitNow404
  | SwitchMoveUnitNow409
  | SwitchMoveUnitNow422
  | SwitchMoveUnitNow500
>;
export type SwitchMoveUnitNowMutationVariables = { unitId: string };

/**
 * @summary Switch a unit waiting for playback or a Media Server without waiting longer
 */
export const useSwitchMoveUnitNow = <
  TError = ErrorType<
    | SwitchMoveUnitNow401
    | SwitchMoveUnitNow403
    | SwitchMoveUnitNow404
    | SwitchMoveUnitNow409
    | SwitchMoveUnitNow422
    | SwitchMoveUnitNow500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof switchMoveUnitNow>>,
      TError,
      SwitchMoveUnitNowMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof switchMoveUnitNow>>,
  TError,
  SwitchMoveUnitNowMutationVariables,
  TContext
> => {
  return useMutation(getSwitchMoveUnitNowMutationOptions(options), queryClient);
};
export const getGetMoveUrl = (batchId: string, params?: GetMoveParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/moves/${batchId}?${stringifiedParams}`
    : `/api/v1/moves/${batchId}`;
};

/**
 * @summary Get a Move's batch with a page of its units
 */
export const getMove = async (
  batchId: string,
  params?: GetMoveParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesBatchPage> => {
  return apiFetch<MovesBatchPage>(getGetMoveUrl(batchId, params), {
    ...options,
    method: "GET",
  });
};

export const getGetMoveQueryKey = (batchId: string, params?: GetMoveParams) => {
  return [`/api/v1/moves/${batchId}`, ...(params ? [params] : [])] as const;
};

export const getGetMoveSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getMove>>,
  TError = ErrorType<GetMove401 | GetMove403 | GetMove404 | GetMove409 | GetMove422 | GetMove500>,
>(
  batchId: string,
  params?: GetMoveParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMove>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetMoveQueryKey(batchId, params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getMove>>> = ({ signal }) =>
    getMove(batchId, params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMove>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetMoveSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getMove>>>;
export type GetMoveSuspenseQueryError = ErrorType<
  GetMove401 | GetMove403 | GetMove404 | GetMove409 | GetMove422 | GetMove500
>;

export function useGetMoveSuspense<
  TData = Awaited<ReturnType<typeof getMove>>,
  TError = ErrorType<GetMove401 | GetMove403 | GetMove404 | GetMove409 | GetMove422 | GetMove500>,
>(
  batchId: string,
  params: undefined | GetMoveParams,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMove>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMoveSuspense<
  TData = Awaited<ReturnType<typeof getMove>>,
  TError = ErrorType<GetMove401 | GetMove403 | GetMove404 | GetMove409 | GetMove422 | GetMove500>,
>(
  batchId: string,
  params?: GetMoveParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMove>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMoveSuspense<
  TData = Awaited<ReturnType<typeof getMove>>,
  TError = ErrorType<GetMove401 | GetMove403 | GetMove404 | GetMove409 | GetMove422 | GetMove500>,
>(
  batchId: string,
  params?: GetMoveParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMove>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Move's batch with a page of its units
 */

export function useGetMoveSuspense<
  TData = Awaited<ReturnType<typeof getMove>>,
  TError = ErrorType<GetMove401 | GetMove403 | GetMove404 | GetMove409 | GetMove422 | GetMove500>,
>(
  batchId: string,
  params?: GetMoveParams,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMove>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetMoveSuspenseQueryOptions(batchId, params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getPreviewMoveBackUrl = (batchId: string) => {
  return `/api/v1/moves/${batchId}/back`;
};

/**
 * @summary Preview moving a Move's Works back to the roots they left, and its torrents back to the Download Clients they left
 */
export const previewMoveBack = async (
  batchId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesMovePreview> => {
  return apiFetch<MovesMovePreview>(getPreviewMoveBackUrl(batchId), {
    ...options,
    method: "GET",
  });
};

export const getPreviewMoveBackQueryKey = (batchId: string) => {
  return [`/api/v1/moves/${batchId}/back`] as const;
};

export const getPreviewMoveBackSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof previewMoveBack>>,
  TError = ErrorType<
    | PreviewMoveBack401
    | PreviewMoveBack403
    | PreviewMoveBack404
    | PreviewMoveBack409
    | PreviewMoveBack422
    | PreviewMoveBack500
  >,
>(
  batchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewMoveBack>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getPreviewMoveBackQueryKey(batchId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof previewMoveBack>>> = ({ signal }) =>
    previewMoveBack(batchId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewMoveBack>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type PreviewMoveBackSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof previewMoveBack>>
>;
export type PreviewMoveBackSuspenseQueryError = ErrorType<
  | PreviewMoveBack401
  | PreviewMoveBack403
  | PreviewMoveBack404
  | PreviewMoveBack409
  | PreviewMoveBack422
  | PreviewMoveBack500
>;

export function usePreviewMoveBackSuspense<
  TData = Awaited<ReturnType<typeof previewMoveBack>>,
  TError = ErrorType<
    | PreviewMoveBack401
    | PreviewMoveBack403
    | PreviewMoveBack404
    | PreviewMoveBack409
    | PreviewMoveBack422
    | PreviewMoveBack500
  >,
>(
  batchId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewMoveBack>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function usePreviewMoveBackSuspense<
  TData = Awaited<ReturnType<typeof previewMoveBack>>,
  TError = ErrorType<
    | PreviewMoveBack401
    | PreviewMoveBack403
    | PreviewMoveBack404
    | PreviewMoveBack409
    | PreviewMoveBack422
    | PreviewMoveBack500
  >,
>(
  batchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewMoveBack>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function usePreviewMoveBackSuspense<
  TData = Awaited<ReturnType<typeof previewMoveBack>>,
  TError = ErrorType<
    | PreviewMoveBack401
    | PreviewMoveBack403
    | PreviewMoveBack404
    | PreviewMoveBack409
    | PreviewMoveBack422
    | PreviewMoveBack500
  >,
>(
  batchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewMoveBack>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Preview moving a Move's Works back to the roots they left, and its torrents back to the Download Clients they left
 */

export function usePreviewMoveBackSuspense<
  TData = Awaited<ReturnType<typeof previewMoveBack>>,
  TError = ErrorType<
    | PreviewMoveBack401
    | PreviewMoveBack403
    | PreviewMoveBack404
    | PreviewMoveBack409
    | PreviewMoveBack422
    | PreviewMoveBack500
  >,
>(
  batchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof previewMoveBack>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getPreviewMoveBackSuspenseQueryOptions(batchId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCancelMovesUrl = (batchId: string) => {
  return `/api/v1/moves/${batchId}/cancel`;
};

/**
 * @summary Cancel every unit of a Move that hasn't switched. Each one removes what it placed
 */
export const cancelMoves = async (
  batchId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MovesCanceledMoves> => {
  return apiFetch<MovesCanceledMoves>(getCancelMovesUrl(batchId), {
    ...options,
    method: "POST",
  });
};

export const getCancelMovesMutationKey = () => ["cancelMoves"] as const;

export const getCancelMovesMutationOptions = <
  TError = ErrorType<
    | CancelMoves401
    | CancelMoves403
    | CancelMoves404
    | CancelMoves409
    | CancelMoves422
    | CancelMoves500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof cancelMoves>>,
    TError,
    CancelMovesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof cancelMoves>>,
  TError,
  CancelMovesMutationVariables,
  TContext
> => {
  const mutationKey = getCancelMovesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof cancelMoves>>,
    CancelMovesMutationVariables
  > = (props) => {
    const { batchId } = props ?? {};

    return cancelMoves(batchId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CancelMovesMutationResult = NonNullable<Awaited<ReturnType<typeof cancelMoves>>>;

export type CancelMovesMutationError = ErrorType<
  | CancelMoves401
  | CancelMoves403
  | CancelMoves404
  | CancelMoves409
  | CancelMoves422
  | CancelMoves500
>;
export type CancelMovesMutationVariables = { batchId: string };

/**
 * @summary Cancel every unit of a Move that hasn't switched. Each one removes what it placed
 */
export const useCancelMoves = <
  TError = ErrorType<
    | CancelMoves401
    | CancelMoves403
    | CancelMoves404
    | CancelMoves409
    | CancelMoves422
    | CancelMoves500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof cancelMoves>>,
      TError,
      CancelMovesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof cancelMoves>>,
  TError,
  CancelMovesMutationVariables,
  TContext
> => {
  return useMutation(getCancelMovesMutationOptions(options), queryClient);
};
