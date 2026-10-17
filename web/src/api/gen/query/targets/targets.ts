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
  DeleteTarget401,
  DeleteTarget403,
  DeleteTarget404,
  DeleteTarget409,
  DeleteTarget422,
  DeleteTarget500,
  DuplicateTarget401,
  DuplicateTarget403,
  DuplicateTarget404,
  DuplicateTarget409,
  DuplicateTarget422,
  DuplicateTarget500,
  GetTarget401,
  GetTarget403,
  GetTarget404,
  GetTarget409,
  GetTarget422,
  GetTarget500,
  ListProfiles401,
  ListProfiles403,
  ListProfiles409,
  ListProfiles500,
  ListQualities401,
  ListQualities403,
  ListQualities409,
  ListQualities500,
  ListTargets401,
  ListTargets403,
  ListTargets409,
  ListTargets500,
  TargetsNewProfile,
  TargetsNewTarget,
  TargetsProfile,
  TargetsProfileList,
  TargetsQualityList,
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
export const getListQualitiesUrl = () => {
  return `/api/v1/qualities`;
};

/**
 * @summary List the qualities a Target can accept, and the presets that order them
 */
export const listQualities = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsQualityList> => {
  return apiFetch<TargetsQualityList>(getListQualitiesUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListQualitiesQueryKey = () => {
  return [`/api/v1/qualities`] as const;
};

export const getListQualitiesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listQualities>>,
  TError = ErrorType<ListQualities401 | ListQualities403 | ListQualities409 | ListQualities500>,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listQualities>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListQualitiesQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listQualities>>> = ({ signal }) =>
    listQualities({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listQualities>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListQualitiesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listQualities>>
>;
export type ListQualitiesSuspenseQueryError = ErrorType<
  ListQualities401 | ListQualities403 | ListQualities409 | ListQualities500
>;

export function useListQualitiesSuspense<
  TData = Awaited<ReturnType<typeof listQualities>>,
  TError = ErrorType<ListQualities401 | ListQualities403 | ListQualities409 | ListQualities500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listQualities>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListQualitiesSuspense<
  TData = Awaited<ReturnType<typeof listQualities>>,
  TError = ErrorType<ListQualities401 | ListQualities403 | ListQualities409 | ListQualities500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listQualities>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListQualitiesSuspense<
  TData = Awaited<ReturnType<typeof listQualities>>,
  TError = ErrorType<ListQualities401 | ListQualities403 | ListQualities409 | ListQualities500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listQualities>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the qualities a Target can accept, and the presets that order them
 */

export function useListQualitiesSuspense<
  TData = Awaited<ReturnType<typeof listQualities>>,
  TError = ErrorType<ListQualities401 | ListQualities403 | ListQualities409 | ListQualities500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listQualities>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListQualitiesSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

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
 * @summary Create a Target from its tiers of qualities, cutoff, filters and sizes
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
 * @summary Create a Target from its tiers of qualities, cutoff, filters and sizes
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
export const getDeleteTargetUrl = (targetId: string) => {
  return `/api/v1/targets/${targetId}`;
};

/**
 * @summary Delete a Target no Profile holds
 */
export const deleteTarget = async (
  targetId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeleteTargetUrl(targetId), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteTargetMutationKey = () => ["deleteTarget"] as const;

export const getDeleteTargetMutationOptions = <
  TError = ErrorType<
    | DeleteTarget401
    | DeleteTarget403
    | DeleteTarget404
    | DeleteTarget409
    | DeleteTarget422
    | DeleteTarget500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteTarget>>,
    TError,
    DeleteTargetMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteTarget>>,
  TError,
  DeleteTargetMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteTargetMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteTarget>>,
    DeleteTargetMutationVariables
  > = (props) => {
    const { targetId } = props ?? {};

    return deleteTarget(targetId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteTargetMutationResult = NonNullable<Awaited<ReturnType<typeof deleteTarget>>>;

export type DeleteTargetMutationError = ErrorType<
  | DeleteTarget401
  | DeleteTarget403
  | DeleteTarget404
  | DeleteTarget409
  | DeleteTarget422
  | DeleteTarget500
>;
export type DeleteTargetMutationVariables = { targetId: string };

/**
 * @summary Delete a Target no Profile holds
 */
export const useDeleteTarget = <
  TError = ErrorType<
    | DeleteTarget401
    | DeleteTarget403
    | DeleteTarget404
    | DeleteTarget409
    | DeleteTarget422
    | DeleteTarget500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteTarget>>,
      TError,
      DeleteTargetMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteTarget>>,
  TError,
  DeleteTargetMutationVariables,
  TContext
> => {
  return useMutation(getDeleteTargetMutationOptions(options), queryClient);
};
export const getGetTargetUrl = (targetId: string) => {
  return `/api/v1/targets/${targetId}`;
};

/**
 * @summary Get a Target
 */
export const getTarget = async (
  targetId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsTarget> => {
  return apiFetch<TargetsTarget>(getGetTargetUrl(targetId), {
    ...options,
    method: "GET",
  });
};

export const getGetTargetQueryKey = (targetId: string) => {
  return [`/api/v1/targets/${targetId}`] as const;
};

export const getGetTargetSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getTarget>>,
  TError = ErrorType<
    GetTarget401 | GetTarget403 | GetTarget404 | GetTarget409 | GetTarget422 | GetTarget500
  >,
>(
  targetId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTarget>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetTargetQueryKey(targetId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getTarget>>> = ({ signal }) =>
    getTarget(targetId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTarget>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetTargetSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getTarget>>>;
export type GetTargetSuspenseQueryError = ErrorType<
  GetTarget401 | GetTarget403 | GetTarget404 | GetTarget409 | GetTarget422 | GetTarget500
>;

export function useGetTargetSuspense<
  TData = Awaited<ReturnType<typeof getTarget>>,
  TError = ErrorType<
    GetTarget401 | GetTarget403 | GetTarget404 | GetTarget409 | GetTarget422 | GetTarget500
  >,
>(
  targetId: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTarget>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTargetSuspense<
  TData = Awaited<ReturnType<typeof getTarget>>,
  TError = ErrorType<
    GetTarget401 | GetTarget403 | GetTarget404 | GetTarget409 | GetTarget422 | GetTarget500
  >,
>(
  targetId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTarget>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTargetSuspense<
  TData = Awaited<ReturnType<typeof getTarget>>,
  TError = ErrorType<
    GetTarget401 | GetTarget403 | GetTarget404 | GetTarget409 | GetTarget422 | GetTarget500
  >,
>(
  targetId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTarget>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Target
 */

export function useGetTargetSuspense<
  TData = Awaited<ReturnType<typeof getTarget>>,
  TError = ErrorType<
    GetTarget401 | GetTarget403 | GetTarget404 | GetTarget409 | GetTarget422 | GetTarget500
  >,
>(
  targetId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTarget>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetTargetSuspenseQueryOptions(targetId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getDuplicateTargetUrl = (targetId: string) => {
  return `/api/v1/targets/${targetId}/duplicate`;
};

/**
 * @summary Copy a Target's rules under the first free name ending in copy
 */
export const duplicateTarget = async (
  targetId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsTarget> => {
  return apiFetch<TargetsTarget>(getDuplicateTargetUrl(targetId), {
    ...options,
    method: "POST",
  });
};

export const getDuplicateTargetMutationKey = () => ["duplicateTarget"] as const;

export const getDuplicateTargetMutationOptions = <
  TError = ErrorType<
    | DuplicateTarget401
    | DuplicateTarget403
    | DuplicateTarget404
    | DuplicateTarget409
    | DuplicateTarget422
    | DuplicateTarget500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof duplicateTarget>>,
    TError,
    DuplicateTargetMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof duplicateTarget>>,
  TError,
  DuplicateTargetMutationVariables,
  TContext
> => {
  const mutationKey = getDuplicateTargetMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof duplicateTarget>>,
    DuplicateTargetMutationVariables
  > = (props) => {
    const { targetId } = props ?? {};

    return duplicateTarget(targetId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DuplicateTargetMutationResult = NonNullable<
  Awaited<ReturnType<typeof duplicateTarget>>
>;

export type DuplicateTargetMutationError = ErrorType<
  | DuplicateTarget401
  | DuplicateTarget403
  | DuplicateTarget404
  | DuplicateTarget409
  | DuplicateTarget422
  | DuplicateTarget500
>;
export type DuplicateTargetMutationVariables = { targetId: string };

/**
 * @summary Copy a Target's rules under the first free name ending in copy
 */
export const useDuplicateTarget = <
  TError = ErrorType<
    | DuplicateTarget401
    | DuplicateTarget403
    | DuplicateTarget404
    | DuplicateTarget409
    | DuplicateTarget422
    | DuplicateTarget500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof duplicateTarget>>,
      TError,
      DuplicateTargetMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof duplicateTarget>>,
  TError,
  DuplicateTargetMutationVariables,
  TContext
> => {
  return useMutation(getDuplicateTargetMutationOptions(options), queryClient);
};
