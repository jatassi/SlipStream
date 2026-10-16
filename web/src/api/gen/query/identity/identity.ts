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
  ChangePIN401,
  ChangePIN403,
  ChangePIN409,
  ChangePIN422,
  ChangePIN429,
  ChangePIN500,
  ClaimController401,
  ClaimController403,
  ClaimController409,
  ClaimController422,
  ClaimController500,
  CreateSession401,
  CreateSession403,
  CreateSession422,
  CreateSession429,
  CreateSession500,
  DeleteSession401,
  DeleteSession403,
  DeleteSession409,
  DeleteSession500,
  GetAPIKey401,
  GetAPIKey403,
  GetAPIKey409,
  GetAPIKey500,
  GetIdentitySettings401,
  GetIdentitySettings403,
  GetIdentitySettings409,
  GetIdentitySettings500,
  GetSession401,
  GetSession403,
  GetSession409,
  GetSession500,
  IdentityApiKeyMade,
  IdentityClaimBody,
  IdentityClaimed,
  IdentityPinChange,
  IdentityReconfirmation,
  IdentityRegeneratedAPIKey,
  IdentitySession,
  IdentitySettings,
  IdentitySettingsPatch,
  IdentitySignInBody,
  ReconfirmSession401,
  ReconfirmSession403,
  ReconfirmSession409,
  ReconfirmSession422,
  ReconfirmSession429,
  ReconfirmSession500,
  RegenerateAPIKey401,
  RegenerateAPIKey403,
  RegenerateAPIKey409,
  RegenerateAPIKey500,
  UpdateIdentitySettings401,
  UpdateIdentitySettings403,
  UpdateIdentitySettings409,
  UpdateIdentitySettings422,
  UpdateIdentitySettings500,
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

export const getChangePINUrl = () => {
  return `/api/v1/account/pin`;
};

/**
 * @summary Change your PIN, or set a new one after a passkey sign-in on a turned-off PIN
 */
export const changePIN = async (
  identityPinChange: IdentityPinChange,
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
  return apiFetch<void>(getChangePINUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(identityPinChange),
  });
};

export const getChangePINMutationKey = () => ["changePIN"] as const;

export const getChangePINMutationOptions = <
  TError = ErrorType<
    ChangePIN401 | ChangePIN403 | ChangePIN409 | ChangePIN422 | ChangePIN429 | ChangePIN500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof changePIN>>,
    TError,
    ChangePINMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof changePIN>>,
  TError,
  ChangePINMutationVariables,
  TContext
