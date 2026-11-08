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
  ClearTrackerGoal401,
  ClearTrackerGoal403,
  ClearTrackerGoal404,
  ClearTrackerGoal409,
  ClearTrackerGoal422,
  ClearTrackerGoal500,
  ClearTrackerRule401,
  ClearTrackerRule403,
  ClearTrackerRule404,
  ClearTrackerRule409,
  ClearTrackerRule422,
  ClearTrackerRule500,
  GetDefaultGoal401,
  GetDefaultGoal403,
  GetDefaultGoal409,
  GetDefaultGoal500,
  GetSeedingSettings401,
  GetSeedingSettings403,
  GetSeedingSettings409,
  GetSeedingSettings500,
  GetTracker401,
  GetTracker403,
  GetTracker404,
  GetTracker409,
  GetTracker422,
  GetTracker500,
  ListTrackers401,
  ListTrackers403,
  ListTrackers409,
  ListTrackers500,
  MergeTracker401,
  MergeTracker403,
  MergeTracker404,
  MergeTracker409,
  MergeTracker422,
  MergeTracker500,
  MergeTrackerInputBody,
  RemoveTracker401,
  RemoveTracker403,
  RemoveTracker404,
  RemoveTracker409,
  RemoveTracker422,
  RemoveTracker500,
  RenameTracker401,
  RenameTracker403,
  RenameTracker404,
  RenameTracker409,
  RenameTracker422,
  RenameTracker500,
  RenameTrackerInputBody,
  SetDefaultGoal401,
  SetDefaultGoal403,
  SetDefaultGoal409,
  SetDefaultGoal422,
  SetDefaultGoal500,
  SetTrackerGoal401,
  SetTrackerGoal403,
  SetTrackerGoal404,
  SetTrackerGoal409,
  SetTrackerGoal422,
  SetTrackerGoal500,
  SetTrackerRule401,
  SetTrackerRule403,
  SetTrackerRule404,
  SetTrackerRule409,
  SetTrackerRule422,
  SetTrackerRule500,
  TrackersGoal,
  TrackersRuleBody,
  TrackersSettings,
  TrackersSettingsChange,
  TrackersTracker,
  TrackersTrackerList,
  UpdateSeedingSettings401,
  UpdateSeedingSettings403,
  UpdateSeedingSettings409,
  UpdateSeedingSettings422,
  UpdateSeedingSettings500,
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

export const getListTrackersUrl = () => {
  return `/api/v1/trackers`;
};

/**
 * @summary List the Trackers and the Indexers that reach each
 */
