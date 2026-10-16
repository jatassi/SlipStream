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
  GetHealth500,
  GetNodeConfig401,
  GetNodeConfig403,
  GetNodeConfig404,
  GetNodeConfig409,
  GetNodeConfig422,
  GetNodeConfig500,
  GetNodeConfig503,
  GetNodeSystem401,
  GetNodeSystem403,
  GetNodeSystem404,
  GetNodeSystem409,
  GetNodeSystem422,
  GetNodeSystem500,
  GetNodeSystem503,
  GetStatus500,
  GetSystem401,
  GetSystem403,
  GetSystem409,
  GetSystem500,
  RestartNode401,
  RestartNode403,
  RestartNode404,
  RestartNode409,
  RestartNode422,
  RestartNode500,
  RestartNode503,
  SystemHealth,
  SystemNodeConfig,
  SystemReport,
  SystemRestarting,
  SystemStatus,
  UpdateNodeConfig401,
  UpdateNodeConfig403,
  UpdateNodeConfig404,
  UpdateNodeConfig409,
  UpdateNodeConfig422,
  UpdateNodeConfig500,
  UpdateNodeConfig503,
  UpdateNodeConfigBody,
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

export const getGetHealthUrl = () => {
  return `/api/v1/health`;
};

/**
 * @summary Report that the Node is serving
 */
