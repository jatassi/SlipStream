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
  CreateProfile401,
  CreateProfile403,
  CreateProfile404,
  CreateProfile409,
  CreateProfile422,
  CreateProfile500,
  CreateTarget401,
  CreateTarget403,
  CreateTarget409,
  CreateTarget422,
  CreateTarget500,
  ListProfiles401,
  ListProfiles403,
  ListProfiles409,
  ListProfiles500,
  ListTargets401,
  ListTargets403,
  ListTargets409,
  ListTargets500,
  TargetsNewProfile,
  TargetsNewTarget,
  TargetsProfile,
  TargetsProfileList,
  TargetsTarget,
  TargetsTargetList,
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

export const getListProfilesUrl = () => {
  return `/api/v1/profiles`;
};

/**
 * @summary List the Profiles
 */
export const listProfiles = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsProfileList> => {
  return apiFetch<TargetsProfileList>(getListProfilesUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListProfilesQueryKey = () => {
  return [`/api/v1/profiles`] as const;
};

export const getListProfilesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listProfiles>>,
  TError = ErrorType<ListProfiles401 | ListProfiles403 | ListProfiles409 | ListProfiles500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listProfiles>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListProfilesQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listProfiles>>> = ({ signal }) =>
    listProfiles({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listProfiles>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListProfilesSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listProfiles>>>;
export type ListProfilesSuspenseQueryError = ErrorType<
  ListProfiles401 | ListProfiles403 | ListProfiles409 | ListProfiles500
>;

export function useListProfilesSuspense<
  TData = Awaited<ReturnType<typeof listProfiles>>,
  TError = ErrorType<ListProfiles401 | ListProfiles403 | ListProfiles409 | ListProfiles500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listProfiles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListProfilesSuspense<
  TData = Awaited<ReturnType<typeof listProfiles>>,
  TError = ErrorType<ListProfiles401 | ListProfiles403 | ListProfiles409 | ListProfiles500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listProfiles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListProfilesSuspense<
  TData = Awaited<ReturnType<typeof listProfiles>>,
  TError = ErrorType<ListProfiles401 | ListProfiles403 | ListProfiles409 | ListProfiles500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listProfiles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Profiles
 */

export function useListProfilesSuspense<
  TData = Awaited<ReturnType<typeof listProfiles>>,
  TError = ErrorType<ListProfiles401 | ListProfiles403 | ListProfiles409 | ListProfiles500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listProfiles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListProfilesSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCreateProfileUrl = () => {
  return `/api/v1/profiles`;
};

/**
 * @summary Create a Profile; a Content Type's first is its default
 */
export const createProfile = async (
  targetsNewProfile: TargetsNewProfile,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsProfile> => {
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
  return apiFetch<TargetsProfile>(getCreateProfileUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(targetsNewProfile),
  });
};

export const getCreateProfileMutationKey = () => ["createProfile"] as const;

export const getCreateProfileMutationOptions = <
  TError = ErrorType<
    | CreateProfile401
    | CreateProfile403
    | CreateProfile404
    | CreateProfile409
    | CreateProfile422
    | CreateProfile500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof createProfile>>,
    TError,
    CreateProfileMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof createProfile>>,
  TError,
  CreateProfileMutationVariables,
  TContext
> => {
  const mutationKey = getCreateProfileMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof createProfile>>,
    CreateProfileMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return createProfile(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateProfileMutationResult = NonNullable<Awaited<ReturnType<typeof createProfile>>>;
export type CreateProfileMutationBody = TargetsNewProfile;
export type CreateProfileMutationError = ErrorType<
  | CreateProfile401
  | CreateProfile403
  | CreateProfile404
  | CreateProfile409
  | CreateProfile422
  | CreateProfile500
>;
export type CreateProfileMutationVariables = { data: TargetsNewProfile };

/**
 * @summary Create a Profile; a Content Type's first is its default
 */
export const useCreateProfile = <
  TError = ErrorType<
    | CreateProfile401
    | CreateProfile403
    | CreateProfile404
    | CreateProfile409
    | CreateProfile422
    | CreateProfile500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createProfile>>,
      TError,
      CreateProfileMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof createProfile>>,
  TError,
  CreateProfileMutationVariables,
  TContext
> => {
  return useMutation(getCreateProfileMutationOptions(options), queryClient);
};
export const getListTargetsUrl = () => {
  return `/api/v1/targets`;
};

/**
 * @summary List the Targets
 */
export const listTargets = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsTargetList> => {
  return apiFetch<TargetsTargetList>(getListTargetsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListTargetsQueryKey = () => {
  return [`/api/v1/targets`] as const;
};

export const getListTargetsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listTargets>>,
  TError = ErrorType<ListTargets401 | ListTargets403 | ListTargets409 | ListTargets500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTargets>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListTargetsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listTargets>>> = ({ signal }) =>
    listTargets({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTargets>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListTargetsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listTargets>>>;
export type ListTargetsSuspenseQueryError = ErrorType<
  ListTargets401 | ListTargets403 | ListTargets409 | ListTargets500
>;

export function useListTargetsSuspense<
  TData = Awaited<ReturnType<typeof listTargets>>,
  TError = ErrorType<ListTargets401 | ListTargets403 | ListTargets409 | ListTargets500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTargets>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTargetsSuspense<
  TData = Awaited<ReturnType<typeof listTargets>>,
  TError = ErrorType<ListTargets401 | ListTargets403 | ListTargets409 | ListTargets500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTargets>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTargetsSuspense<
  TData = Awaited<ReturnType<typeof listTargets>>,
  TError = ErrorType<ListTargets401 | ListTargets403 | ListTargets409 | ListTargets500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTargets>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Targets
 */

export function useListTargetsSuspense<
  TData = Awaited<ReturnType<typeof listTargets>>,
  TError = ErrorType<ListTargets401 | ListTargets403 | ListTargets409 | ListTargets500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTargets>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListTargetsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCreateTargetUrl = () => {
  return `/api/v1/targets`;
};

/**
 * @summary Create a Target from its resolutions
 */
export const createTarget = async (
  targetsNewTarget: TargetsNewTarget,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsTarget> => {
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
  return apiFetch<TargetsTarget>(getCreateTargetUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(targetsNewTarget),
  });
};

export const getCreateTargetMutationKey = () => ["createTarget"] as const;

export const getCreateTargetMutationOptions = <
  TError = ErrorType<
    CreateTarget401 | CreateTarget403 | CreateTarget409 | CreateTarget422 | CreateTarget500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof createTarget>>,
    TError,
    CreateTargetMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof createTarget>>,
  TError,
  CreateTargetMutationVariables,
  TContext
> => {
  const mutationKey = getCreateTargetMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof createTarget>>,
    CreateTargetMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return createTarget(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateTargetMutationResult = NonNullable<Awaited<ReturnType<typeof createTarget>>>;
export type CreateTargetMutationBody = TargetsNewTarget;
export type CreateTargetMutationError = ErrorType<
  CreateTarget401 | CreateTarget403 | CreateTarget409 | CreateTarget422 | CreateTarget500
>;
export type CreateTargetMutationVariables = { data: TargetsNewTarget };

/**
 * @summary Create a Target from its resolutions
 */
export const useCreateTarget = <
  TError = ErrorType<
    CreateTarget401 | CreateTarget403 | CreateTarget409 | CreateTarget422 | CreateTarget500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createTarget>>,
      TError,
      CreateTargetMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof createTarget>>,
  TError,
  CreateTargetMutationVariables,
  TContext
> => {
  return useMutation(getCreateTargetMutationOptions(options), queryClient);
};
