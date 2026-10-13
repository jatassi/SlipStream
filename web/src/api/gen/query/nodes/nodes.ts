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
  CreateJoinCode401,
  CreateJoinCode403,
  CreateJoinCode409,
  CreateJoinCode500,
  GetNode401,
  GetNode403,
  GetNode404,
  GetNode409,
  GetNode422,
  GetNode500,
  GetNodeSettings401,
  GetNodeSettings403,
  GetNodeSettings409,
  GetNodeSettings500,
  ListNodes401,
  ListNodes403,
  ListNodes409,
  ListNodes500,
  NodesJoinCode,
  NodesNode,
  NodesNodeList,
  NodesRoles,
  NodesSettings,
  NodesSettingsPatch,
  SetNodeRoles401,
  SetNodeRoles403,
  SetNodeRoles404,
  SetNodeRoles409,
  SetNodeRoles422,
  SetNodeRoles500,
  UpdateNodeSettings401,
  UpdateNodeSettings403,
  UpdateNodeSettings409,
  UpdateNodeSettings422,
  UpdateNodeSettings500,
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

export const getListNodesUrl = () => {
  return `/api/v1/nodes`;
};

/**
 * @summary List every Node with its Roles, link and facts
 */
export const listNodes = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesNodeList> => {
  return apiFetch<NodesNodeList>(getListNodesUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListNodesQueryKey = () => {
  return [`/api/v1/nodes`] as const;
};

export const getListNodesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listNodes>>,
  TError = ErrorType<ListNodes401 | ListNodes403 | ListNodes409 | ListNodes500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNodes>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListNodesQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listNodes>>> = ({ signal }) =>
    listNodes({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNodes>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListNodesSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listNodes>>>;
export type ListNodesSuspenseQueryError = ErrorType<
  ListNodes401 | ListNodes403 | ListNodes409 | ListNodes500
>;

export function useListNodesSuspense<
  TData = Awaited<ReturnType<typeof listNodes>>,
  TError = ErrorType<ListNodes401 | ListNodes403 | ListNodes409 | ListNodes500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNodes>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNodesSuspense<
  TData = Awaited<ReturnType<typeof listNodes>>,
  TError = ErrorType<ListNodes401 | ListNodes403 | ListNodes409 | ListNodes500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNodes>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNodesSuspense<
  TData = Awaited<ReturnType<typeof listNodes>>,
  TError = ErrorType<ListNodes401 | ListNodes403 | ListNodes409 | ListNodes500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNodes>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List every Node with its Roles, link and facts
 */

export function useListNodesSuspense<
  TData = Awaited<ReturnType<typeof listNodes>>,
  TError = ErrorType<ListNodes401 | ListNodes403 | ListNodes409 | ListNodes500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNodes>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListNodesSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCreateJoinCodeUrl = () => {
  return `/api/v1/nodes/join-codes`;
};

/**
 * @summary Mint a Join Code that pairs one Node with this Controller
 */
export const createJoinCode = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesJoinCode> => {
  return apiFetch<NodesJoinCode>(getCreateJoinCodeUrl(), {
    ...options,
    method: "POST",
  });
};

export const getCreateJoinCodeMutationKey = () => ["createJoinCode"] as const;

export const getCreateJoinCodeMutationOptions = <
  TError = ErrorType<CreateJoinCode401 | CreateJoinCode403 | CreateJoinCode409 | CreateJoinCode500>,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<Awaited<ReturnType<typeof createJoinCode>>, TError, void, TContext>;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof createJoinCode>>, TError, void, TContext> => {
  const mutationKey = getCreateJoinCodeMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof createJoinCode>>, void> = () => {
    return createJoinCode(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateJoinCodeMutationResult = NonNullable<Awaited<ReturnType<typeof createJoinCode>>>;

export type CreateJoinCodeMutationError = ErrorType<
  CreateJoinCode401 | CreateJoinCode403 | CreateJoinCode409 | CreateJoinCode500
>;

/**
 * @summary Mint a Join Code that pairs one Node with this Controller
 */
export const useCreateJoinCode = <
  TError = ErrorType<CreateJoinCode401 | CreateJoinCode403 | CreateJoinCode409 | CreateJoinCode500>,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createJoinCode>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof createJoinCode>>, TError, void, TContext> => {
  return useMutation(getCreateJoinCodeMutationOptions(options), queryClient);
};
export const getGetNodeSettingsUrl = () => {
  return `/api/v1/nodes/settings`;
};

/**
 * @summary Get the nodes settings
 */
export const getNodeSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesSettings> => {
  return apiFetch<NodesSettings>(getGetNodeSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetNodeSettingsQueryKey = () => {
  return [`/api/v1/nodes/settings`] as const;
};

export const getGetNodeSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getNodeSettings>>,
  TError = ErrorType<
    GetNodeSettings401 | GetNodeSettings403 | GetNodeSettings409 | GetNodeSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetNodeSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getNodeSettings>>> = ({ signal }) =>
    getNodeSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetNodeSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getNodeSettings>>
>;
export type GetNodeSettingsSuspenseQueryError = ErrorType<
  GetNodeSettings401 | GetNodeSettings403 | GetNodeSettings409 | GetNodeSettings500
>;

export function useGetNodeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNodeSettings>>,
  TError = ErrorType<
    GetNodeSettings401 | GetNodeSettings403 | GetNodeSettings409 | GetNodeSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNodeSettings>>,
  TError = ErrorType<
    GetNodeSettings401 | GetNodeSettings403 | GetNodeSettings409 | GetNodeSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNodeSettings>>,
  TError = ErrorType<
    GetNodeSettings401 | GetNodeSettings403 | GetNodeSettings409 | GetNodeSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the nodes settings
 */

export function useGetNodeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNodeSettings>>,
  TError = ErrorType<
    GetNodeSettings401 | GetNodeSettings403 | GetNodeSettings409 | GetNodeSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNodeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetNodeSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateNodeSettingsUrl = () => {
  return `/api/v1/nodes/settings`;
};

/**
 * @summary Change the nodes settings
 */
export const updateNodeSettings = async (
  nodesSettingsPatch: NodesSettingsPatch,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesSettings> => {
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
  return apiFetch<NodesSettings>(getUpdateNodeSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(nodesSettingsPatch),
  });
};

export const getUpdateNodeSettingsMutationKey = () => ["updateNodeSettings"] as const;

export const getUpdateNodeSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateNodeSettings401
    | UpdateNodeSettings403
    | UpdateNodeSettings409
    | UpdateNodeSettings422
    | UpdateNodeSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateNodeSettings>>,
    TError,
    UpdateNodeSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateNodeSettings>>,
  TError,
  UpdateNodeSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateNodeSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateNodeSettings>>,
    UpdateNodeSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateNodeSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateNodeSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateNodeSettings>>
>;
export type UpdateNodeSettingsMutationBody = NodesSettingsPatch;
export type UpdateNodeSettingsMutationError = ErrorType<
  | UpdateNodeSettings401
  | UpdateNodeSettings403
  | UpdateNodeSettings409
  | UpdateNodeSettings422
  | UpdateNodeSettings500
>;
export type UpdateNodeSettingsMutationVariables = { data: NodesSettingsPatch };

/**
 * @summary Change the nodes settings
 */
export const useUpdateNodeSettings = <
  TError = ErrorType<
    | UpdateNodeSettings401
    | UpdateNodeSettings403
    | UpdateNodeSettings409
    | UpdateNodeSettings422
    | UpdateNodeSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateNodeSettings>>,
      TError,
      UpdateNodeSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateNodeSettings>>,
  TError,
  UpdateNodeSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateNodeSettingsMutationOptions(options), queryClient);
};
export const getGetNodeUrl = (id: string) => {
  return `/api/v1/nodes/${id}`;
};

/**
 * @summary Get one Node with its Roles, link and facts
 */
export const getNode = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesNode> => {
  return apiFetch<NodesNode>(getGetNodeUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetNodeQueryKey = (id: string) => {
  return [`/api/v1/nodes/${id}`] as const;
};

export const getGetNodeSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getNode>>,
  TError = ErrorType<GetNode401 | GetNode403 | GetNode404 | GetNode409 | GetNode422 | GetNode500>,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNode>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetNodeQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getNode>>> = ({ signal }) =>
    getNode(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNode>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetNodeSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getNode>>>;
export type GetNodeSuspenseQueryError = ErrorType<
  GetNode401 | GetNode403 | GetNode404 | GetNode409 | GetNode422 | GetNode500
>;

export function useGetNodeSuspense<
  TData = Awaited<ReturnType<typeof getNode>>,
  TError = ErrorType<GetNode401 | GetNode403 | GetNode404 | GetNode409 | GetNode422 | GetNode500>,
>(
  id: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNode>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeSuspense<
  TData = Awaited<ReturnType<typeof getNode>>,
  TError = ErrorType<GetNode401 | GetNode403 | GetNode404 | GetNode409 | GetNode422 | GetNode500>,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNode>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNodeSuspense<
  TData = Awaited<ReturnType<typeof getNode>>,
  TError = ErrorType<GetNode401 | GetNode403 | GetNode404 | GetNode409 | GetNode422 | GetNode500>,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNode>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get one Node with its Roles, link and facts
 */

export function useGetNodeSuspense<
  TData = Awaited<ReturnType<typeof getNode>>,
  TError = ErrorType<GetNode401 | GetNode403 | GetNode404 | GetNode409 | GetNode422 | GetNode500>,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNode>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetNodeSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetNodeRolesUrl = (id: string) => {
  return `/api/v1/nodes/${id}/roles`;
};

/**
 * @summary Set the Roles a Node holds; a connected Node in step has them when this answers
 */
export const setNodeRoles = async (
  id: string,
  nodesRoles: NodesRoles,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesNode> => {
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
  return apiFetch<NodesNode>(getSetNodeRolesUrl(id), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(nodesRoles),
  });
};

export const getSetNodeRolesMutationKey = () => ["setNodeRoles"] as const;

export const getSetNodeRolesMutationOptions = <
  TError = ErrorType<
    | SetNodeRoles401
    | SetNodeRoles403
    | SetNodeRoles404
    | SetNodeRoles409
    | SetNodeRoles422
    | SetNodeRoles500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setNodeRoles>>,
    TError,
    SetNodeRolesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setNodeRoles>>,
  TError,
  SetNodeRolesMutationVariables,
  TContext
> => {
  const mutationKey = getSetNodeRolesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setNodeRoles>>,
    SetNodeRolesMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return setNodeRoles(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetNodeRolesMutationResult = NonNullable<Awaited<ReturnType<typeof setNodeRoles>>>;
export type SetNodeRolesMutationBody = NodesRoles;
export type SetNodeRolesMutationError = ErrorType<
  | SetNodeRoles401
  | SetNodeRoles403
  | SetNodeRoles404
  | SetNodeRoles409
  | SetNodeRoles422
  | SetNodeRoles500
>;
export type SetNodeRolesMutationVariables = { id: string; data: NodesRoles };

/**
 * @summary Set the Roles a Node holds; a connected Node in step has them when this answers
 */
export const useSetNodeRoles = <
  TError = ErrorType<
    | SetNodeRoles401
    | SetNodeRoles403
    | SetNodeRoles404
    | SetNodeRoles409
    | SetNodeRoles422
    | SetNodeRoles500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setNodeRoles>>,
      TError,
      SetNodeRolesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setNodeRoles>>,
  TError,
  SetNodeRolesMutationVariables,
  TContext
> => {
  return useMutation(getSetNodeRolesMutationOptions(options), queryClient);
};