export const getHealth = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<SystemHealth> => {
  return apiFetch<SystemHealth>(getGetHealthUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetHealthQueryKey = () => {
  return [`/api/v1/health`] as const;
};

export const getGetHealthSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getHealth>>,
  TError = ErrorType<GetHealth500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetHealthQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getHealth>>> = ({ signal }) =>
    getHealth({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetHealthSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getHealth>>>;
export type GetHealthSuspenseQueryError = ErrorType<GetHealth500>;

export function useGetHealthSuspense<
  TData = Awaited<ReturnType<typeof getHealth>>,
  TError = ErrorType<GetHealth500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHealthSuspense<
  TData = Awaited<ReturnType<typeof getHealth>>,
  TError = ErrorType<GetHealth500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetHealthSuspense<
  TData = Awaited<ReturnType<typeof getHealth>>,
  TError = ErrorType<GetHealth500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Report that the Node is serving
 */

export function useGetHealthSuspense<
  TData = Awaited<ReturnType<typeof getHealth>>,
  TError = ErrorType<GetHealth500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getHealth>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetHealthSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetNodeConfigUrl = (id: string) => {
  return `/api/v1/nodes/${id}/config`;
};

/**
 * @summary Get a Node's settings, by section
 */
export const getNodeConfig = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SystemNodeConfig> => {
  return apiFetch<SystemNodeConfig>(getGetNodeConfigUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetNodeConfigQueryKey = (id: string) => {
  return [`/api/v1/nodes/${id}/config`] as const;
};

export const getGetNodeConfigSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getNodeConfig>>,
  TError = ErrorType<
    | GetNodeConfig401
    | GetNodeConfig403
    | GetNodeConfig404
    | GetNodeConfig409
    | GetNodeConfig422
    | GetNodeConfig500
    | GetNodeConfig503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConfig>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetNodeConfigQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getNodeConfig>>> = ({ signal }) =>
    getNodeConfig(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConfig>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetNodeConfigSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getNodeConfig>>
>;
export type GetNodeConfigSuspenseQueryError = ErrorType<
  | GetNodeConfig401
  | GetNodeConfig403
  | GetNodeConfig404
  | GetNodeConfig409
  | GetNodeConfig422
  | GetNodeConfig500
  | GetNodeConfig503
>;

export function useGetNodeConfigSuspense<
  TData = Awaited<ReturnType<typeof getNodeConfig>>,
  TError = ErrorType<
    | GetNodeConfig401
    | GetNodeConfig403
    | GetNodeConfig404
    | GetNodeConfig409
    | GetNodeConfig422
    | GetNodeConfig500
    | GetNodeConfig503
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConfig>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeConfigSuspense<
  TData = Awaited<ReturnType<typeof getNodeConfig>>,
  TError = ErrorType<
    | GetNodeConfig401
    | GetNodeConfig403
    | GetNodeConfig404
    | GetNodeConfig409
    | GetNodeConfig422
    | GetNodeConfig500
    | GetNodeConfig503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConfig>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeConfigSuspense<
  TData = Awaited<ReturnType<typeof getNodeConfig>>,
  TError = ErrorType<
    | GetNodeConfig401
    | GetNodeConfig403
    | GetNodeConfig404
    | GetNodeConfig409
    | GetNodeConfig422
    | GetNodeConfig500
    | GetNodeConfig503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConfig>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Node's settings, by section
 */

export function useGetNodeConfigSuspense<
  TData = Awaited<ReturnType<typeof getNodeConfig>>,
  TError = ErrorType<
    | GetNodeConfig401
    | GetNodeConfig403
    | GetNodeConfig404
    | GetNodeConfig409
    | GetNodeConfig422
    | GetNodeConfig500
    | GetNodeConfig503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeConfig>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetNodeConfigSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateNodeConfigUrl = (id: string) => {
  return `/api/v1/nodes/${id}/config`;
};

/**
 * @summary Change a Node's settings, every key or none
 */
export const updateNodeConfig = async (
  id: string,
  updateNodeConfigBody: UpdateNodeConfigBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SystemNodeConfig> => {
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
  return apiFetch<SystemNodeConfig>(getUpdateNodeConfigUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(updateNodeConfigBody),
  });
};

export const getUpdateNodeConfigMutationKey = () => ["updateNodeConfig"] as const;

export const getUpdateNodeConfigMutationOptions = <
  TError = ErrorType<
    | UpdateNodeConfig401
    | UpdateNodeConfig403
    | UpdateNodeConfig404
    | UpdateNodeConfig409
    | UpdateNodeConfig422
    | UpdateNodeConfig500
    | UpdateNodeConfig503
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateNodeConfig>>,
    TError,
    UpdateNodeConfigMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateNodeConfig>>,
  TError,
  UpdateNodeConfigMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateNodeConfigMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateNodeConfig>>,
    UpdateNodeConfigMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return updateNodeConfig(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateNodeConfigMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateNodeConfig>>
>;
export type UpdateNodeConfigMutationBody = UpdateNodeConfigBody;
export type UpdateNodeConfigMutationError = ErrorType<
  | UpdateNodeConfig401
  | UpdateNodeConfig403
  | UpdateNodeConfig404
  | UpdateNodeConfig409
  | UpdateNodeConfig422
  | UpdateNodeConfig500
  | UpdateNodeConfig503
>;
export type UpdateNodeConfigMutationVariables = { id: string; data: UpdateNodeConfigBody };

/**
 * @summary Change a Node's settings, every key or none
 */
export const useUpdateNodeConfig = <
  TError = ErrorType<
    | UpdateNodeConfig401
    | UpdateNodeConfig403
    | UpdateNodeConfig404
    | UpdateNodeConfig409
    | UpdateNodeConfig422
    | UpdateNodeConfig500
    | UpdateNodeConfig503
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateNodeConfig>>,
      TError,
      UpdateNodeConfigMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateNodeConfig>>,
  TError,
  UpdateNodeConfigMutationVariables,
  TContext
> => {
  return useMutation(getUpdateNodeConfigMutationOptions(options), queryClient);
};
export const getRestartNodeUrl = (id: string) => {
  return `/api/v1/nodes/${id}/restart`;
};

/**
 * @summary Restart a Node; it answers before it stops
 */
export const restartNode = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SystemRestarting> => {
  return apiFetch<SystemRestarting>(getRestartNodeUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getRestartNodeMutationKey = () => ["restartNode"] as const;

export const getRestartNodeMutationOptions = <
  TError = ErrorType<
    | RestartNode401
    | RestartNode403
    | RestartNode404
    | RestartNode409
    | RestartNode422
    | RestartNode500
    | RestartNode503
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof restartNode>>,
    TError,
    RestartNodeMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof restartNode>>,
  TError,
  RestartNodeMutationVariables,
  TContext
> => {
  const mutationKey = getRestartNodeMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof restartNode>>,
    RestartNodeMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return restartNode(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RestartNodeMutationResult = NonNullable<Awaited<ReturnType<typeof restartNode>>>;

export type RestartNodeMutationError = ErrorType<
  | RestartNode401
  | RestartNode403
  | RestartNode404
  | RestartNode409
  | RestartNode422
  | RestartNode500
  | RestartNode503
>;
export type RestartNodeMutationVariables = { id: string };

/**
 * @summary Restart a Node; it answers before it stops
 */
export const useRestartNode = <
  TError = ErrorType<
    | RestartNode401
    | RestartNode403
    | RestartNode404
    | RestartNode409
    | RestartNode422
    | RestartNode500
    | RestartNode503
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof restartNode>>,
      TError,
      RestartNodeMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof restartNode>>,
  TError,
  RestartNodeMutationVariables,
  TContext
> => {
  return useMutation(getRestartNodeMutationOptions(options), queryClient);
};
export const getGetNodeSystemUrl = (id: string) => {
  return `/api/v1/nodes/${id}/system`;
};

/**
 * @summary Report a Node's build, data dir, process and listeners
 */
export const getNodeSystem = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SystemReport> => {
  return apiFetch<SystemReport>(getGetNodeSystemUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetNodeSystemQueryKey = (id: string) => {
  return [`/api/v1/nodes/${id}/system`] as const;
};

export const getGetNodeSystemSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getNodeSystem>>,
  TError = ErrorType<
    | GetNodeSystem401
    | GetNodeSystem403
    | GetNodeSystem404
    | GetNodeSystem409
    | GetNodeSystem422
    | GetNodeSystem500
    | GetNodeSystem503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSystem>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetNodeSystemQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getNodeSystem>>> = ({ signal }) =>
    getNodeSystem(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSystem>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetNodeSystemSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getNodeSystem>>
>;
export type GetNodeSystemSuspenseQueryError = ErrorType<
  | GetNodeSystem401
  | GetNodeSystem403
  | GetNodeSystem404
  | GetNodeSystem409
  | GetNodeSystem422
  | GetNodeSystem500
  | GetNodeSystem503
>;

export function useGetNodeSystemSuspense<
  TData = Awaited<ReturnType<typeof getNodeSystem>>,
  TError = ErrorType<
    | GetNodeSystem401
    | GetNodeSystem403
    | GetNodeSystem404
    | GetNodeSystem409
    | GetNodeSystem422
    | GetNodeSystem500
    | GetNodeSystem503
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSystem>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeSystemSuspense<
  TData = Awaited<ReturnType<typeof getNodeSystem>>,
  TError = ErrorType<
    | GetNodeSystem401
    | GetNodeSystem403
    | GetNodeSystem404
    | GetNodeSystem409
    | GetNodeSystem422
    | GetNodeSystem500
    | GetNodeSystem503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSystem>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeSystemSuspense<
  TData = Awaited<ReturnType<typeof getNodeSystem>>,
  TError = ErrorType<
    | GetNodeSystem401
    | GetNodeSystem403
    | GetNodeSystem404
    | GetNodeSystem409
    | GetNodeSystem422
    | GetNodeSystem500
    | GetNodeSystem503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSystem>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Report a Node's build, data dir, process and listeners
 */

export function useGetNodeSystemSuspense<
  TData = Awaited<ReturnType<typeof getNodeSystem>>,
  TError = ErrorType<
    | GetNodeSystem401
    | GetNodeSystem403
    | GetNodeSystem404
    | GetNodeSystem409
    | GetNodeSystem422
    | GetNodeSystem500
    | GetNodeSystem503
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSystem>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetNodeSystemSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetStatusUrl = () => {
  return `/api/v1/status`;
};

/**
 * @summary Report whether the Node is Unclaimed, a Controller or a Worker
 */
export const getStatus = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<SystemStatus> => {
  return apiFetch<SystemStatus>(getGetStatusUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetStatusQueryKey = () => {
  return [`/api/v1/status`] as const;
};

export const getGetStatusSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getStatus>>,
  TError = ErrorType<GetStatus500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetStatusQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getStatus>>> = ({ signal }) =>
    getStatus({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetStatusSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getStatus>>>;
export type GetStatusSuspenseQueryError = ErrorType<GetStatus500>;

export function useGetStatusSuspense<
  TData = Awaited<ReturnType<typeof getStatus>>,
  TError = ErrorType<GetStatus500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetStatusSuspense<
  TData = Awaited<ReturnType<typeof getStatus>>,
  TError = ErrorType<GetStatus500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetStatusSuspense<
  TData = Awaited<ReturnType<typeof getStatus>>,
  TError = ErrorType<GetStatus500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Report whether the Node is Unclaimed, a Controller or a Worker
 */

export function useGetStatusSuspense<
  TData = Awaited<ReturnType<typeof getStatus>>,
  TError = ErrorType<GetStatus500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetStatusSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetSystemUrl = () => {
  return `/api/v1/system`;
};

/**
 * @summary Report the build, data dir, process and listeners
 */
export const getSystem = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<SystemReport> => {
  return apiFetch<SystemReport>(getGetSystemUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetSystemQueryKey = () => {
  return [`/api/v1/system`] as const;
};

export const getGetSystemSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSystem>>,
  TError = ErrorType<GetSystem401 | GetSystem403 | GetSystem409 | GetSystem500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSystemQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSystem>>> = ({ signal }) =>
    getSystem({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSystemSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getSystem>>>;
export type GetSystemSuspenseQueryError = ErrorType<
  GetSystem401 | GetSystem403 | GetSystem409 | GetSystem500
>;

export function useGetSystemSuspense<
  TData = Awaited<ReturnType<typeof getSystem>>,
  TError = ErrorType<GetSystem401 | GetSystem403 | GetSystem409 | GetSystem500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSystemSuspense<
  TData = Awaited<ReturnType<typeof getSystem>>,
  TError = ErrorType<GetSystem401 | GetSystem403 | GetSystem409 | GetSystem500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSystemSuspense<
  TData = Awaited<ReturnType<typeof getSystem>>,
  TError = ErrorType<GetSystem401 | GetSystem403 | GetSystem409 | GetSystem500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Report the build, data dir, process and listeners
 */

export function useGetSystemSuspense<
  TData = Awaited<ReturnType<typeof getSystem>>,
  TError = ErrorType<GetSystem401 | GetSystem403 | GetSystem409 | GetSystem500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSystem>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSystemSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
