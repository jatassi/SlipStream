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
  CreateCut401,
  CreateCut403,
  CreateCut409,
  CreateCut422,
  CreateCut500,
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
  DeleteCut401,
  DeleteCut403,
  DeleteCut404,
  DeleteCut409,
  DeleteCut422,
  DeleteCut500,
  DeleteProfile401,
  DeleteProfile403,
  DeleteProfile404,
  DeleteProfile409,
  DeleteProfile422,
  DeleteProfile500,
  DeleteProfileParams,
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
  GetProfile401,
  GetProfile403,
  GetProfile404,
  GetProfile409,
  GetProfile422,
  GetProfile500,
  GetTarget401,
  GetTarget403,
  GetTarget404,
  GetTarget409,
  GetTarget422,
  GetTarget500,
  GetWorkWants401,
  GetWorkWants403,
  GetWorkWants404,
  GetWorkWants409,
  GetWorkWants422,
  GetWorkWants500,
  ListCuts401,
  ListCuts403,
  ListCuts409,
  ListCuts500,
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
  MakeDefaultProfile401,
  MakeDefaultProfile403,
  MakeDefaultProfile404,
  MakeDefaultProfile409,
  MakeDefaultProfile422,
  MakeDefaultProfile500,
  SetWorkWants401,
  SetWorkWants403,
  SetWorkWants404,
  SetWorkWants409,
  SetWorkWants422,
  SetWorkWants500,
  SetWorkWantsParams,
  TargetsCut,
  TargetsCutChange,
  TargetsCutList,
  TargetsNewCut,
  TargetsNewProfile,
  TargetsNewTarget,
  TargetsNewWants,
  TargetsProfile,
  TargetsProfileChange,
  TargetsProfileDetail,
  TargetsProfileList,
  TargetsQualityList,
  TargetsTarget,
  TargetsTargetChange,
  TargetsTargetList,
  TargetsWants,
  TargetsWantsChange,
  UpdateCut401,
  UpdateCut403,
  UpdateCut404,
  UpdateCut409,
  UpdateCut422,
  UpdateCut500,
  UpdateCutParams,
  UpdateProfile401,
  UpdateProfile403,
  UpdateProfile404,
  UpdateProfile409,
  UpdateProfile422,
  UpdateProfile500,
  UpdateProfileParams,
  UpdateTarget401,
  UpdateTarget403,
  UpdateTarget404,
  UpdateTarget409,
  UpdateTarget422,
  UpdateTarget500,
  UpdateTargetParams,
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

export const getListCutsUrl = () => {
  return `/api/v1/cuts`;
};

/**
 * @summary List the cuts an Edition list can name: built in, added by a Content Type, or custom
 */
