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
  AddExposure401,
  AddExposure403,
  AddExposure404,
  AddExposure409,
  AddExposure422,
  AddExposure500,
  DismissSuggestion401,
  DismissSuggestion403,
  DismissSuggestion404,
  DismissSuggestion409,
  DismissSuggestion422,
  DismissSuggestion500,
  EditExposure401,
  EditExposure403,
  EditExposure404,
  EditExposure409,
  EditExposure422,
  EditExposure500,
  GetNodeNetwork401,
  GetNodeNetwork403,
  GetNodeNetwork404,
  GetNodeNetwork409,
  GetNodeNetwork422,
  GetNodeNetwork500,
  GetNodeNetworkSettings401,
  GetNodeNetworkSettings403,
  GetNodeNetworkSettings404,
  GetNodeNetworkSettings409,
  GetNodeNetworkSettings422,
  GetNodeNetworkSettings500,
  GetRemoteAccess401,
  GetRemoteAccess403,
  GetRemoteAccess409,
  GetRemoteAccess500,
  GetRemoteAccessSettings401,
  GetRemoteAccessSettings403,
  GetRemoteAccessSettings409,
  GetRemoteAccessSettings500,
  PatchNodeNetworkSettings401,
  PatchNodeNetworkSettings403,
  PatchNodeNetworkSettings404,
  PatchNodeNetworkSettings409,
  PatchNodeNetworkSettings422,
  PatchNodeNetworkSettings500,
  PatchRemoteAccessSettings401,
  PatchRemoteAccessSettings403,
  PatchRemoteAccessSettings409,
  PatchRemoteAccessSettings422,
  PatchRemoteAccessSettings500,
  ReachabilityExposurePublic,
  ReachabilityExposureView,
  ReachabilityNewExposure,
  ReachabilityNodeNetwork,
  ReachabilityRemoteAccess,
  ReachabilityRunStarted,
  ReachabilitySettingsBody,
  ReachabilitySettingsValues,
  RemoveExposure401,
  RemoveExposure403,
  RemoveExposure404,
  RemoveExposure409,
  RemoveExposure422,
  RemoveExposure500,
  RunRemoteAccess401,
  RunRemoteAccess403,
  RunRemoteAccess404,
  RunRemoteAccess409,
  RunRemoteAccess422,
  RunRemoteAccess500,
  RunRemoteAccessParams,
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

export const getGetRemoteAccessUrl = () => {
  return `/api/v1/remote-access`;
};

/**
 * @summary Get Remote access: every route set up for use away from home with its verdict, the suggestions and each Node's class
 */