export const listTrackers = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersTrackerList> => {
  return apiFetch<TrackersTrackerList>(getListTrackersUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListTrackersQueryKey = () => {
  return [`/api/v1/trackers`] as const;
};

export const getListTrackersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listTrackers>>,
  TError = ErrorType<ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListTrackersQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listTrackers>>> = ({ signal }) =>
    listTrackers({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListTrackersSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listTrackers>>>;
export type ListTrackersSuspenseQueryError = ErrorType<
  ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500
>;

export function useListTrackersSuspense<
  TData = Awaited<ReturnType<typeof listTrackers>>,
  TError = ErrorType<ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTrackersSuspense<
  TData = Awaited<ReturnType<typeof listTrackers>>,
  TError = ErrorType<ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTrackersSuspense<
  TData = Awaited<ReturnType<typeof listTrackers>>,
  TError = ErrorType<ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Trackers and the Indexers that reach each
 */

export function useListTrackersSuspense<
  TData = Awaited<ReturnType<typeof listTrackers>>,
  TError = ErrorType<ListTrackers401 | ListTrackers403 | ListTrackers409 | ListTrackers500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTrackers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListTrackersSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetDefaultGoalUrl = () => {
  return `/api/v1/trackers/default-goal`;
};

/**
 * @summary Get the seeding goal of every Tracker with none of its own
 */
export const getDefaultGoal = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersGoal> => {
  return apiFetch<TrackersGoal>(getGetDefaultGoalUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetDefaultGoalQueryKey = () => {
  return [`/api/v1/trackers/default-goal`] as const;
};

export const getGetDefaultGoalSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getDefaultGoal>>,
  TError = ErrorType<GetDefaultGoal401 | GetDefaultGoal403 | GetDefaultGoal409 | GetDefaultGoal500>,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDefaultGoal>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetDefaultGoalQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getDefaultGoal>>> = ({ signal }) =>
    getDefaultGoal({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDefaultGoal>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetDefaultGoalSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getDefaultGoal>>
>;
export type GetDefaultGoalSuspenseQueryError = ErrorType<
  GetDefaultGoal401 | GetDefaultGoal403 | GetDefaultGoal409 | GetDefaultGoal500
>;

export function useGetDefaultGoalSuspense<
  TData = Awaited<ReturnType<typeof getDefaultGoal>>,
  TError = ErrorType<GetDefaultGoal401 | GetDefaultGoal403 | GetDefaultGoal409 | GetDefaultGoal500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDefaultGoal>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetDefaultGoalSuspense<
  TData = Awaited<ReturnType<typeof getDefaultGoal>>,
  TError = ErrorType<GetDefaultGoal401 | GetDefaultGoal403 | GetDefaultGoal409 | GetDefaultGoal500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDefaultGoal>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetDefaultGoalSuspense<
  TData = Awaited<ReturnType<typeof getDefaultGoal>>,
  TError = ErrorType<GetDefaultGoal401 | GetDefaultGoal403 | GetDefaultGoal409 | GetDefaultGoal500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDefaultGoal>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the seeding goal of every Tracker with none of its own
 */

export function useGetDefaultGoalSuspense<
  TData = Awaited<ReturnType<typeof getDefaultGoal>>,
  TError = ErrorType<GetDefaultGoal401 | GetDefaultGoal403 | GetDefaultGoal409 | GetDefaultGoal500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getDefaultGoal>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetDefaultGoalSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSetDefaultGoalUrl = () => {
  return `/api/v1/trackers/default-goal`;
};

/**
 * @summary Set the seeding goal of every Tracker with none of its own
 */
export const setDefaultGoal = async (
  trackersGoal: TrackersGoal,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersGoal> => {
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
  return apiFetch<TrackersGoal>(getSetDefaultGoalUrl(), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(trackersGoal),
  });
};

export const getSetDefaultGoalMutationKey = () => ["setDefaultGoal"] as const;

export const getSetDefaultGoalMutationOptions = <
  TError = ErrorType<
    | SetDefaultGoal401
    | SetDefaultGoal403
    | SetDefaultGoal409
    | SetDefaultGoal422
    | SetDefaultGoal500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setDefaultGoal>>,
    TError,
    SetDefaultGoalMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setDefaultGoal>>,
  TError,
  SetDefaultGoalMutationVariables,
  TContext
> => {
  const mutationKey = getSetDefaultGoalMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setDefaultGoal>>,
    SetDefaultGoalMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return setDefaultGoal(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetDefaultGoalMutationResult = NonNullable<Awaited<ReturnType<typeof setDefaultGoal>>>;
export type SetDefaultGoalMutationBody = TrackersGoal;
export type SetDefaultGoalMutationError = ErrorType<
  SetDefaultGoal401 | SetDefaultGoal403 | SetDefaultGoal409 | SetDefaultGoal422 | SetDefaultGoal500
>;
export type SetDefaultGoalMutationVariables = { data: TrackersGoal };

/**
 * @summary Set the seeding goal of every Tracker with none of its own
 */
export const useSetDefaultGoal = <
  TError = ErrorType<
    | SetDefaultGoal401
    | SetDefaultGoal403
    | SetDefaultGoal409
    | SetDefaultGoal422
    | SetDefaultGoal500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setDefaultGoal>>,
      TError,
      SetDefaultGoalMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setDefaultGoal>>,
  TError,
  SetDefaultGoalMutationVariables,
  TContext
> => {
  return useMutation(getSetDefaultGoalMutationOptions(options), queryClient);
};
export const getGetSeedingSettingsUrl = () => {
  return `/api/v1/trackers/settings`;
};

/**
 * @summary Get the seeding settings
 */
export const getSeedingSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersSettings> => {
  return apiFetch<TrackersSettings>(getGetSeedingSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetSeedingSettingsQueryKey = () => {
  return [`/api/v1/trackers/settings`] as const;
};

export const getGetSeedingSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSeedingSettings>>,
  TError = ErrorType<
    GetSeedingSettings401 | GetSeedingSettings403 | GetSeedingSettings409 | GetSeedingSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSeedingSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSeedingSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSeedingSettings>>> = ({ signal }) =>
    getSeedingSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSeedingSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSeedingSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getSeedingSettings>>
>;
export type GetSeedingSettingsSuspenseQueryError = ErrorType<
  GetSeedingSettings401 | GetSeedingSettings403 | GetSeedingSettings409 | GetSeedingSettings500
>;

export function useGetSeedingSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSeedingSettings>>,
  TError = ErrorType<
    GetSeedingSettings401 | GetSeedingSettings403 | GetSeedingSettings409 | GetSeedingSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSeedingSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSeedingSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSeedingSettings>>,
  TError = ErrorType<
    GetSeedingSettings401 | GetSeedingSettings403 | GetSeedingSettings409 | GetSeedingSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSeedingSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSeedingSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSeedingSettings>>,
  TError = ErrorType<
    GetSeedingSettings401 | GetSeedingSettings403 | GetSeedingSettings409 | GetSeedingSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSeedingSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the seeding settings
 */

export function useGetSeedingSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSeedingSettings>>,
  TError = ErrorType<
    GetSeedingSettings401 | GetSeedingSettings403 | GetSeedingSettings409 | GetSeedingSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSeedingSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSeedingSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateSeedingSettingsUrl = () => {
  return `/api/v1/trackers/settings`;
};

/**
 * @summary Change seeding settings by key, all or none
 */
export const updateSeedingSettings = async (
  trackersSettingsChange: TrackersSettingsChange,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersSettings> => {
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
  return apiFetch<TrackersSettings>(getUpdateSeedingSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(trackersSettingsChange),
  });
};

export const getUpdateSeedingSettingsMutationKey = () => ["updateSeedingSettings"] as const;

export const getUpdateSeedingSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateSeedingSettings401
    | UpdateSeedingSettings403
    | UpdateSeedingSettings409
    | UpdateSeedingSettings422
    | UpdateSeedingSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateSeedingSettings>>,
    TError,
    UpdateSeedingSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateSeedingSettings>>,
  TError,
  UpdateSeedingSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateSeedingSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateSeedingSettings>>,
    UpdateSeedingSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateSeedingSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateSeedingSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateSeedingSettings>>
>;
export type UpdateSeedingSettingsMutationBody = TrackersSettingsChange;
export type UpdateSeedingSettingsMutationError = ErrorType<
  | UpdateSeedingSettings401
  | UpdateSeedingSettings403
  | UpdateSeedingSettings409
  | UpdateSeedingSettings422
  | UpdateSeedingSettings500
>;
export type UpdateSeedingSettingsMutationVariables = { data: TrackersSettingsChange };

/**
 * @summary Change seeding settings by key, all or none
 */
export const useUpdateSeedingSettings = <
  TError = ErrorType<
    | UpdateSeedingSettings401
    | UpdateSeedingSettings403
    | UpdateSeedingSettings409
    | UpdateSeedingSettings422
    | UpdateSeedingSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateSeedingSettings>>,
      TError,
      UpdateSeedingSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateSeedingSettings>>,
  TError,
  UpdateSeedingSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateSeedingSettingsMutationOptions(options), queryClient);
};
export const getRemoveTrackerUrl = (id: string) => {
  return `/api/v1/trackers/${id}`;
};

/**
 * @summary Remove a Tracker that no Indexer reaches and no live torrent or unfinished Grab names. Removing it also deletes its seeding rule, goal and seeding history. Its finished Grabs and removed torrents are kept with no Tracker.
 */
export const removeTracker = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getRemoveTrackerUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getRemoveTrackerMutationKey = () => ["removeTracker"] as const;

export const getRemoveTrackerMutationOptions = <
  TError = ErrorType<
    | RemoveTracker401
    | RemoveTracker403
    | RemoveTracker404
    | RemoveTracker409
    | RemoveTracker422
    | RemoveTracker500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof removeTracker>>,
    TError,
    RemoveTrackerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof removeTracker>>,
  TError,
  RemoveTrackerMutationVariables,
  TContext
> => {
  const mutationKey = getRemoveTrackerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof removeTracker>>,
    RemoveTrackerMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return removeTracker(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RemoveTrackerMutationResult = NonNullable<Awaited<ReturnType<typeof removeTracker>>>;

export type RemoveTrackerMutationError = ErrorType<
  | RemoveTracker401
  | RemoveTracker403
  | RemoveTracker404
  | RemoveTracker409
  | RemoveTracker422
  | RemoveTracker500
>;
export type RemoveTrackerMutationVariables = { id: string };

/**
 * @summary Remove a Tracker that no Indexer reaches and no live torrent or unfinished Grab names. Removing it also deletes its seeding rule, goal and seeding history. Its finished Grabs and removed torrents are kept with no Tracker.
 */
export const useRemoveTracker = <
  TError = ErrorType<
    | RemoveTracker401
    | RemoveTracker403
    | RemoveTracker404
    | RemoveTracker409
    | RemoveTracker422
    | RemoveTracker500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof removeTracker>>,
      TError,
      RemoveTrackerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof removeTracker>>,
  TError,
  RemoveTrackerMutationVariables,
  TContext
> => {
  return useMutation(getRemoveTrackerMutationOptions(options), queryClient);
};
export const getGetTrackerUrl = (id: string) => {
  return `/api/v1/trackers/${id}`;
};

/**
 * @summary Get a Tracker
 */
export const getTracker = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersTracker> => {
  return apiFetch<TrackersTracker>(getGetTrackerUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetTrackerQueryKey = (id: string) => {
  return [`/api/v1/trackers/${id}`] as const;
};

export const getGetTrackerSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getTracker>>,
  TError = ErrorType<
    GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetTrackerQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getTracker>>> = ({ signal }) =>
    getTracker(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetTrackerSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getTracker>>>;
export type GetTrackerSuspenseQueryError = ErrorType<
  GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
>;

export function useGetTrackerSuspense<
  TData = Awaited<ReturnType<typeof getTracker>>,
  TError = ErrorType<
    GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
  >,
>(
  id: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTrackerSuspense<
  TData = Awaited<ReturnType<typeof getTracker>>,
  TError = ErrorType<
    GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTrackerSuspense<
  TData = Awaited<ReturnType<typeof getTracker>>,
  TError = ErrorType<
    GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Tracker
 */

export function useGetTrackerSuspense<
  TData = Awaited<ReturnType<typeof getTracker>>,
  TError = ErrorType<
    GetTracker401 | GetTracker403 | GetTracker404 | GetTracker409 | GetTracker422 | GetTracker500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTracker>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetTrackerSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getRenameTrackerUrl = (id: string) => {
  return `/api/v1/trackers/${id}`;
};

/**
 * @summary Rename a Tracker. Every Indexer and Release that reached it still does.
 */
export const renameTracker = async (
  id: string,
  renameTrackerInputBody: RenameTrackerInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersTracker> => {
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
  return apiFetch<TrackersTracker>(getRenameTrackerUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(renameTrackerInputBody),
  });
};

export const getRenameTrackerMutationKey = () => ["renameTracker"] as const;

export const getRenameTrackerMutationOptions = <
  TError = ErrorType<
    | RenameTracker401
    | RenameTracker403
    | RenameTracker404
    | RenameTracker409
    | RenameTracker422
    | RenameTracker500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof renameTracker>>,
    TError,
    RenameTrackerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof renameTracker>>,
  TError,
  RenameTrackerMutationVariables,
  TContext
> => {
  const mutationKey = getRenameTrackerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof renameTracker>>,
    RenameTrackerMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return renameTracker(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RenameTrackerMutationResult = NonNullable<Awaited<ReturnType<typeof renameTracker>>>;
export type RenameTrackerMutationBody = RenameTrackerInputBody;
export type RenameTrackerMutationError = ErrorType<
  | RenameTracker401
  | RenameTracker403
  | RenameTracker404
  | RenameTracker409
  | RenameTracker422
  | RenameTracker500
>;
export type RenameTrackerMutationVariables = { id: string; data: RenameTrackerInputBody };

/**
 * @summary Rename a Tracker. Every Indexer and Release that reached it still does.
 */
export const useRenameTracker = <
  TError = ErrorType<
    | RenameTracker401
    | RenameTracker403
    | RenameTracker404
    | RenameTracker409
    | RenameTracker422
    | RenameTracker500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof renameTracker>>,
      TError,
      RenameTrackerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof renameTracker>>,
  TError,
  RenameTrackerMutationVariables,
  TContext
> => {
  return useMutation(getRenameTrackerMutationOptions(options), queryClient);
};
export const getClearTrackerGoalUrl = (id: string) => {
  return `/api/v1/trackers/${id}/goal`;
};

/**
 * @summary Delete a Tracker's own seeding goal, so it follows the default goal
 */
export const clearTrackerGoal = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersTracker> => {
  return apiFetch<TrackersTracker>(getClearTrackerGoalUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getClearTrackerGoalMutationKey = () => ["clearTrackerGoal"] as const;

export const getClearTrackerGoalMutationOptions = <
  TError = ErrorType<
    | ClearTrackerGoal401
    | ClearTrackerGoal403
    | ClearTrackerGoal404
    | ClearTrackerGoal409
    | ClearTrackerGoal422
    | ClearTrackerGoal500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof clearTrackerGoal>>,
    TError,
    ClearTrackerGoalMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof clearTrackerGoal>>,
  TError,
  ClearTrackerGoalMutationVariables,
  TContext
> => {
  const mutationKey = getClearTrackerGoalMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof clearTrackerGoal>>,
    ClearTrackerGoalMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return clearTrackerGoal(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ClearTrackerGoalMutationResult = NonNullable<
  Awaited<ReturnType<typeof clearTrackerGoal>>
>;

export type ClearTrackerGoalMutationError = ErrorType<
  | ClearTrackerGoal401
  | ClearTrackerGoal403
  | ClearTrackerGoal404
  | ClearTrackerGoal409
  | ClearTrackerGoal422
  | ClearTrackerGoal500
>;
export type ClearTrackerGoalMutationVariables = { id: string };

/**
 * @summary Delete a Tracker's own seeding goal, so it follows the default goal
 */
export const useClearTrackerGoal = <
  TError = ErrorType<
    | ClearTrackerGoal401
    | ClearTrackerGoal403
    | ClearTrackerGoal404
    | ClearTrackerGoal409
    | ClearTrackerGoal422
    | ClearTrackerGoal500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof clearTrackerGoal>>,
      TError,
      ClearTrackerGoalMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof clearTrackerGoal>>,
  TError,
  ClearTrackerGoalMutationVariables,
  TContext
> => {
  return useMutation(getClearTrackerGoalMutationOptions(options), queryClient);
};
export const getSetTrackerGoalUrl = (id: string) => {
  return `/api/v1/trackers/${id}/goal`;
};

/**
 * @summary Set a Tracker's own seeding goal. A goal equal to the default goal is deleted, so the Tracker follows the default.
 */
export const setTrackerGoal = async (
  id: string,
  trackersGoal: TrackersGoal,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersTracker> => {
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
  return apiFetch<TrackersTracker>(getSetTrackerGoalUrl(id), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(trackersGoal),
  });
};

export const getSetTrackerGoalMutationKey = () => ["setTrackerGoal"] as const;

export const getSetTrackerGoalMutationOptions = <
  TError = ErrorType<
    | SetTrackerGoal401
    | SetTrackerGoal403
    | SetTrackerGoal404
    | SetTrackerGoal409
    | SetTrackerGoal422
    | SetTrackerGoal500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setTrackerGoal>>,
    TError,
    SetTrackerGoalMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setTrackerGoal>>,
  TError,
  SetTrackerGoalMutationVariables,
  TContext
> => {
  const mutationKey = getSetTrackerGoalMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setTrackerGoal>>,
    SetTrackerGoalMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return setTrackerGoal(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetTrackerGoalMutationResult = NonNullable<Awaited<ReturnType<typeof setTrackerGoal>>>;
export type SetTrackerGoalMutationBody = TrackersGoal;
export type SetTrackerGoalMutationError = ErrorType<
  | SetTrackerGoal401
  | SetTrackerGoal403
  | SetTrackerGoal404
  | SetTrackerGoal409
  | SetTrackerGoal422
  | SetTrackerGoal500
>;
export type SetTrackerGoalMutationVariables = { id: string; data: TrackersGoal };

/**
 * @summary Set a Tracker's own seeding goal. A goal equal to the default goal is deleted, so the Tracker follows the default.
 */
export const useSetTrackerGoal = <
  TError = ErrorType<
    | SetTrackerGoal401
    | SetTrackerGoal403
    | SetTrackerGoal404
    | SetTrackerGoal409
    | SetTrackerGoal422
    | SetTrackerGoal500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setTrackerGoal>>,
      TError,
      SetTrackerGoalMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setTrackerGoal>>,
  TError,
  SetTrackerGoalMutationVariables,
  TContext
> => {
  return useMutation(getSetTrackerGoalMutationOptions(options), queryClient);
};
export const getMergeTrackerUrl = (id: string) => {
  return `/api/v1/trackers/${id}/merge`;
};

/**
 * @summary Merge a Tracker into another, which keeps its name and takes its Indexers, Grabs, Torrents, hosts and seeding history. The merged Tracker is deleted.
 */
export const mergeTracker = async (
  id: string,
  mergeTrackerInputBody: MergeTrackerInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersTracker> => {
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
  return apiFetch<TrackersTracker>(getMergeTrackerUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(mergeTrackerInputBody),
  });
};

export const getMergeTrackerMutationKey = () => ["mergeTracker"] as const;

export const getMergeTrackerMutationOptions = <
  TError = ErrorType<
    | MergeTracker401
    | MergeTracker403
    | MergeTracker404
    | MergeTracker409
    | MergeTracker422
    | MergeTracker500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof mergeTracker>>,
    TError,
    MergeTrackerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof mergeTracker>>,
  TError,
  MergeTrackerMutationVariables,
  TContext
> => {
  const mutationKey = getMergeTrackerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof mergeTracker>>,
    MergeTrackerMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return mergeTracker(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MergeTrackerMutationResult = NonNullable<Awaited<ReturnType<typeof mergeTracker>>>;
export type MergeTrackerMutationBody = MergeTrackerInputBody;
export type MergeTrackerMutationError = ErrorType<
  | MergeTracker401
  | MergeTracker403
  | MergeTracker404
  | MergeTracker409
  | MergeTracker422
  | MergeTracker500
>;
export type MergeTrackerMutationVariables = { id: string; data: MergeTrackerInputBody };

/**
 * @summary Merge a Tracker into another, which keeps its name and takes its Indexers, Grabs, Torrents, hosts and seeding history. The merged Tracker is deleted.
 */
export const useMergeTracker = <
  TError = ErrorType<
    | MergeTracker401
    | MergeTracker403
    | MergeTracker404
    | MergeTracker409
    | MergeTracker422
    | MergeTracker500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof mergeTracker>>,
      TError,
      MergeTrackerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof mergeTracker>>,
  TError,
  MergeTrackerMutationVariables,
  TContext
> => {
  return useMutation(getMergeTrackerMutationOptions(options), queryClient);
};
export const getClearTrackerRuleUrl = (id: string) => {
  return `/api/v1/trackers/${id}/rule`;
};

/**
 * @summary Delete the seeding rule an Admin set for a Tracker, so the latest rule its Releases state is in force
 */
export const clearTrackerRule = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersTracker> => {
  return apiFetch<TrackersTracker>(getClearTrackerRuleUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getClearTrackerRuleMutationKey = () => ["clearTrackerRule"] as const;

export const getClearTrackerRuleMutationOptions = <
  TError = ErrorType<
    | ClearTrackerRule401
    | ClearTrackerRule403
    | ClearTrackerRule404
    | ClearTrackerRule409
    | ClearTrackerRule422
    | ClearTrackerRule500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof clearTrackerRule>>,
    TError,
    ClearTrackerRuleMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof clearTrackerRule>>,
  TError,
  ClearTrackerRuleMutationVariables,
  TContext
> => {
  const mutationKey = getClearTrackerRuleMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof clearTrackerRule>>,
    ClearTrackerRuleMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return clearTrackerRule(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ClearTrackerRuleMutationResult = NonNullable<
  Awaited<ReturnType<typeof clearTrackerRule>>
>;

export type ClearTrackerRuleMutationError = ErrorType<
  | ClearTrackerRule401
  | ClearTrackerRule403
  | ClearTrackerRule404
  | ClearTrackerRule409
  | ClearTrackerRule422
  | ClearTrackerRule500
>;
export type ClearTrackerRuleMutationVariables = { id: string };

/**
 * @summary Delete the seeding rule an Admin set for a Tracker, so the latest rule its Releases state is in force
 */
export const useClearTrackerRule = <
  TError = ErrorType<
    | ClearTrackerRule401
    | ClearTrackerRule403
    | ClearTrackerRule404
    | ClearTrackerRule409
    | ClearTrackerRule422
    | ClearTrackerRule500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof clearTrackerRule>>,
      TError,
      ClearTrackerRuleMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof clearTrackerRule>>,
  TError,
  ClearTrackerRuleMutationVariables,
  TContext
> => {
  return useMutation(getClearTrackerRuleMutationOptions(options), queryClient);
};
export const getSetTrackerRuleUrl = (id: string) => {
  return `/api/v1/trackers/${id}/rule`;
};

/**
 * @summary Set a Tracker's seeding rule. Rules its Releases state never replace it.
 */
export const setTrackerRule = async (
  id: string,
  trackersRuleBody: TrackersRuleBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TrackersTracker> => {
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
  return apiFetch<TrackersTracker>(getSetTrackerRuleUrl(id), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(trackersRuleBody),
  });
};

export const getSetTrackerRuleMutationKey = () => ["setTrackerRule"] as const;

export const getSetTrackerRuleMutationOptions = <
  TError = ErrorType<
    | SetTrackerRule401
    | SetTrackerRule403
    | SetTrackerRule404
    | SetTrackerRule409
    | SetTrackerRule422
    | SetTrackerRule500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setTrackerRule>>,
    TError,
    SetTrackerRuleMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setTrackerRule>>,
  TError,
  SetTrackerRuleMutationVariables,
  TContext
> => {
  const mutationKey = getSetTrackerRuleMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setTrackerRule>>,
    SetTrackerRuleMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return setTrackerRule(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetTrackerRuleMutationResult = NonNullable<Awaited<ReturnType<typeof setTrackerRule>>>;
export type SetTrackerRuleMutationBody = TrackersRuleBody;
export type SetTrackerRuleMutationError = ErrorType<
  | SetTrackerRule401
  | SetTrackerRule403
  | SetTrackerRule404
  | SetTrackerRule409
  | SetTrackerRule422
  | SetTrackerRule500
>;
export type SetTrackerRuleMutationVariables = { id: string; data: TrackersRuleBody };

/**
 * @summary Set a Tracker's seeding rule. Rules its Releases state never replace it.
 */
export const useSetTrackerRule = <
  TError = ErrorType<
    | SetTrackerRule401
    | SetTrackerRule403
    | SetTrackerRule404
    | SetTrackerRule409
    | SetTrackerRule422
    | SetTrackerRule500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setTrackerRule>>,
      TError,
      SetTrackerRuleMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setTrackerRule>>,
  TError,
  SetTrackerRuleMutationVariables,
  TContext
> => {
  return useMutation(getSetTrackerRuleMutationOptions(options), queryClient);
};
