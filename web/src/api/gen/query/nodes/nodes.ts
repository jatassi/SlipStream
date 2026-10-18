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
  DialForCode401,
  DialForCode403,
  DialForCode404,
  DialForCode409,
  DialForCode422,
  DialForCode500,
  DialInputBody,
  GetJoinCode401,
  GetJoinCode403,
  GetJoinCode404,
  GetJoinCode409,
  GetJoinCode422,
  GetJoinCode500,
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
  GetRoleBoard401,
  GetRoleBoard403,
  GetRoleBoard409,
  GetRoleBoard500,
  ListJoinCodes401,
  ListJoinCodes403,
  ListJoinCodes409,
  ListJoinCodes500,
  ListNodes401,
  ListNodes403,
  ListNodes409,
  ListNodes500,
  NodesCodeState,
  NodesJoinCode,
  NodesNode,
  NodesNodeDial,
  NodesNodeList,
  NodesOpenCodes,
  NodesOutcome,
  NodesPlanChange,
  NodesRoleBoard,
  NodesRolesChange,
  NodesSettings,
  NodesSettingsPatch,
  NodesWriteOffBody,
  PlanRoles401,
  PlanRoles403,
  PlanRoles404,
  PlanRoles409,
  PlanRoles422,
  PlanRoles500,
  RevokeJoinCode401,
  RevokeJoinCode403,
  RevokeJoinCode404,
  RevokeJoinCode409,
  RevokeJoinCode422,
  RevokeJoinCode500,
  SetNodeDial401,
  SetNodeDial403,
  SetNodeDial404,
  SetNodeDial409,
  SetNodeDial422,
  SetNodeDial500,
  SetRoles401,
  SetRoles403,
  SetRoles404,
  SetRoles409,
  SetRoles422,
  SetRoles500,
  UpdateNodeSettings401,
  UpdateNodeSettings403,
  UpdateNodeSettings409,
  UpdateNodeSettings422,
  UpdateNodeSettings500,
  WriteOff401,
  WriteOff403,
  WriteOff404,
  WriteOff409,
  WriteOff422,
  WriteOff500,
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

export const getListJoinCodesUrl = () => {
  return `/api/v1/join-codes`;
};

/**
 * @summary List the Join Codes that may still pair a Node
 */