export const getRemoteAccess = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReachabilityRemoteAccess> => {
  return apiFetch<ReachabilityRemoteAccess>(getGetRemoteAccessUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetRemoteAccessQueryKey = () => {
  return [`/api/v1/remote-access`] as const;
};

export const getGetRemoteAccessSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getRemoteAccess>>,
  TError = ErrorType<
    GetRemoteAccess401 | GetRemoteAccess403 | GetRemoteAccess409 | GetRemoteAccess500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccess>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetRemoteAccessQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getRemoteAccess>>> = ({ signal }) =>
    getRemoteAccess({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccess>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetRemoteAccessSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getRemoteAccess>>
>;
export type GetRemoteAccessSuspenseQueryError = ErrorType<
  GetRemoteAccess401 | GetRemoteAccess403 | GetRemoteAccess409 | GetRemoteAccess500
>;

export function useGetRemoteAccessSuspense<
  TData = Awaited<ReturnType<typeof getRemoteAccess>>,
  TError = ErrorType<
    GetRemoteAccess401 | GetRemoteAccess403 | GetRemoteAccess409 | GetRemoteAccess500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetRemoteAccessSuspense<
  TData = Awaited<ReturnType<typeof getRemoteAccess>>,
  TError = ErrorType<
    GetRemoteAccess401 | GetRemoteAccess403 | GetRemoteAccess409 | GetRemoteAccess500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetRemoteAccessSuspense<
  TData = Awaited<ReturnType<typeof getRemoteAccess>>,
  TError = ErrorType<
    GetRemoteAccess401 | GetRemoteAccess403 | GetRemoteAccess409 | GetRemoteAccess500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get Remote access: every route set up for use away from home with its verdict, the suggestions and each Node's class
 */

export function useGetRemoteAccessSuspense<
  TData = Awaited<ReturnType<typeof getRemoteAccess>>,
  TError = ErrorType<
    GetRemoteAccess401 | GetRemoteAccess403 | GetRemoteAccess409 | GetRemoteAccess500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetRemoteAccessSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getAddExposureUrl = () => {
  return `/api/v1/remote-access/exposures`;
};

/**
 * @summary Set the UI and portal, or a Media Server, up for use away from home, and check it
 */
export const addExposure = async (
  reachabilityNewExposure: ReachabilityNewExposure,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReachabilityExposureView> => {
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
  return apiFetch<ReachabilityExposureView>(getAddExposureUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(reachabilityNewExposure),
  });
};

export const getAddExposureMutationKey = () => ["addExposure"] as const;

export const getAddExposureMutationOptions = <
  TError = ErrorType<
    | AddExposure401
    | AddExposure403
    | AddExposure404
    | AddExposure409
    | AddExposure422
    | AddExposure500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof addExposure>>,
    TError,
    AddExposureMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof addExposure>>,
  TError,
  AddExposureMutationVariables,
  TContext
> => {
  const mutationKey = getAddExposureMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof addExposure>>,
    AddExposureMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return addExposure(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type AddExposureMutationResult = NonNullable<Awaited<ReturnType<typeof addExposure>>>;
export type AddExposureMutationBody = ReachabilityNewExposure;
export type AddExposureMutationError = ErrorType<
  | AddExposure401
  | AddExposure403
  | AddExposure404
  | AddExposure409
  | AddExposure422
  | AddExposure500
>;
export type AddExposureMutationVariables = { data: ReachabilityNewExposure };

/**
 * @summary Set the UI and portal, or a Media Server, up for use away from home, and check it
 */
export const useAddExposure = <
  TError = ErrorType<
    | AddExposure401
    | AddExposure403
    | AddExposure404
    | AddExposure409
    | AddExposure422
    | AddExposure500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof addExposure>>,
      TError,
      AddExposureMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof addExposure>>,
  TError,
  AddExposureMutationVariables,
  TContext
> => {
  return useMutation(getAddExposureMutationOptions(options), queryClient);
};
export const getRemoveExposureUrl = (id: string) => {
  return `/api/v1/remote-access/exposures/${id}`;
};

/**
 * @summary Stop checking a route, and forget its result
 */
export const removeExposure = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getRemoveExposureUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getRemoveExposureMutationKey = () => ["removeExposure"] as const;

export const getRemoveExposureMutationOptions = <
  TError = ErrorType<
    | RemoveExposure401
    | RemoveExposure403
    | RemoveExposure404
    | RemoveExposure409
    | RemoveExposure422
    | RemoveExposure500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof removeExposure>>,
    TError,
    RemoveExposureMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof removeExposure>>,
  TError,
  RemoveExposureMutationVariables,
  TContext
> => {
  const mutationKey = getRemoveExposureMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof removeExposure>>,
    RemoveExposureMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return removeExposure(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RemoveExposureMutationResult = NonNullable<Awaited<ReturnType<typeof removeExposure>>>;

export type RemoveExposureMutationError = ErrorType<
  | RemoveExposure401
  | RemoveExposure403
  | RemoveExposure404
  | RemoveExposure409
  | RemoveExposure422
  | RemoveExposure500
>;
export type RemoveExposureMutationVariables = { id: string };

/**
 * @summary Stop checking a route, and forget its result
 */
export const useRemoveExposure = <
  TError = ErrorType<
    | RemoveExposure401
    | RemoveExposure403
    | RemoveExposure404
    | RemoveExposure409
    | RemoveExposure422
    | RemoveExposure500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof removeExposure>>,
      TError,
      RemoveExposureMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof removeExposure>>,
  TError,
  RemoveExposureMutationVariables,
  TContext
> => {
  return useMutation(getRemoveExposureMutationOptions(options), queryClient);
};
export const getEditExposureUrl = (id: string) => {
  return `/api/v1/remote-access/exposures/${id}`;
};

/**
 * @summary Change a Media Server's public address, and check it again
 */
export const editExposure = async (
  id: string,
  reachabilityExposurePublic: ReachabilityExposurePublic,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReachabilityExposureView> => {
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
  return apiFetch<ReachabilityExposureView>(getEditExposureUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(reachabilityExposurePublic),
  });
};

export const getEditExposureMutationKey = () => ["editExposure"] as const;

export const getEditExposureMutationOptions = <
  TError = ErrorType<
    | EditExposure401
    | EditExposure403
    | EditExposure404
    | EditExposure409
    | EditExposure422
    | EditExposure500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof editExposure>>,
    TError,
    EditExposureMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof editExposure>>,
  TError,
  EditExposureMutationVariables,
  TContext
> => {
  const mutationKey = getEditExposureMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof editExposure>>,
    EditExposureMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return editExposure(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type EditExposureMutationResult = NonNullable<Awaited<ReturnType<typeof editExposure>>>;
export type EditExposureMutationBody = ReachabilityExposurePublic;
export type EditExposureMutationError = ErrorType<
  | EditExposure401
  | EditExposure403
  | EditExposure404
  | EditExposure409
  | EditExposure422
  | EditExposure500
>;
export type EditExposureMutationVariables = { id: string; data: ReachabilityExposurePublic };

/**
 * @summary Change a Media Server's public address, and check it again
 */
export const useEditExposure = <
  TError = ErrorType<
    | EditExposure401
    | EditExposure403
    | EditExposure404
    | EditExposure409
    | EditExposure422
    | EditExposure500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof editExposure>>,
      TError,
      EditExposureMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof editExposure>>,
  TError,
  EditExposureMutationVariables,
  TContext
> => {
  return useMutation(getEditExposureMutationOptions(options), queryClient);
};
export const getGetNodeNetworkUrl = (id: string) => {
  return `/api/v1/remote-access/nodes/${id}`;
};

/**
 * @summary Get a Node's Network section, read now with nothing sent outside the machine
 */
export const getNodeNetwork = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReachabilityNodeNetwork> => {
  return apiFetch<ReachabilityNodeNetwork>(getGetNodeNetworkUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetNodeNetworkQueryKey = (id: string) => {
  return [`/api/v1/remote-access/nodes/${id}`] as const;
};

export const getGetNodeNetworkSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getNodeNetwork>>,
  TError = ErrorType<
    | GetNodeNetwork401
    | GetNodeNetwork403
    | GetNodeNetwork404
    | GetNodeNetwork409
    | GetNodeNetwork422
    | GetNodeNetwork500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetwork>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetNodeNetworkQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getNodeNetwork>>> = ({ signal }) =>
    getNodeNetwork(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetwork>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetNodeNetworkSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getNodeNetwork>>
>;
export type GetNodeNetworkSuspenseQueryError = ErrorType<
  | GetNodeNetwork401
  | GetNodeNetwork403
  | GetNodeNetwork404
  | GetNodeNetwork409
  | GetNodeNetwork422
  | GetNodeNetwork500
>;

export function useGetNodeNetworkSuspense<
  TData = Awaited<ReturnType<typeof getNodeNetwork>>,
  TError = ErrorType<
    | GetNodeNetwork401
    | GetNodeNetwork403
    | GetNodeNetwork404
    | GetNodeNetwork409
    | GetNodeNetwork422
    | GetNodeNetwork500
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetwork>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeNetworkSuspense<
  TData = Awaited<ReturnType<typeof getNodeNetwork>>,
  TError = ErrorType<
    | GetNodeNetwork401
    | GetNodeNetwork403
    | GetNodeNetwork404
    | GetNodeNetwork409
    | GetNodeNetwork422
    | GetNodeNetwork500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetwork>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeNetworkSuspense<
  TData = Awaited<ReturnType<typeof getNodeNetwork>>,
  TError = ErrorType<
    | GetNodeNetwork401
    | GetNodeNetwork403
    | GetNodeNetwork404
    | GetNodeNetwork409
    | GetNodeNetwork422
    | GetNodeNetwork500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetwork>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Node's Network section, read now with nothing sent outside the machine
 */

export function useGetNodeNetworkSuspense<
  TData = Awaited<ReturnType<typeof getNodeNetwork>>,
  TError = ErrorType<
    | GetNodeNetwork401
    | GetNodeNetwork403
    | GetNodeNetwork404
    | GetNodeNetwork409
    | GetNodeNetwork422
    | GetNodeNetwork500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetwork>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetNodeNetworkSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetNodeNetworkSettingsUrl = (id: string) => {
  return `/api/v1/remote-access/nodes/${id}/settings`;
};

/**
 * @summary Get a Node's Remote access settings
 */
export const getNodeNetworkSettings = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReachabilitySettingsBody> => {
  return apiFetch<ReachabilitySettingsBody>(getGetNodeNetworkSettingsUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetNodeNetworkSettingsQueryKey = (id: string) => {
  return [`/api/v1/remote-access/nodes/${id}/settings`] as const;
};

export const getGetNodeNetworkSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getNodeNetworkSettings>>,
  TError = ErrorType<
    | GetNodeNetworkSettings401
    | GetNodeNetworkSettings403
    | GetNodeNetworkSettings404
    | GetNodeNetworkSettings409
    | GetNodeNetworkSettings422
    | GetNodeNetworkSettings500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetworkSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetNodeNetworkSettingsQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getNodeNetworkSettings>>> = ({ signal }) =>
    getNodeNetworkSettings(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof getNodeNetworkSettings>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetNodeNetworkSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getNodeNetworkSettings>>
>;
export type GetNodeNetworkSettingsSuspenseQueryError = ErrorType<
  | GetNodeNetworkSettings401
  | GetNodeNetworkSettings403
  | GetNodeNetworkSettings404
  | GetNodeNetworkSettings409
  | GetNodeNetworkSettings422
  | GetNodeNetworkSettings500
>;

export function useGetNodeNetworkSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNodeNetworkSettings>>,
  TError = ErrorType<
    | GetNodeNetworkSettings401
    | GetNodeNetworkSettings403
    | GetNodeNetworkSettings404
    | GetNodeNetworkSettings409
    | GetNodeNetworkSettings422
    | GetNodeNetworkSettings500
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetworkSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeNetworkSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNodeNetworkSettings>>,
  TError = ErrorType<
    | GetNodeNetworkSettings401
    | GetNodeNetworkSettings403
    | GetNodeNetworkSettings404
    | GetNodeNetworkSettings409
    | GetNodeNetworkSettings422
    | GetNodeNetworkSettings500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetworkSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeNetworkSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNodeNetworkSettings>>,
  TError = ErrorType<
    | GetNodeNetworkSettings401
    | GetNodeNetworkSettings403
    | GetNodeNetworkSettings404
    | GetNodeNetworkSettings409
    | GetNodeNetworkSettings422
    | GetNodeNetworkSettings500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetworkSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Node's Remote access settings
 */

export function useGetNodeNetworkSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNodeNetworkSettings>>,
  TError = ErrorType<
    | GetNodeNetworkSettings401
    | GetNodeNetworkSettings403
    | GetNodeNetworkSettings404
    | GetNodeNetworkSettings409
    | GetNodeNetworkSettings422
    | GetNodeNetworkSettings500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeNetworkSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetNodeNetworkSettingsSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getPatchNodeNetworkSettingsUrl = (id: string) => {
  return `/api/v1/remote-access/nodes/${id}/settings`;
};

/**
 * @summary Change a Node's Remote access settings by key, all or none
 */
export const patchNodeNetworkSettings = async (
  id: string,
  reachabilitySettingsValues: ReachabilitySettingsValues,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReachabilitySettingsBody> => {
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
  return apiFetch<ReachabilitySettingsBody>(getPatchNodeNetworkSettingsUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(reachabilitySettingsValues),
  });
};

export const getPatchNodeNetworkSettingsMutationKey = () => ["patchNodeNetworkSettings"] as const;

export const getPatchNodeNetworkSettingsMutationOptions = <
  TError = ErrorType<
    | PatchNodeNetworkSettings401
    | PatchNodeNetworkSettings403
    | PatchNodeNetworkSettings404
    | PatchNodeNetworkSettings409
    | PatchNodeNetworkSettings422
    | PatchNodeNetworkSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof patchNodeNetworkSettings>>,
    TError,
    PatchNodeNetworkSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof patchNodeNetworkSettings>>,
  TError,
  PatchNodeNetworkSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getPatchNodeNetworkSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof patchNodeNetworkSettings>>,
    PatchNodeNetworkSettingsMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return patchNodeNetworkSettings(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PatchNodeNetworkSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof patchNodeNetworkSettings>>
>;
export type PatchNodeNetworkSettingsMutationBody = ReachabilitySettingsValues;
export type PatchNodeNetworkSettingsMutationError = ErrorType<
  | PatchNodeNetworkSettings401
  | PatchNodeNetworkSettings403
  | PatchNodeNetworkSettings404
  | PatchNodeNetworkSettings409
  | PatchNodeNetworkSettings422
  | PatchNodeNetworkSettings500
>;
export type PatchNodeNetworkSettingsMutationVariables = {
  id: string;
  data: ReachabilitySettingsValues;
};

/**
 * @summary Change a Node's Remote access settings by key, all or none
 */
export const usePatchNodeNetworkSettings = <
  TError = ErrorType<
    | PatchNodeNetworkSettings401
    | PatchNodeNetworkSettings403
    | PatchNodeNetworkSettings404
    | PatchNodeNetworkSettings409
    | PatchNodeNetworkSettings422
    | PatchNodeNetworkSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof patchNodeNetworkSettings>>,
      TError,
      PatchNodeNetworkSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof patchNodeNetworkSettings>>,
  TError,
  PatchNodeNetworkSettingsMutationVariables,
  TContext
> => {
  return useMutation(getPatchNodeNetworkSettingsMutationOptions(options), queryClient);
};
export const getRunRemoteAccessUrl = (params?: RunRemoteAccessParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/remote-access/run?${stringifiedParams}`
    : `/api/v1/remote-access/run`;
};

/**
 * @summary Check every route again, or one, with a fresh look at each Node's network
 */
export const runRemoteAccess = async (
  params?: RunRemoteAccessParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReachabilityRunStarted> => {
  return apiFetch<ReachabilityRunStarted>(getRunRemoteAccessUrl(params), {
    ...options,
    method: "POST",
  });
};

export const getRunRemoteAccessMutationKey = () => ["runRemoteAccess"] as const;

export const getRunRemoteAccessMutationOptions = <
  TError = ErrorType<
    | RunRemoteAccess401
    | RunRemoteAccess403
    | RunRemoteAccess404
    | RunRemoteAccess409
    | RunRemoteAccess422
    | RunRemoteAccess500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof runRemoteAccess>>,
    TError,
    RunRemoteAccessMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof runRemoteAccess>>,
  TError,
  RunRemoteAccessMutationVariables,
  TContext
> => {
  const mutationKey = getRunRemoteAccessMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof runRemoteAccess>>,
    RunRemoteAccessMutationVariables
  > = (props) => {
    const { params } = props ?? {};

    return runRemoteAccess(params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RunRemoteAccessMutationResult = NonNullable<
  Awaited<ReturnType<typeof runRemoteAccess>>
>;

export type RunRemoteAccessMutationError = ErrorType<
  | RunRemoteAccess401
  | RunRemoteAccess403
  | RunRemoteAccess404
  | RunRemoteAccess409
  | RunRemoteAccess422
  | RunRemoteAccess500
>;
export type RunRemoteAccessMutationVariables = { params?: RunRemoteAccessParams };

/**
 * @summary Check every route again, or one, with a fresh look at each Node's network
 */
export const useRunRemoteAccess = <
  TError = ErrorType<
    | RunRemoteAccess401
    | RunRemoteAccess403
    | RunRemoteAccess404
    | RunRemoteAccess409
    | RunRemoteAccess422
    | RunRemoteAccess500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof runRemoteAccess>>,
      TError,
      RunRemoteAccessMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof runRemoteAccess>>,
  TError,
  RunRemoteAccessMutationVariables,
  TContext
> => {
  return useMutation(getRunRemoteAccessMutationOptions(options), queryClient);
};
export const getGetRemoteAccessSettingsUrl = () => {
  return `/api/v1/remote-access/settings`;
};

/**
 * @summary Get the Remote access settings of Settings > Access
 */
export const getRemoteAccessSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReachabilitySettingsBody> => {
  return apiFetch<ReachabilitySettingsBody>(getGetRemoteAccessSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetRemoteAccessSettingsQueryKey = () => {
  return [`/api/v1/remote-access/settings`] as const;
};

export const getGetRemoteAccessSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getRemoteAccessSettings>>,
  TError = ErrorType<
    | GetRemoteAccessSettings401
    | GetRemoteAccessSettings403
    | GetRemoteAccessSettings409
    | GetRemoteAccessSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccessSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetRemoteAccessSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getRemoteAccessSettings>>> = ({
    signal,
  }) => getRemoteAccessSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof getRemoteAccessSettings>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetRemoteAccessSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getRemoteAccessSettings>>
>;
export type GetRemoteAccessSettingsSuspenseQueryError = ErrorType<
  | GetRemoteAccessSettings401
  | GetRemoteAccessSettings403
  | GetRemoteAccessSettings409
  | GetRemoteAccessSettings500
>;

export function useGetRemoteAccessSettingsSuspense<
  TData = Awaited<ReturnType<typeof getRemoteAccessSettings>>,
  TError = ErrorType<
    | GetRemoteAccessSettings401
    | GetRemoteAccessSettings403
    | GetRemoteAccessSettings409
    | GetRemoteAccessSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccessSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetRemoteAccessSettingsSuspense<
  TData = Awaited<ReturnType<typeof getRemoteAccessSettings>>,
  TError = ErrorType<
    | GetRemoteAccessSettings401
    | GetRemoteAccessSettings403
    | GetRemoteAccessSettings409
    | GetRemoteAccessSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccessSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetRemoteAccessSettingsSuspense<
  TData = Awaited<ReturnType<typeof getRemoteAccessSettings>>,
  TError = ErrorType<
    | GetRemoteAccessSettings401
    | GetRemoteAccessSettings403
    | GetRemoteAccessSettings409
    | GetRemoteAccessSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccessSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Remote access settings of Settings > Access
 */

export function useGetRemoteAccessSettingsSuspense<
  TData = Awaited<ReturnType<typeof getRemoteAccessSettings>>,
  TError = ErrorType<
    | GetRemoteAccessSettings401
    | GetRemoteAccessSettings403
    | GetRemoteAccessSettings409
    | GetRemoteAccessSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRemoteAccessSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetRemoteAccessSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getPatchRemoteAccessSettingsUrl = () => {
  return `/api/v1/remote-access/settings`;
};

/**
 * @summary Change the Remote access settings by key, all or none
 */
export const patchRemoteAccessSettings = async (
  reachabilitySettingsValues: ReachabilitySettingsValues,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ReachabilitySettingsBody> => {
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
  return apiFetch<ReachabilitySettingsBody>(getPatchRemoteAccessSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(reachabilitySettingsValues),
  });
};

export const getPatchRemoteAccessSettingsMutationKey = () => ["patchRemoteAccessSettings"] as const;

export const getPatchRemoteAccessSettingsMutationOptions = <
  TError = ErrorType<
    | PatchRemoteAccessSettings401
    | PatchRemoteAccessSettings403
    | PatchRemoteAccessSettings409
    | PatchRemoteAccessSettings422
    | PatchRemoteAccessSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof patchRemoteAccessSettings>>,
    TError,
    PatchRemoteAccessSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof patchRemoteAccessSettings>>,
  TError,
  PatchRemoteAccessSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getPatchRemoteAccessSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof patchRemoteAccessSettings>>,
    PatchRemoteAccessSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return patchRemoteAccessSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PatchRemoteAccessSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof patchRemoteAccessSettings>>
>;
export type PatchRemoteAccessSettingsMutationBody = ReachabilitySettingsValues;
export type PatchRemoteAccessSettingsMutationError = ErrorType<
  | PatchRemoteAccessSettings401
  | PatchRemoteAccessSettings403
  | PatchRemoteAccessSettings409
  | PatchRemoteAccessSettings422
  | PatchRemoteAccessSettings500
>;
export type PatchRemoteAccessSettingsMutationVariables = { data: ReachabilitySettingsValues };

/**
 * @summary Change the Remote access settings by key, all or none
 */
export const usePatchRemoteAccessSettings = <
  TError = ErrorType<
    | PatchRemoteAccessSettings401
    | PatchRemoteAccessSettings403
    | PatchRemoteAccessSettings409
    | PatchRemoteAccessSettings422
    | PatchRemoteAccessSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof patchRemoteAccessSettings>>,
      TError,
      PatchRemoteAccessSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof patchRemoteAccessSettings>>,
  TError,
  PatchRemoteAccessSettingsMutationVariables,
  TContext
> => {
  return useMutation(getPatchRemoteAccessSettingsMutationOptions(options), queryClient);
};
export const getDismissSuggestionUrl = (id: string) => {
  return `/api/v1/remote-access/suggestions/${id}/dismiss`;
};

/**
 * @summary Hide a suggestion: ui-portal, or a Media Server's id
 */
export const dismissSuggestion = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDismissSuggestionUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getDismissSuggestionMutationKey = () => ["dismissSuggestion"] as const;

export const getDismissSuggestionMutationOptions = <
  TError = ErrorType<
    | DismissSuggestion401
    | DismissSuggestion403
    | DismissSuggestion404
    | DismissSuggestion409
    | DismissSuggestion422
    | DismissSuggestion500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof dismissSuggestion>>,
    TError,
    DismissSuggestionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof dismissSuggestion>>,
  TError,
  DismissSuggestionMutationVariables,
  TContext
> => {
  const mutationKey = getDismissSuggestionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof dismissSuggestion>>,
    DismissSuggestionMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return dismissSuggestion(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DismissSuggestionMutationResult = NonNullable<
  Awaited<ReturnType<typeof dismissSuggestion>>
>;

export type DismissSuggestionMutationError = ErrorType<
  | DismissSuggestion401
  | DismissSuggestion403
  | DismissSuggestion404
  | DismissSuggestion409
  | DismissSuggestion422
  | DismissSuggestion500
>;
export type DismissSuggestionMutationVariables = { id: string };

/**
 * @summary Hide a suggestion: ui-portal, or a Media Server's id
 */
export const useDismissSuggestion = <
  TError = ErrorType<
    | DismissSuggestion401
    | DismissSuggestion403
    | DismissSuggestion404
    | DismissSuggestion409
    | DismissSuggestion422
    | DismissSuggestion500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof dismissSuggestion>>,
      TError,
      DismissSuggestionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof dismissSuggestion>>,
  TError,
  DismissSuggestionMutationVariables,
  TContext
> => {
  return useMutation(getDismissSuggestionMutationOptions(options), queryClient);
};