export const listCuts = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsCutList> => {
  return apiFetch<TargetsCutList>(getListCutsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListCutsQueryKey = () => {
  return [`/api/v1/cuts`] as const;
};

export const getListCutsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listCuts>>,
  TError = ErrorType<ListCuts401 | ListCuts403 | ListCuts409 | ListCuts500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCuts>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListCutsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listCuts>>> = ({ signal }) =>
    listCuts({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCuts>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListCutsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listCuts>>>;
export type ListCutsSuspenseQueryError = ErrorType<
  ListCuts401 | ListCuts403 | ListCuts409 | ListCuts500
>;

export function useListCutsSuspense<
  TData = Awaited<ReturnType<typeof listCuts>>,
  TError = ErrorType<ListCuts401 | ListCuts403 | ListCuts409 | ListCuts500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCuts>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListCutsSuspense<
  TData = Awaited<ReturnType<typeof listCuts>>,
  TError = ErrorType<ListCuts401 | ListCuts403 | ListCuts409 | ListCuts500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCuts>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListCutsSuspense<
  TData = Awaited<ReturnType<typeof listCuts>>,
  TError = ErrorType<ListCuts401 | ListCuts403 | ListCuts409 | ListCuts500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCuts>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the cuts an Edition list can name: built in, added by a Content Type, or custom
 */

export function useListCutsSuspense<
  TData = Awaited<ReturnType<typeof listCuts>>,
  TError = ErrorType<ListCuts401 | ListCuts403 | ListCuts409 | ListCuts500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listCuts>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListCutsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCreateCutUrl = () => {
  return `/api/v1/cuts`;
};

/**
 * @summary Add a custom cut, keyed by its name's words, which every title parse reads from then on
 */
export const createCut = async (
  targetsNewCut: TargetsNewCut,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsCut> => {
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
  return apiFetch<TargetsCut>(getCreateCutUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(targetsNewCut),
  });
};

export const getCreateCutMutationKey = () => ["createCut"] as const;

export const getCreateCutMutationOptions = <
  TError = ErrorType<CreateCut401 | CreateCut403 | CreateCut409 | CreateCut422 | CreateCut500>,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof createCut>>,
    TError,
    CreateCutMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof createCut>>,
  TError,
  CreateCutMutationVariables,
  TContext
> => {
  const mutationKey = getCreateCutMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof createCut>>,
    CreateCutMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return createCut(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateCutMutationResult = NonNullable<Awaited<ReturnType<typeof createCut>>>;
export type CreateCutMutationBody = TargetsNewCut;
export type CreateCutMutationError = ErrorType<
  CreateCut401 | CreateCut403 | CreateCut409 | CreateCut422 | CreateCut500
>;
export type CreateCutMutationVariables = { data: TargetsNewCut };

/**
 * @summary Add a custom cut, keyed by its name's words, which every title parse reads from then on
 */
export const useCreateCut = <
  TError = ErrorType<CreateCut401 | CreateCut403 | CreateCut409 | CreateCut422 | CreateCut500>,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createCut>>,
      TError,
      CreateCutMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof createCut>>,
  TError,
  CreateCutMutationVariables,
  TContext
> => {
  return useMutation(getCreateCutMutationOptions(options), queryClient);
};
export const getDeleteCutUrl = (cutKey: string) => {
  return `/api/v1/cuts/${cutKey}`;
};

/**
 * @summary Remove a custom cut no Edition list names and no file is of
 */
export const deleteCut = async (
  cutKey: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeleteCutUrl(cutKey), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteCutMutationKey = () => ["deleteCut"] as const;

export const getDeleteCutMutationOptions = <
  TError = ErrorType<
    DeleteCut401 | DeleteCut403 | DeleteCut404 | DeleteCut409 | DeleteCut422 | DeleteCut500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteCut>>,
    TError,
    DeleteCutMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteCut>>,
  TError,
  DeleteCutMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteCutMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteCut>>,
    DeleteCutMutationVariables
  > = (props) => {
    const { cutKey } = props ?? {};

    return deleteCut(cutKey, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteCutMutationResult = NonNullable<Awaited<ReturnType<typeof deleteCut>>>;

export type DeleteCutMutationError = ErrorType<
  DeleteCut401 | DeleteCut403 | DeleteCut404 | DeleteCut409 | DeleteCut422 | DeleteCut500
>;
export type DeleteCutMutationVariables = { cutKey: string };

/**
 * @summary Remove a custom cut no Edition list names and no file is of
 */
export const useDeleteCut = <
  TError = ErrorType<
    DeleteCut401 | DeleteCut403 | DeleteCut404 | DeleteCut409 | DeleteCut422 | DeleteCut500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteCut>>,
      TError,
      DeleteCutMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteCut>>,
  TError,
  DeleteCutMutationVariables,
  TContext
> => {
  return useMutation(getDeleteCutMutationOptions(options), queryClient);
};
export const getUpdateCutUrl = (cutKey: string, params?: UpdateCutParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/cuts/${cutKey}?${stringifiedParams}`
    : `/api/v1/cuts/${cutKey}`;
};

/**
 * @summary Rename a custom cut or replace its aliases, renaming the files of it, or preview the change's Impact
 */
export const updateCut = async (
  cutKey: string,
  targetsNewCut: TargetsNewCut,
  params?: UpdateCutParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsCutChange> => {
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
  return apiFetch<TargetsCutChange>(getUpdateCutUrl(cutKey, params), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(targetsNewCut),
  });
};

export const getUpdateCutMutationKey = () => ["updateCut"] as const;

export const getUpdateCutMutationOptions = <
  TError = ErrorType<
    UpdateCut401 | UpdateCut403 | UpdateCut404 | UpdateCut409 | UpdateCut422 | UpdateCut500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateCut>>,
    TError,
    UpdateCutMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateCut>>,
  TError,
  UpdateCutMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateCutMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateCut>>,
    UpdateCutMutationVariables
  > = (props) => {
    const { cutKey, data, params } = props ?? {};

    return updateCut(cutKey, data, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateCutMutationResult = NonNullable<Awaited<ReturnType<typeof updateCut>>>;
export type UpdateCutMutationBody = TargetsNewCut;
export type UpdateCutMutationError = ErrorType<
  UpdateCut401 | UpdateCut403 | UpdateCut404 | UpdateCut409 | UpdateCut422 | UpdateCut500
>;
export type UpdateCutMutationVariables = {
  cutKey: string;
  data: TargetsNewCut;
  params?: UpdateCutParams;
};

/**
 * @summary Rename a custom cut or replace its aliases, renaming the files of it, or preview the change's Impact
 */
export const useUpdateCut = <
  TError = ErrorType<
    UpdateCut401 | UpdateCut403 | UpdateCut404 | UpdateCut409 | UpdateCut422 | UpdateCut500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateCut>>,
      TError,
      UpdateCutMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateCut>>,
  TError,
  UpdateCutMutationVariables,
  TContext
> => {
  return useMutation(getUpdateCutMutationOptions(options), queryClient);
};
export const getListProfilesUrl = () => {
  return `/api/v1/profiles`;
};

/**
 * @summary List the Profiles, each with the count of Works wanting from it
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
 * @summary List the Profiles, each with the count of Works wanting from it
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
 * @summary Create a Profile of Targets in Profile order; it is never the default
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
 * @summary Create a Profile of Targets in Profile order; it is never the default
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
export const getDeleteProfileUrl = (profileId: string, params: DeleteProfileParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/profiles/${profileId}?${stringifiedParams}`
    : `/api/v1/profiles/${profileId}`;
};

/**
 * @summary Delete a Profile that isn't its Content Type's default, moving its Works to another of the type, or preview the change's Impact
 */
export const deleteProfile = async (
  profileId: string,
  params: DeleteProfileParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsProfileChange> => {
  return apiFetch<TargetsProfileChange>(getDeleteProfileUrl(profileId, params), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteProfileMutationKey = () => ["deleteProfile"] as const;

export const getDeleteProfileMutationOptions = <
  TError = ErrorType<
    | DeleteProfile401
    | DeleteProfile403
    | DeleteProfile404
    | DeleteProfile409
    | DeleteProfile422
    | DeleteProfile500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteProfile>>,
    TError,
    DeleteProfileMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteProfile>>,
  TError,
  DeleteProfileMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteProfileMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteProfile>>,
    DeleteProfileMutationVariables
  > = (props) => {
    const { profileId, params } = props ?? {};

    return deleteProfile(profileId, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteProfileMutationResult = NonNullable<Awaited<ReturnType<typeof deleteProfile>>>;

export type DeleteProfileMutationError = ErrorType<
  | DeleteProfile401
  | DeleteProfile403
  | DeleteProfile404
  | DeleteProfile409
  | DeleteProfile422
  | DeleteProfile500
>;
export type DeleteProfileMutationVariables = { profileId: string; params: DeleteProfileParams };

/**
 * @summary Delete a Profile that isn't its Content Type's default, moving its Works to another of the type, or preview the change's Impact
 */
export const useDeleteProfile = <
  TError = ErrorType<
    | DeleteProfile401
    | DeleteProfile403
    | DeleteProfile404
    | DeleteProfile409
    | DeleteProfile422
    | DeleteProfile500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteProfile>>,
      TError,
      DeleteProfileMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteProfile>>,
  TError,
  DeleteProfileMutationVariables,
  TContext
> => {
  return useMutation(getDeleteProfileMutationOptions(options), queryClient);
};
export const getGetProfileUrl = (profileId: string) => {
  return `/api/v1/profiles/${profileId}`;
};

/**
 * @summary Get a Profile with the disk each of its Targets costs across its Works
 */
export const getProfile = async (
  profileId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsProfileDetail> => {
  return apiFetch<TargetsProfileDetail>(getGetProfileUrl(profileId), {
    ...options,
    method: "GET",
  });
};

export const getGetProfileQueryKey = (profileId: string) => {
  return [`/api/v1/profiles/${profileId}`] as const;
};

export const getGetProfileSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getProfile>>,
  TError = ErrorType<
    GetProfile401 | GetProfile403 | GetProfile404 | GetProfile409 | GetProfile422 | GetProfile500
  >,
>(
  profileId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getProfile>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetProfileQueryKey(profileId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getProfile>>> = ({ signal }) =>
    getProfile(profileId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getProfile>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetProfileSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getProfile>>>;
export type GetProfileSuspenseQueryError = ErrorType<
  GetProfile401 | GetProfile403 | GetProfile404 | GetProfile409 | GetProfile422 | GetProfile500
>;

export function useGetProfileSuspense<
  TData = Awaited<ReturnType<typeof getProfile>>,
  TError = ErrorType<
    GetProfile401 | GetProfile403 | GetProfile404 | GetProfile409 | GetProfile422 | GetProfile500
  >,
>(
  profileId: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getProfile>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetProfileSuspense<
  TData = Awaited<ReturnType<typeof getProfile>>,
  TError = ErrorType<
    GetProfile401 | GetProfile403 | GetProfile404 | GetProfile409 | GetProfile422 | GetProfile500
  >,
>(
  profileId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getProfile>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetProfileSuspense<
  TData = Awaited<ReturnType<typeof getProfile>>,
  TError = ErrorType<
    GetProfile401 | GetProfile403 | GetProfile404 | GetProfile409 | GetProfile422 | GetProfile500
  >,
>(
  profileId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getProfile>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Profile with the disk each of its Targets costs across its Works
 */

export function useGetProfileSuspense<
  TData = Awaited<ReturnType<typeof getProfile>>,
  TError = ErrorType<
    GetProfile401 | GetProfile403 | GetProfile404 | GetProfile409 | GetProfile422 | GetProfile500
  >,
>(
  profileId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getProfile>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetProfileSuspenseQueryOptions(profileId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateProfileUrl = (profileId: string, params?: UpdateProfileParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/profiles/${profileId}?${stringifiedParams}`
    : `/api/v1/profiles/${profileId}`;
};

/**
 * @summary Replace a Profile's name, Targets and Edition list, refilling and renaming its Works' files, or preview the change's Impact
 */
export const updateProfile = async (
  profileId: string,
  targetsNewProfile: TargetsNewProfile,
  params?: UpdateProfileParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsProfileChange> => {
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
  return apiFetch<TargetsProfileChange>(getUpdateProfileUrl(profileId, params), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(targetsNewProfile),
  });
};

export const getUpdateProfileMutationKey = () => ["updateProfile"] as const;

export const getUpdateProfileMutationOptions = <
  TError = ErrorType<
    | UpdateProfile401
    | UpdateProfile403
    | UpdateProfile404
    | UpdateProfile409
    | UpdateProfile422
    | UpdateProfile500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateProfile>>,
    TError,
    UpdateProfileMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateProfile>>,
  TError,
  UpdateProfileMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateProfileMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateProfile>>,
    UpdateProfileMutationVariables
  > = (props) => {
    const { profileId, data, params } = props ?? {};

    return updateProfile(profileId, data, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateProfileMutationResult = NonNullable<Awaited<ReturnType<typeof updateProfile>>>;
export type UpdateProfileMutationBody = TargetsNewProfile;
export type UpdateProfileMutationError = ErrorType<
  | UpdateProfile401
  | UpdateProfile403
  | UpdateProfile404
  | UpdateProfile409
  | UpdateProfile422
  | UpdateProfile500
>;
export type UpdateProfileMutationVariables = {
  profileId: string;
  data: TargetsNewProfile;
  params?: UpdateProfileParams;
};

/**
 * @summary Replace a Profile's name, Targets and Edition list, refilling and renaming its Works' files, or preview the change's Impact
 */
export const useUpdateProfile = <
  TError = ErrorType<
    | UpdateProfile401
    | UpdateProfile403
    | UpdateProfile404
    | UpdateProfile409
    | UpdateProfile422
    | UpdateProfile500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateProfile>>,
      TError,
      UpdateProfileMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateProfile>>,
  TError,
  UpdateProfileMutationVariables,
  TContext
> => {
  return useMutation(getUpdateProfileMutationOptions(options), queryClient);
};
export const getMakeDefaultProfileUrl = (profileId: string) => {
  return `/api/v1/profiles/${profileId}/default`;
};

/**
 * @summary Make a Profile its Content Type's default; existing Works keep theirs
 */
export const makeDefaultProfile = async (
  profileId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsProfile> => {
  return apiFetch<TargetsProfile>(getMakeDefaultProfileUrl(profileId), {
    ...options,
    method: "POST",
  });
};

export const getMakeDefaultProfileMutationKey = () => ["makeDefaultProfile"] as const;

export const getMakeDefaultProfileMutationOptions = <
  TError = ErrorType<
    | MakeDefaultProfile401
    | MakeDefaultProfile403
    | MakeDefaultProfile404
    | MakeDefaultProfile409
    | MakeDefaultProfile422
    | MakeDefaultProfile500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof makeDefaultProfile>>,
    TError,
    MakeDefaultProfileMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof makeDefaultProfile>>,
  TError,
  MakeDefaultProfileMutationVariables,
  TContext
> => {
  const mutationKey = getMakeDefaultProfileMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof makeDefaultProfile>>,
    MakeDefaultProfileMutationVariables
  > = (props) => {
    const { profileId } = props ?? {};

    return makeDefaultProfile(profileId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MakeDefaultProfileMutationResult = NonNullable<
  Awaited<ReturnType<typeof makeDefaultProfile>>
>;

export type MakeDefaultProfileMutationError = ErrorType<
  | MakeDefaultProfile401
  | MakeDefaultProfile403
  | MakeDefaultProfile404
  | MakeDefaultProfile409
  | MakeDefaultProfile422
  | MakeDefaultProfile500
>;
export type MakeDefaultProfileMutationVariables = { profileId: string };

/**
 * @summary Make a Profile its Content Type's default; existing Works keep theirs
 */
export const useMakeDefaultProfile = <
  TError = ErrorType<
    | MakeDefaultProfile401
    | MakeDefaultProfile403
    | MakeDefaultProfile404
    | MakeDefaultProfile409
    | MakeDefaultProfile422
    | MakeDefaultProfile500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof makeDefaultProfile>>,
      TError,
      MakeDefaultProfileMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof makeDefaultProfile>>,
  TError,
  MakeDefaultProfileMutationVariables,
  TContext
> => {
  return useMutation(getMakeDefaultProfileMutationOptions(options), queryClient);
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

export const getUpdateTargetUrl = (targetId: string, params?: UpdateTargetParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/targets/${targetId}?${stringifiedParams}`
    : `/api/v1/targets/${targetId}`;
};

/**
 * @summary Replace a Target's name and rules, renaming and refilling its Works' files, or preview the change's Impact
 */
export const updateTarget = async (
  targetId: string,
  targetsNewTarget: TargetsNewTarget,
  params?: UpdateTargetParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsTargetChange> => {
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
  return apiFetch<TargetsTargetChange>(getUpdateTargetUrl(targetId, params), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(targetsNewTarget),
  });
};

export const getUpdateTargetMutationKey = () => ["updateTarget"] as const;

export const getUpdateTargetMutationOptions = <
  TError = ErrorType<
    | UpdateTarget401
    | UpdateTarget403
    | UpdateTarget404
    | UpdateTarget409
    | UpdateTarget422
    | UpdateTarget500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateTarget>>,
    TError,
    UpdateTargetMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateTarget>>,
  TError,
  UpdateTargetMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateTargetMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateTarget>>,
    UpdateTargetMutationVariables
  > = (props) => {
    const { targetId, data, params } = props ?? {};

    return updateTarget(targetId, data, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateTargetMutationResult = NonNullable<Awaited<ReturnType<typeof updateTarget>>>;
export type UpdateTargetMutationBody = TargetsNewTarget;
export type UpdateTargetMutationError = ErrorType<
  | UpdateTarget401
  | UpdateTarget403
  | UpdateTarget404
  | UpdateTarget409
  | UpdateTarget422
  | UpdateTarget500
>;
export type UpdateTargetMutationVariables = {
  targetId: string;
  data: TargetsNewTarget;
  params?: UpdateTargetParams;
};

/**
 * @summary Replace a Target's name and rules, renaming and refilling its Works' files, or preview the change's Impact
 */
export const useUpdateTarget = <
  TError = ErrorType<
    | UpdateTarget401
    | UpdateTarget403
    | UpdateTarget404
    | UpdateTarget409
    | UpdateTarget422
    | UpdateTarget500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateTarget>>,
      TError,
      UpdateTargetMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateTarget>>,
  TError,
  UpdateTargetMutationVariables,
  TContext
> => {
  return useMutation(getUpdateTargetMutationOptions(options), queryClient);
};
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
export const getGetWorkWantsUrl = (workId: string) => {
  return `/api/v1/works/${workId}/wants`;
};

/**
 * @summary Get the Profile a Work wants from and its Edition list
 */
export const getWorkWants = async (
  workId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsWants> => {
  return apiFetch<TargetsWants>(getGetWorkWantsUrl(workId), {
    ...options,
    method: "GET",
  });
};

export const getGetWorkWantsQueryKey = (workId: string) => {
  return [`/api/v1/works/${workId}/wants`] as const;
};

export const getGetWorkWantsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getWorkWants>>,
  TError = ErrorType<
    | GetWorkWants401
    | GetWorkWants403
    | GetWorkWants404
    | GetWorkWants409
    | GetWorkWants422
    | GetWorkWants500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkWants>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetWorkWantsQueryKey(workId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getWorkWants>>> = ({ signal }) =>
    getWorkWants(workId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkWants>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetWorkWantsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getWorkWants>>>;
export type GetWorkWantsSuspenseQueryError = ErrorType<
  | GetWorkWants401
  | GetWorkWants403
  | GetWorkWants404
  | GetWorkWants409
  | GetWorkWants422
  | GetWorkWants500
>;

export function useGetWorkWantsSuspense<
  TData = Awaited<ReturnType<typeof getWorkWants>>,
  TError = ErrorType<
    | GetWorkWants401
    | GetWorkWants403
    | GetWorkWants404
    | GetWorkWants409
    | GetWorkWants422
    | GetWorkWants500
  >,
>(
  workId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkWants>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkWantsSuspense<
  TData = Awaited<ReturnType<typeof getWorkWants>>,
  TError = ErrorType<
    | GetWorkWants401
    | GetWorkWants403
    | GetWorkWants404
    | GetWorkWants409
    | GetWorkWants422
    | GetWorkWants500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkWants>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkWantsSuspense<
  TData = Awaited<ReturnType<typeof getWorkWants>>,
  TError = ErrorType<
    | GetWorkWants401
    | GetWorkWants403
    | GetWorkWants404
    | GetWorkWants409
    | GetWorkWants422
    | GetWorkWants500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkWants>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Profile a Work wants from and its Edition list
 */

export function useGetWorkWantsSuspense<
  TData = Awaited<ReturnType<typeof getWorkWants>>,
  TError = ErrorType<
    | GetWorkWants401
    | GetWorkWants403
    | GetWorkWants404
    | GetWorkWants409
    | GetWorkWants422
    | GetWorkWants500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkWants>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetWorkWantsSuspenseQueryOptions(workId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetWorkWantsUrl = (workId: string, params?: SetWorkWantsParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/works/${workId}/wants?${stringifiedParams}`
    : `/api/v1/works/${workId}/wants`;
};

/**
 * @summary Set the Profile a Work wants from and its own Edition list, refilling and renaming its files, or preview the change's Impact
 */
export const setWorkWants = async (
  workId: string,
  targetsNewWants: TargetsNewWants,
  params?: SetWorkWantsParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TargetsWantsChange> => {
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
  return apiFetch<TargetsWantsChange>(getSetWorkWantsUrl(workId, params), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(targetsNewWants),
  });
};

export const getSetWorkWantsMutationKey = () => ["setWorkWants"] as const;

export const getSetWorkWantsMutationOptions = <
  TError = ErrorType<
    | SetWorkWants401
    | SetWorkWants403
    | SetWorkWants404
    | SetWorkWants409
    | SetWorkWants422
    | SetWorkWants500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setWorkWants>>,
    TError,
    SetWorkWantsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setWorkWants>>,
  TError,
  SetWorkWantsMutationVariables,
  TContext
> => {
  const mutationKey = getSetWorkWantsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setWorkWants>>,
    SetWorkWantsMutationVariables
  > = (props) => {
    const { workId, data, params } = props ?? {};

    return setWorkWants(workId, data, params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetWorkWantsMutationResult = NonNullable<Awaited<ReturnType<typeof setWorkWants>>>;
export type SetWorkWantsMutationBody = TargetsNewWants;
export type SetWorkWantsMutationError = ErrorType<
  | SetWorkWants401
  | SetWorkWants403
  | SetWorkWants404
  | SetWorkWants409
  | SetWorkWants422
  | SetWorkWants500
>;
export type SetWorkWantsMutationVariables = {
  workId: string;
  data: TargetsNewWants;
  params?: SetWorkWantsParams;
};

/**
 * @summary Set the Profile a Work wants from and its own Edition list, refilling and renaming its files, or preview the change's Impact
 */
export const useSetWorkWants = <
  TError = ErrorType<
    | SetWorkWants401
    | SetWorkWants403
    | SetWorkWants404
    | SetWorkWants409
    | SetWorkWants422
    | SetWorkWants500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setWorkWants>>,
      TError,
      SetWorkWantsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setWorkWants>>,
  TError,
  SetWorkWantsMutationVariables,
  TContext
> => {
  return useMutation(getSetWorkWantsMutationOptions(options), queryClient);
};