export const listJoinCodes = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesOpenCodes> => {
  return apiFetch<NodesOpenCodes>(getListJoinCodesUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListJoinCodesQueryKey = () => {
  return [`/api/v1/join-codes`] as const;
};

export const getListJoinCodesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listJoinCodes>>,
  TError = ErrorType<ListJoinCodes401 | ListJoinCodes403 | ListJoinCodes409 | ListJoinCodes500>,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listJoinCodes>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListJoinCodesQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listJoinCodes>>> = ({ signal }) =>
    listJoinCodes({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listJoinCodes>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListJoinCodesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listJoinCodes>>
>;
export type ListJoinCodesSuspenseQueryError = ErrorType<
  ListJoinCodes401 | ListJoinCodes403 | ListJoinCodes409 | ListJoinCodes500
>;

export function useListJoinCodesSuspense<
  TData = Awaited<ReturnType<typeof listJoinCodes>>,
  TError = ErrorType<ListJoinCodes401 | ListJoinCodes403 | ListJoinCodes409 | ListJoinCodes500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listJoinCodes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListJoinCodesSuspense<
  TData = Awaited<ReturnType<typeof listJoinCodes>>,
  TError = ErrorType<ListJoinCodes401 | ListJoinCodes403 | ListJoinCodes409 | ListJoinCodes500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listJoinCodes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListJoinCodesSuspense<
  TData = Awaited<ReturnType<typeof listJoinCodes>>,
  TError = ErrorType<ListJoinCodes401 | ListJoinCodes403 | ListJoinCodes409 | ListJoinCodes500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listJoinCodes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Join Codes that may still pair a Node
 */

export function useListJoinCodesSuspense<
  TData = Awaited<ReturnType<typeof listJoinCodes>>,
  TError = ErrorType<ListJoinCodes401 | ListJoinCodes403 | ListJoinCodes409 | ListJoinCodes500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listJoinCodes>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListJoinCodesSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCreateJoinCodeUrl = () => {
  return `/api/v1/join-codes`;
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
export const getRevokeJoinCodeUrl = (id: string) => {
  return `/api/v1/join-codes/${id}`;
};

/**
 * @summary Revoke a Join Code, so no Node pairs with it
 */
export const revokeJoinCode = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getRevokeJoinCodeUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getRevokeJoinCodeMutationKey = () => ["revokeJoinCode"] as const;

export const getRevokeJoinCodeMutationOptions = <
  TError = ErrorType<
    | RevokeJoinCode401
    | RevokeJoinCode403
    | RevokeJoinCode404
    | RevokeJoinCode409
    | RevokeJoinCode422
    | RevokeJoinCode500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof revokeJoinCode>>,
    TError,
    RevokeJoinCodeMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof revokeJoinCode>>,
  TError,
  RevokeJoinCodeMutationVariables,
  TContext
> => {
  const mutationKey = getRevokeJoinCodeMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof revokeJoinCode>>,
    RevokeJoinCodeMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return revokeJoinCode(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RevokeJoinCodeMutationResult = NonNullable<Awaited<ReturnType<typeof revokeJoinCode>>>;

export type RevokeJoinCodeMutationError = ErrorType<
  | RevokeJoinCode401
  | RevokeJoinCode403
  | RevokeJoinCode404
  | RevokeJoinCode409
  | RevokeJoinCode422
  | RevokeJoinCode500
>;
export type RevokeJoinCodeMutationVariables = { id: string };

/**
 * @summary Revoke a Join Code, so no Node pairs with it
 */
export const useRevokeJoinCode = <
  TError = ErrorType<
    | RevokeJoinCode401
    | RevokeJoinCode403
    | RevokeJoinCode404
    | RevokeJoinCode409
    | RevokeJoinCode422
    | RevokeJoinCode500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof revokeJoinCode>>,
      TError,
      RevokeJoinCodeMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof revokeJoinCode>>,
  TError,
  RevokeJoinCodeMutationVariables,
  TContext
> => {
  return useMutation(getRevokeJoinCodeMutationOptions(options), queryClient);
};
export const getGetJoinCodeUrl = (id: string) => {
  return `/api/v1/join-codes/${id}`;
};

/**
 * @summary Get whether a Join Code is open, used or expired, and its dial
 */
export const getJoinCode = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesCodeState> => {
  return apiFetch<NodesCodeState>(getGetJoinCodeUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetJoinCodeQueryKey = (id: string) => {
  return [`/api/v1/join-codes/${id}`] as const;
};

export const getGetJoinCodeSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getJoinCode>>,
  TError = ErrorType<
    | GetJoinCode401
    | GetJoinCode403
    | GetJoinCode404
    | GetJoinCode409
    | GetJoinCode422
    | GetJoinCode500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getJoinCode>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetJoinCodeQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getJoinCode>>> = ({ signal }) =>
    getJoinCode(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getJoinCode>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetJoinCodeSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getJoinCode>>>;
export type GetJoinCodeSuspenseQueryError = ErrorType<
  | GetJoinCode401
  | GetJoinCode403
  | GetJoinCode404
  | GetJoinCode409
  | GetJoinCode422
  | GetJoinCode500
>;

export function useGetJoinCodeSuspense<
  TData = Awaited<ReturnType<typeof getJoinCode>>,
  TError = ErrorType<
    | GetJoinCode401
    | GetJoinCode403
    | GetJoinCode404
    | GetJoinCode409
    | GetJoinCode422
    | GetJoinCode500
  >,
>(
  id: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getJoinCode>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetJoinCodeSuspense<
  TData = Awaited<ReturnType<typeof getJoinCode>>,
  TError = ErrorType<
    | GetJoinCode401
    | GetJoinCode403
    | GetJoinCode404
    | GetJoinCode409
    | GetJoinCode422
    | GetJoinCode500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getJoinCode>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetJoinCodeSuspense<
  TData = Awaited<ReturnType<typeof getJoinCode>>,
  TError = ErrorType<
    | GetJoinCode401
    | GetJoinCode403
    | GetJoinCode404
    | GetJoinCode409
    | GetJoinCode422
    | GetJoinCode500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getJoinCode>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get whether a Join Code is open, used or expired, and its dial
 */

export function useGetJoinCodeSuspense<
  TData = Awaited<ReturnType<typeof getJoinCode>>,
  TError = ErrorType<
    | GetJoinCode401
    | GetJoinCode403
    | GetJoinCode404
    | GetJoinCode409
    | GetJoinCode422
    | GetJoinCode500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getJoinCode>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetJoinCodeSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getDialForCodeUrl = (id: string) => {
  return `/api/v1/join-codes/${id}/dial`;
};

/**
 * @summary Dial the Node waiting at a Node address, which pairs with the code over that session
 */
export const dialForCode = async (
  id: string,
  dialInputBody: DialInputBody,
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
  return apiFetch<void>(getDialForCodeUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(dialInputBody),
  });
};

export const getDialForCodeMutationKey = () => ["dialForCode"] as const;

export const getDialForCodeMutationOptions = <
  TError = ErrorType<
    | DialForCode401
    | DialForCode403
    | DialForCode404
    | DialForCode409
    | DialForCode422
    | DialForCode500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof dialForCode>>,
    TError,
    DialForCodeMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof dialForCode>>,
  TError,
  DialForCodeMutationVariables,
  TContext
> => {
  const mutationKey = getDialForCodeMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof dialForCode>>,
    DialForCodeMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return dialForCode(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DialForCodeMutationResult = NonNullable<Awaited<ReturnType<typeof dialForCode>>>;
export type DialForCodeMutationBody = DialInputBody;
export type DialForCodeMutationError = ErrorType<
  | DialForCode401
  | DialForCode403
  | DialForCode404
  | DialForCode409
  | DialForCode422
  | DialForCode500
>;
export type DialForCodeMutationVariables = { id: string; data: DialInputBody };

/**
 * @summary Dial the Node waiting at a Node address, which pairs with the code over that session
 */
export const useDialForCode = <
  TError = ErrorType<
    | DialForCode401
    | DialForCode403
    | DialForCode404
    | DialForCode409
    | DialForCode422
    | DialForCode500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof dialForCode>>,
      TError,
      DialForCodeMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof dialForCode>>,
  TError,
  DialForCodeMutationVariables,
  TContext
> => {
  return useMutation(getDialForCodeMutationOptions(options), queryClient);
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

export const getGetRoleBoardUrl = () => {
  return `/api/v1/nodes/roles`;
};

/**
 * @summary Get every Node's Roles and the plan's verdict on every drag
 */
export const getRoleBoard = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesRoleBoard> => {
  return apiFetch<NodesRoleBoard>(getGetRoleBoardUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetRoleBoardQueryKey = () => {
  return [`/api/v1/nodes/roles`] as const;
};

export const getGetRoleBoardSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getRoleBoard>>,
  TError = ErrorType<GetRoleBoard401 | GetRoleBoard403 | GetRoleBoard409 | GetRoleBoard500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRoleBoard>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetRoleBoardQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getRoleBoard>>> = ({ signal }) =>
    getRoleBoard({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRoleBoard>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetRoleBoardSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getRoleBoard>>>;
export type GetRoleBoardSuspenseQueryError = ErrorType<
  GetRoleBoard401 | GetRoleBoard403 | GetRoleBoard409 | GetRoleBoard500
>;

export function useGetRoleBoardSuspense<
  TData = Awaited<ReturnType<typeof getRoleBoard>>,
  TError = ErrorType<GetRoleBoard401 | GetRoleBoard403 | GetRoleBoard409 | GetRoleBoard500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRoleBoard>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetRoleBoardSuspense<
  TData = Awaited<ReturnType<typeof getRoleBoard>>,
  TError = ErrorType<GetRoleBoard401 | GetRoleBoard403 | GetRoleBoard409 | GetRoleBoard500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRoleBoard>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetRoleBoardSuspense<
  TData = Awaited<ReturnType<typeof getRoleBoard>>,
  TError = ErrorType<GetRoleBoard401 | GetRoleBoard403 | GetRoleBoard409 | GetRoleBoard500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRoleBoard>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get every Node's Roles and the plan's verdict on every drag
 */

export function useGetRoleBoardSuspense<
  TData = Awaited<ReturnType<typeof getRoleBoard>>,
  TError = ErrorType<GetRoleBoard401 | GetRoleBoard403 | GetRoleBoard409 | GetRoleBoard500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getRoleBoard>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetRoleBoardSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetRolesUrl = () => {
  return `/api/v1/nodes/roles`;
};

/**
 * @summary Apply a change of Roles; the Nodes in step it changes have their new Roles when this answers
 */
export const setRoles = async (
  nodesRolesChange: NodesRolesChange,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesOutcome> => {
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
  return apiFetch<NodesOutcome>(getSetRolesUrl(), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(nodesRolesChange),
  });
};

export const getSetRolesMutationKey = () => ["setRoles"] as const;

export const getSetRolesMutationOptions = <
  TError = ErrorType<
    SetRoles401 | SetRoles403 | SetRoles404 | SetRoles409 | SetRoles422 | SetRoles500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setRoles>>,
    TError,
    SetRolesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setRoles>>,
  TError,
  SetRolesMutationVariables,
  TContext
> => {
  const mutationKey = getSetRolesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setRoles>>,
    SetRolesMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return setRoles(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetRolesMutationResult = NonNullable<Awaited<ReturnType<typeof setRoles>>>;
export type SetRolesMutationBody = NodesRolesChange;
export type SetRolesMutationError = ErrorType<
  SetRoles401 | SetRoles403 | SetRoles404 | SetRoles409 | SetRoles422 | SetRoles500
>;
export type SetRolesMutationVariables = { data: NodesRolesChange };

/**
 * @summary Apply a change of Roles; the Nodes in step it changes have their new Roles when this answers
 */
export const useSetRoles = <
  TError = ErrorType<
    SetRoles401 | SetRoles403 | SetRoles404 | SetRoles409 | SetRoles422 | SetRoles500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setRoles>>,
      TError,
      SetRolesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setRoles>>,
  TError,
  SetRolesMutationVariables,
  TContext
> => {
  return useMutation(getSetRolesMutationOptions(options), queryClient);
};
export const getPlanRolesUrl = () => {
  return `/api/v1/nodes/roles/plan`;
};

/**
 * @summary Plan a change of Roles without applying it
 */
export const planRoles = async (
  nodesPlanChange: NodesPlanChange,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesOutcome> => {
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
  return apiFetch<NodesOutcome>(getPlanRolesUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(nodesPlanChange),
  });
};

export const getPlanRolesMutationKey = () => ["planRoles"] as const;

export const getPlanRolesMutationOptions = <
  TError = ErrorType<
    PlanRoles401 | PlanRoles403 | PlanRoles404 | PlanRoles409 | PlanRoles422 | PlanRoles500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof planRoles>>,
    TError,
    PlanRolesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof planRoles>>,
  TError,
  PlanRolesMutationVariables,
  TContext
> => {
  const mutationKey = getPlanRolesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof planRoles>>,
    PlanRolesMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return planRoles(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PlanRolesMutationResult = NonNullable<Awaited<ReturnType<typeof planRoles>>>;
export type PlanRolesMutationBody = NodesPlanChange;
export type PlanRolesMutationError = ErrorType<
  PlanRoles401 | PlanRoles403 | PlanRoles404 | PlanRoles409 | PlanRoles422 | PlanRoles500
>;
export type PlanRolesMutationVariables = { data: NodesPlanChange };

/**
 * @summary Plan a change of Roles without applying it
 */
export const usePlanRoles = <
  TError = ErrorType<
    PlanRoles401 | PlanRoles403 | PlanRoles404 | PlanRoles409 | PlanRoles422 | PlanRoles500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof planRoles>>,
      TError,
      PlanRolesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof planRoles>>,
  TError,
  PlanRolesMutationVariables,
  TContext
> => {
  return useMutation(getPlanRolesMutationOptions(options), queryClient);
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

export const getSetNodeDialUrl = (id: string) => {
  return `/api/v1/nodes/${id}/dial`;
};

/**
 * @summary Set the Node address the Controller dials a Node at, or stop dialing it
 */
export const setNodeDial = async (
  id: string,
  nodesNodeDial: NodesNodeDial,
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
  return apiFetch<NodesNode>(getSetNodeDialUrl(id), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(nodesNodeDial),
  });
};

export const getSetNodeDialMutationKey = () => ["setNodeDial"] as const;

export const getSetNodeDialMutationOptions = <
  TError = ErrorType<
    | SetNodeDial401
    | SetNodeDial403
    | SetNodeDial404
    | SetNodeDial409
    | SetNodeDial422
    | SetNodeDial500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setNodeDial>>,
    TError,
    SetNodeDialMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setNodeDial>>,
  TError,
  SetNodeDialMutationVariables,
  TContext
> => {
  const mutationKey = getSetNodeDialMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setNodeDial>>,
    SetNodeDialMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return setNodeDial(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetNodeDialMutationResult = NonNullable<Awaited<ReturnType<typeof setNodeDial>>>;
export type SetNodeDialMutationBody = NodesNodeDial;
export type SetNodeDialMutationError = ErrorType<
  | SetNodeDial401
  | SetNodeDial403
  | SetNodeDial404
  | SetNodeDial409
  | SetNodeDial422
  | SetNodeDial500
>;
export type SetNodeDialMutationVariables = { id: string; data: NodesNodeDial };

/**
 * @summary Set the Node address the Controller dials a Node at, or stop dialing it
 */
export const useSetNodeDial = <
  TError = ErrorType<
    | SetNodeDial401
    | SetNodeDial403
    | SetNodeDial404
    | SetNodeDial409
    | SetNodeDial422
    | SetNodeDial500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setNodeDial>>,
      TError,
      SetNodeDialMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setNodeDial>>,
  TError,
  SetNodeDialMutationVariables,
  TContext
> => {
  return useMutation(getSetNodeDialMutationOptions(options), queryClient);
};
export const getWriteOffUrl = (id: string) => {
  return `/api/v1/nodes/${id}/write-offs`;
};

/**
 * @summary Give up the work a Node holds for Roles, live or offline; its files are kept
 */
export const writeOff = async (
  id: string,
  nodesWriteOffBody: NodesWriteOffBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NodesOutcome> => {
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
  return apiFetch<NodesOutcome>(getWriteOffUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(nodesWriteOffBody),
  });
};

export const getWriteOffMutationKey = () => ["writeOff"] as const;

export const getWriteOffMutationOptions = <
  TError = ErrorType<
    WriteOff401 | WriteOff403 | WriteOff404 | WriteOff409 | WriteOff422 | WriteOff500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof writeOff>>,
    TError,
    WriteOffMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof writeOff>>,
  TError,
  WriteOffMutationVariables,
  TContext
> => {
  const mutationKey = getWriteOffMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof writeOff>>,
    WriteOffMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return writeOff(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type WriteOffMutationResult = NonNullable<Awaited<ReturnType<typeof writeOff>>>;
export type WriteOffMutationBody = NodesWriteOffBody;
export type WriteOffMutationError = ErrorType<
  WriteOff401 | WriteOff403 | WriteOff404 | WriteOff409 | WriteOff422 | WriteOff500
>;
export type WriteOffMutationVariables = { id: string; data: NodesWriteOffBody };

/**
 * @summary Give up the work a Node holds for Roles, live or offline; its files are kept
 */
export const useWriteOff = <
  TError = ErrorType<
    WriteOff401 | WriteOff403 | WriteOff404 | WriteOff409 | WriteOff422 | WriteOff500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof writeOff>>,
      TError,
      WriteOffMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof writeOff>>,
  TError,
  WriteOffMutationVariables,
  TContext
> => {
  return useMutation(getWriteOffMutationOptions(options), queryClient);
};
