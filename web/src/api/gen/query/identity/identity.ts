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
  ClaimController401,
  ClaimController403,
  ClaimController409,
  ClaimController422,
  ClaimController500,
  CreateSession401,
  CreateSession422,
  CreateSession500,
  DeleteSession401,
  DeleteSession403,
  DeleteSession500,
  GetSession401,
  GetSession403,
  GetSession500,
  IdentityClaimBody,
  IdentityClaimed,
  IdentitySession,
  IdentitySignInBody,
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

export const getDeleteSessionUrl = () => {
  return `/api/v1/session`;
};

/**
 * @summary Sign out
 */
export const deleteSession = async (options?: Parameters<typeof apiFetch>[1]): Promise<void> => {
  return apiFetch<void>(getDeleteSessionUrl(), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteSessionMutationKey = () => ["deleteSession"] as const;

export const getDeleteSessionMutationOptions = <
  TError = ErrorType<DeleteSession401 | DeleteSession403 | DeleteSession500>,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<Awaited<ReturnType<typeof deleteSession>>, TError, void, TContext>;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof deleteSession>>, TError, void, TContext> => {
  const mutationKey = getDeleteSessionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteSession>>, void> = () => {
    return deleteSession(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteSessionMutationResult = NonNullable<Awaited<ReturnType<typeof deleteSession>>>;

export type DeleteSessionMutationError = ErrorType<
  DeleteSession401 | DeleteSession403 | DeleteSession500
>;

/**
 * @summary Sign out
 */
export const useDeleteSession = <
  TError = ErrorType<DeleteSession401 | DeleteSession403 | DeleteSession500>,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteSession>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof deleteSession>>, TError, void, TContext> => {
  return useMutation(getDeleteSessionMutationOptions(options), queryClient);
};
export const getGetSessionUrl = () => {
  return `/api/v1/session`;
};

/**
 * @summary Get the signed-in Account
 */
export const getSession = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentitySession> => {
  return apiFetch<IdentitySession>(getGetSessionUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetSessionQueryKey = () => {
  return [`/api/v1/session`] as const;
};

export const getGetSessionSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSession>>,
  TError = ErrorType<GetSession401 | GetSession403 | GetSession500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSession>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSessionQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSession>>> = ({ signal }) =>
    getSession({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSession>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSessionSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getSession>>>;
export type GetSessionSuspenseQueryError = ErrorType<GetSession401 | GetSession403 | GetSession500>;

export function useGetSessionSuspense<
  TData = Awaited<ReturnType<typeof getSession>>,
  TError = ErrorType<GetSession401 | GetSession403 | GetSession500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSession>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSessionSuspense<
  TData = Awaited<ReturnType<typeof getSession>>,
  TError = ErrorType<GetSession401 | GetSession403 | GetSession500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSession>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSessionSuspense<
  TData = Awaited<ReturnType<typeof getSession>>,
  TError = ErrorType<GetSession401 | GetSession403 | GetSession500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSession>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the signed-in Account
 */

export function useGetSessionSuspense<
  TData = Awaited<ReturnType<typeof getSession>>,
  TError = ErrorType<GetSession401 | GetSession403 | GetSession500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSession>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSessionSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCreateSessionUrl = () => {
  return `/api/v1/session`;
};

/**
 * @summary Sign in with email and PIN
 */
export const createSession = async (
  identitySignInBody: IdentitySignInBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentitySession> => {
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
  return apiFetch<IdentitySession>(getCreateSessionUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(identitySignInBody),
  });
};

export const getCreateSessionMutationKey = () => ["createSession"] as const;

export const getCreateSessionMutationOptions = <
  TError = ErrorType<CreateSession401 | CreateSession422 | CreateSession500>,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof createSession>>,
    TError,
    CreateSessionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof createSession>>,
  TError,
  CreateSessionMutationVariables,
  TContext
> => {
  const mutationKey = getCreateSessionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof createSession>>,
    CreateSessionMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return createSession(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateSessionMutationResult = NonNullable<Awaited<ReturnType<typeof createSession>>>;
export type CreateSessionMutationBody = IdentitySignInBody;
export type CreateSessionMutationError = ErrorType<
  CreateSession401 | CreateSession422 | CreateSession500
>;
export type CreateSessionMutationVariables = { data: IdentitySignInBody };

/**
 * @summary Sign in with email and PIN
 */
export const useCreateSession = <
  TError = ErrorType<CreateSession401 | CreateSession422 | CreateSession500>,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createSession>>,
      TError,
      CreateSessionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof createSession>>,
  TError,
  CreateSessionMutationVariables,
  TContext
> => {
  return useMutation(getCreateSessionMutationOptions(options), queryClient);
};
export const getClaimControllerUrl = () => {
  return `/api/v1/setup/controller`;
};

/**
 * @summary Create the first Admin and make this Node a Controller
 */
export const claimController = async (
  identityClaimBody: IdentityClaimBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityClaimed> => {
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
  return apiFetch<IdentityClaimed>(getClaimControllerUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(identityClaimBody),
  });
};

export const getClaimControllerMutationKey = () => ["claimController"] as const;

export const getClaimControllerMutationOptions = <
  TError = ErrorType<
    | ClaimController401
    | ClaimController403
    | ClaimController409
    | ClaimController422
    | ClaimController500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof claimController>>,
    TError,
    ClaimControllerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof claimController>>,
  TError,
  ClaimControllerMutationVariables,
  TContext
> => {
  const mutationKey = getClaimControllerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof claimController>>,
    ClaimControllerMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return claimController(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ClaimControllerMutationResult = NonNullable<
  Awaited<ReturnType<typeof claimController>>
>;
export type ClaimControllerMutationBody = IdentityClaimBody;
export type ClaimControllerMutationError = ErrorType<
  | ClaimController401
  | ClaimController403
  | ClaimController409
  | ClaimController422
  | ClaimController500
>;
export type ClaimControllerMutationVariables = { data: IdentityClaimBody };

/**
 * @summary Create the first Admin and make this Node a Controller
 */
export const useClaimController = <
  TError = ErrorType<
    | ClaimController401
    | ClaimController403
    | ClaimController409
    | ClaimController422
    | ClaimController500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof claimController>>,
      TError,
      ClaimControllerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof claimController>>,
  TError,
  ClaimControllerMutationVariables,
  TContext
> => {
  return useMutation(getClaimControllerMutationOptions(options), queryClient);
};