> => {
  const mutationKey = getChangePINMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof changePIN>>,
    ChangePINMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return changePIN(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ChangePINMutationResult = NonNullable<Awaited<ReturnType<typeof changePIN>>>;
export type ChangePINMutationBody = IdentityPinChange;
export type ChangePINMutationError = ErrorType<
  ChangePIN401 | ChangePIN403 | ChangePIN409 | ChangePIN422 | ChangePIN429 | ChangePIN500
>;
export type ChangePINMutationVariables = { data: IdentityPinChange };

/**
 * @summary Change your PIN, or set a new one after a passkey sign-in on a turned-off PIN
 */
export const useChangePIN = <
  TError = ErrorType<
    ChangePIN401 | ChangePIN403 | ChangePIN409 | ChangePIN422 | ChangePIN429 | ChangePIN500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof changePIN>>,
      TError,
      ChangePINMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof changePIN>>,
  TError,
  ChangePINMutationVariables,
  TContext
> => {
  return useMutation(getChangePINMutationOptions(options), queryClient);
};
export const getGetAPIKeyUrl = () => {
  return `/api/v1/api-key`;
};

/**
 * @summary Get when the API key was made
 */
export const getAPIKey = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityApiKeyMade> => {
  return apiFetch<IdentityApiKeyMade>(getGetAPIKeyUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetAPIKeyQueryKey = () => {
  return [`/api/v1/api-key`] as const;
};

export const getGetAPIKeySuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getAPIKey>>,
  TError = ErrorType<GetAPIKey401 | GetAPIKey403 | GetAPIKey409 | GetAPIKey500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAPIKey>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetAPIKeyQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getAPIKey>>> = ({ signal }) =>
    getAPIKey({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAPIKey>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetAPIKeySuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getAPIKey>>>;
export type GetAPIKeySuspenseQueryError = ErrorType<
  GetAPIKey401 | GetAPIKey403 | GetAPIKey409 | GetAPIKey500
>;

export function useGetAPIKeySuspense<
  TData = Awaited<ReturnType<typeof getAPIKey>>,
  TError = ErrorType<GetAPIKey401 | GetAPIKey403 | GetAPIKey409 | GetAPIKey500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAPIKey>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetAPIKeySuspense<
  TData = Awaited<ReturnType<typeof getAPIKey>>,
  TError = ErrorType<GetAPIKey401 | GetAPIKey403 | GetAPIKey409 | GetAPIKey500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAPIKey>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetAPIKeySuspense<
  TData = Awaited<ReturnType<typeof getAPIKey>>,
  TError = ErrorType<GetAPIKey401 | GetAPIKey403 | GetAPIKey409 | GetAPIKey500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAPIKey>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get when the API key was made
 */

export function useGetAPIKeySuspense<
  TData = Awaited<ReturnType<typeof getAPIKey>>,
  TError = ErrorType<GetAPIKey401 | GetAPIKey403 | GetAPIKey409 | GetAPIKey500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAPIKey>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetAPIKeySuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getRegenerateAPIKeyUrl = () => {
  return `/api/v1/api-key`;
};

/**
 * @summary Replace the API key, shown this once; the old one stops working
 */
export const regenerateAPIKey = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityRegeneratedAPIKey> => {
  return apiFetch<IdentityRegeneratedAPIKey>(getRegenerateAPIKeyUrl(), {
    ...options,
    method: "POST",
  });
};

export const getRegenerateAPIKeyMutationKey = () => ["regenerateAPIKey"] as const;

export const getRegenerateAPIKeyMutationOptions = <
  TError = ErrorType<
    RegenerateAPIKey401 | RegenerateAPIKey403 | RegenerateAPIKey409 | RegenerateAPIKey500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof regenerateAPIKey>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof regenerateAPIKey>>, TError, void, TContext> => {
  const mutationKey = getRegenerateAPIKeyMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof regenerateAPIKey>>, void> = () => {
    return regenerateAPIKey(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RegenerateAPIKeyMutationResult = NonNullable<
  Awaited<ReturnType<typeof regenerateAPIKey>>
>;

export type RegenerateAPIKeyMutationError = ErrorType<
  RegenerateAPIKey401 | RegenerateAPIKey403 | RegenerateAPIKey409 | RegenerateAPIKey500
>;

/**
 * @summary Replace the API key, shown this once; the old one stops working
 */
export const useRegenerateAPIKey = <
  TError = ErrorType<
    RegenerateAPIKey401 | RegenerateAPIKey403 | RegenerateAPIKey409 | RegenerateAPIKey500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof regenerateAPIKey>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof regenerateAPIKey>>, TError, void, TContext> => {
  return useMutation(getRegenerateAPIKeyMutationOptions(options), queryClient);
};
export const getGetIdentitySettingsUrl = () => {
  return `/api/v1/identity/settings`;
};

/**
 * @summary Get the sign-in limits and lifetimes, and the address links use
 */
export const getIdentitySettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentitySettings> => {
  return apiFetch<IdentitySettings>(getGetIdentitySettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetIdentitySettingsQueryKey = () => {
  return [`/api/v1/identity/settings`] as const;
};

export const getGetIdentitySettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getIdentitySettings>>,
  TError = ErrorType<
    | GetIdentitySettings401
    | GetIdentitySettings403
    | GetIdentitySettings409
    | GetIdentitySettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIdentitySettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetIdentitySettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getIdentitySettings>>> = ({ signal }) =>
    getIdentitySettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIdentitySettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetIdentitySettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getIdentitySettings>>
>;
export type GetIdentitySettingsSuspenseQueryError = ErrorType<
  GetIdentitySettings401 | GetIdentitySettings403 | GetIdentitySettings409 | GetIdentitySettings500
>;

export function useGetIdentitySettingsSuspense<
  TData = Awaited<ReturnType<typeof getIdentitySettings>>,
  TError = ErrorType<
    | GetIdentitySettings401
    | GetIdentitySettings403
    | GetIdentitySettings409
    | GetIdentitySettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIdentitySettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetIdentitySettingsSuspense<
  TData = Awaited<ReturnType<typeof getIdentitySettings>>,
  TError = ErrorType<
    | GetIdentitySettings401
    | GetIdentitySettings403
    | GetIdentitySettings409
    | GetIdentitySettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIdentitySettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetIdentitySettingsSuspense<
  TData = Awaited<ReturnType<typeof getIdentitySettings>>,
  TError = ErrorType<
    | GetIdentitySettings401
    | GetIdentitySettings403
    | GetIdentitySettings409
    | GetIdentitySettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIdentitySettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the sign-in limits and lifetimes, and the address links use
 */

export function useGetIdentitySettingsSuspense<
  TData = Awaited<ReturnType<typeof getIdentitySettings>>,
  TError = ErrorType<
    | GetIdentitySettings401
    | GetIdentitySettings403
    | GetIdentitySettings409
    | GetIdentitySettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getIdentitySettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetIdentitySettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateIdentitySettingsUrl = () => {
  return `/api/v1/identity/settings`;
};

/**
 * @summary Change the sign-in limits and lifetimes, or the address links use
 */
export const updateIdentitySettings = async (
  identitySettingsPatch: IdentitySettingsPatch,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentitySettings> => {
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
  return apiFetch<IdentitySettings>(getUpdateIdentitySettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(identitySettingsPatch),
  });
};

export const getUpdateIdentitySettingsMutationKey = () => ["updateIdentitySettings"] as const;

export const getUpdateIdentitySettingsMutationOptions = <
  TError = ErrorType<
    | UpdateIdentitySettings401
    | UpdateIdentitySettings403
    | UpdateIdentitySettings409
    | UpdateIdentitySettings422
    | UpdateIdentitySettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateIdentitySettings>>,
    TError,
    UpdateIdentitySettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateIdentitySettings>>,
  TError,
  UpdateIdentitySettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateIdentitySettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateIdentitySettings>>,
    UpdateIdentitySettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateIdentitySettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateIdentitySettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateIdentitySettings>>
>;
export type UpdateIdentitySettingsMutationBody = IdentitySettingsPatch;
export type UpdateIdentitySettingsMutationError = ErrorType<
  | UpdateIdentitySettings401
  | UpdateIdentitySettings403
  | UpdateIdentitySettings409
  | UpdateIdentitySettings422
  | UpdateIdentitySettings500
>;
export type UpdateIdentitySettingsMutationVariables = { data: IdentitySettingsPatch };

/**
 * @summary Change the sign-in limits and lifetimes, or the address links use
 */
export const useUpdateIdentitySettings = <
  TError = ErrorType<
    | UpdateIdentitySettings401
    | UpdateIdentitySettings403
    | UpdateIdentitySettings409
    | UpdateIdentitySettings422
    | UpdateIdentitySettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateIdentitySettings>>,
      TError,
      UpdateIdentitySettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateIdentitySettings>>,
  TError,
  UpdateIdentitySettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateIdentitySettingsMutationOptions(options), queryClient);
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
  TError = ErrorType<DeleteSession401 | DeleteSession403 | DeleteSession409 | DeleteSession500>,
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
  DeleteSession401 | DeleteSession403 | DeleteSession409 | DeleteSession500
>;

/**
 * @summary Sign out
 */
export const useDeleteSession = <
  TError = ErrorType<DeleteSession401 | DeleteSession403 | DeleteSession409 | DeleteSession500>,
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
  TError = ErrorType<GetSession401 | GetSession403 | GetSession409 | GetSession500>,
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
export type GetSessionSuspenseQueryError = ErrorType<
  GetSession401 | GetSession403 | GetSession409 | GetSession500
>;

export function useGetSessionSuspense<
  TData = Awaited<ReturnType<typeof getSession>>,
  TError = ErrorType<GetSession401 | GetSession403 | GetSession409 | GetSession500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSession>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSessionSuspense<
  TData = Awaited<ReturnType<typeof getSession>>,
  TError = ErrorType<GetSession401 | GetSession403 | GetSession409 | GetSession500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSession>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSessionSuspense<
  TData = Awaited<ReturnType<typeof getSession>>,
  TError = ErrorType<GetSession401 | GetSession403 | GetSession409 | GetSession500>,
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
  TError = ErrorType<GetSession401 | GetSession403 | GetSession409 | GetSession500>,
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
  TError = ErrorType<
    CreateSession401 | CreateSession403 | CreateSession422 | CreateSession429 | CreateSession500
  >,
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
  CreateSession401 | CreateSession403 | CreateSession422 | CreateSession429 | CreateSession500
>;
export type CreateSessionMutationVariables = { data: IdentitySignInBody };

/**
 * @summary Sign in with email and PIN
 */
export const useCreateSession = <
  TError = ErrorType<
    CreateSession401 | CreateSession403 | CreateSession422 | CreateSession429 | CreateSession500
  >,
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
export const getReconfirmSessionUrl = () => {
  return `/api/v1/session/reconfirm`;
};

/**
 * @summary Confirm it's you with your PIN, for actions that change who can get in
 */
export const reconfirmSession = async (
  identityReconfirmation: IdentityReconfirmation,
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
  return apiFetch<void>(getReconfirmSessionUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(identityReconfirmation),
  });
};

export const getReconfirmSessionMutationKey = () => ["reconfirmSession"] as const;

export const getReconfirmSessionMutationOptions = <
  TError = ErrorType<
    | ReconfirmSession401
    | ReconfirmSession403
    | ReconfirmSession409
    | ReconfirmSession422
    | ReconfirmSession429
    | ReconfirmSession500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof reconfirmSession>>,
    TError,
    ReconfirmSessionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof reconfirmSession>>,
  TError,
  ReconfirmSessionMutationVariables,
  TContext
> => {
  const mutationKey = getReconfirmSessionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof reconfirmSession>>,
    ReconfirmSessionMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return reconfirmSession(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ReconfirmSessionMutationResult = NonNullable<
  Awaited<ReturnType<typeof reconfirmSession>>
>;
export type ReconfirmSessionMutationBody = IdentityReconfirmation;
export type ReconfirmSessionMutationError = ErrorType<
  | ReconfirmSession401
  | ReconfirmSession403
  | ReconfirmSession409
  | ReconfirmSession422
  | ReconfirmSession429
  | ReconfirmSession500
>;
export type ReconfirmSessionMutationVariables = { data: IdentityReconfirmation };

/**
 * @summary Confirm it's you with your PIN, for actions that change who can get in
 */
export const useReconfirmSession = <
  TError = ErrorType<
    | ReconfirmSession401
    | ReconfirmSession403
    | ReconfirmSession409
    | ReconfirmSession422
    | ReconfirmSession429
    | ReconfirmSession500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof reconfirmSession>>,
      TError,
      ReconfirmSessionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof reconfirmSession>>,
  TError,
  ReconfirmSessionMutationVariables,
  TContext
> => {
  return useMutation(getReconfirmSessionMutationOptions(options), queryClient);
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
