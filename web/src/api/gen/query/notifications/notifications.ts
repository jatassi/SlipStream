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
  AddNotificationConnection401,
  AddNotificationConnection403,
  AddNotificationConnection409,
  AddNotificationConnection422,
  AddNotificationConnection500,
  BellInputBody,
  ConnectionsOutputBody,
  DeleteNotificationConnection401,
  DeleteNotificationConnection403,
  DeleteNotificationConnection404,
  DeleteNotificationConnection409,
  DeleteNotificationConnection422,
  DeleteNotificationConnection500,
  EventsOutputBody,
  GetNotificationBell401,
  GetNotificationBell403,
  GetNotificationBell409,
  GetNotificationBell500,
  GetNotificationConnection401,
  GetNotificationConnection403,
  GetNotificationConnection404,
  GetNotificationConnection409,
  GetNotificationConnection422,
  GetNotificationConnection500,
  GetNotificationSettings401,
  GetNotificationSettings403,
  GetNotificationSettings409,
  GetNotificationSettings500,
  ListNotificationConnections401,
  ListNotificationConnections403,
  ListNotificationConnections409,
  ListNotificationConnections500,
  ListNotificationEvents401,
  ListNotificationEvents403,
  ListNotificationEvents409,
  ListNotificationEvents500,
  ListNotificationProviders401,
  ListNotificationProviders403,
  ListNotificationProviders409,
  ListNotificationProviders500,
  ListNotifications401,
  ListNotifications403,
  ListNotifications409,
  ListNotifications422,
  ListNotifications500,
  ListNotificationsParams,
  MarkNotificationsSeen401,
  MarkNotificationsSeen403,
  MarkNotificationsSeen409,
  MarkNotificationsSeen422,
  MarkNotificationsSeen500,
  NotificationsBell,
  NotificationsConnection,
  NotificationsDraftBody,
  NotificationsEditBody,
  NotificationsFeed,
  NotificationsTestPassed,
  ProvidersOutputBody,
  SeenInputBody,
  SettingsInputBody,
  SettingsOutputBody,
  TestNotificationConnection401,
  TestNotificationConnection403,
  TestNotificationConnection404,
  TestNotificationConnection409,
  TestNotificationConnection422,
  TestNotificationConnection500,
  TestNotificationDraft401,
  TestNotificationDraft403,
  TestNotificationDraft409,
  TestNotificationDraft422,
  TestNotificationDraft500,
  UpdateNotificationBell401,
  UpdateNotificationBell403,
  UpdateNotificationBell409,
  UpdateNotificationBell422,
  UpdateNotificationBell500,
  UpdateNotificationConnection401,
  UpdateNotificationConnection403,
  UpdateNotificationConnection404,
  UpdateNotificationConnection409,
  UpdateNotificationConnection422,
  UpdateNotificationConnection500,
  UpdateNotificationSettings401,
  UpdateNotificationSettings403,
  UpdateNotificationSettings409,
  UpdateNotificationSettings422,
  UpdateNotificationSettings500,
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

export const getGetNotificationBellUrl = () => {
  return `/api/v1/notifications/bell`;
};

/**
 * @summary Get the events the bell shows
 */
export const getNotificationBell = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<NotificationsBell> => {
  return apiFetch<NotificationsBell>(getGetNotificationBellUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetNotificationBellQueryKey = () => {
  return [`/api/v1/notifications/bell`] as const;
};

export const getGetNotificationBellSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getNotificationBell>>,
  TError = ErrorType<
    | GetNotificationBell401
    | GetNotificationBell403
    | GetNotificationBell409
    | GetNotificationBell500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationBell>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetNotificationBellQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getNotificationBell>>> = ({ signal }) =>
    getNotificationBell({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationBell>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetNotificationBellSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getNotificationBell>>
>;
export type GetNotificationBellSuspenseQueryError = ErrorType<
  GetNotificationBell401 | GetNotificationBell403 | GetNotificationBell409 | GetNotificationBell500
>;

export function useGetNotificationBellSuspense<
  TData = Awaited<ReturnType<typeof getNotificationBell>>,
  TError = ErrorType<
    | GetNotificationBell401
    | GetNotificationBell403
    | GetNotificationBell409
    | GetNotificationBell500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationBell>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNotificationBellSuspense<
  TData = Awaited<ReturnType<typeof getNotificationBell>>,
  TError = ErrorType<
    | GetNotificationBell401
    | GetNotificationBell403
    | GetNotificationBell409
    | GetNotificationBell500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationBell>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNotificationBellSuspense<
  TData = Awaited<ReturnType<typeof getNotificationBell>>,
  TError = ErrorType<
    | GetNotificationBell401
    | GetNotificationBell403
    | GetNotificationBell409
    | GetNotificationBell500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationBell>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the events the bell shows
 */

export function useGetNotificationBellSuspense<
  TData = Awaited<ReturnType<typeof getNotificationBell>>,
  TError = ErrorType<
    | GetNotificationBell401
    | GetNotificationBell403
    | GetNotificationBell409
    | GetNotificationBell500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationBell>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetNotificationBellSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateNotificationBellUrl = () => {
  return `/api/v1/notifications/bell`;
};

/**
 * @summary Change the events the bell shows from now on
 */
export const updateNotificationBell = async (
  bellInputBody: BellInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NotificationsBell> => {
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
  return apiFetch<NotificationsBell>(getUpdateNotificationBellUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(bellInputBody),
  });
};

export const getUpdateNotificationBellMutationKey = () => ["updateNotificationBell"] as const;

export const getUpdateNotificationBellMutationOptions = <
  TError = ErrorType<
    | UpdateNotificationBell401
    | UpdateNotificationBell403
    | UpdateNotificationBell409
    | UpdateNotificationBell422
    | UpdateNotificationBell500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateNotificationBell>>,
    TError,
    UpdateNotificationBellMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateNotificationBell>>,
  TError,
  UpdateNotificationBellMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateNotificationBellMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateNotificationBell>>,
    UpdateNotificationBellMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateNotificationBell(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateNotificationBellMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateNotificationBell>>
>;
export type UpdateNotificationBellMutationBody = BellInputBody;
export type UpdateNotificationBellMutationError = ErrorType<
  | UpdateNotificationBell401
  | UpdateNotificationBell403
  | UpdateNotificationBell409
  | UpdateNotificationBell422
  | UpdateNotificationBell500
>;
export type UpdateNotificationBellMutationVariables = { data: BellInputBody };

/**
 * @summary Change the events the bell shows from now on
 */
export const useUpdateNotificationBell = <
  TError = ErrorType<
    | UpdateNotificationBell401
    | UpdateNotificationBell403
    | UpdateNotificationBell409
    | UpdateNotificationBell422
    | UpdateNotificationBell500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateNotificationBell>>,
      TError,
      UpdateNotificationBellMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateNotificationBell>>,
  TError,
  UpdateNotificationBellMutationVariables,
  TContext
> => {
  return useMutation(getUpdateNotificationBellMutationOptions(options), queryClient);
};
export const getListNotificationConnectionsUrl = () => {
  return `/api/v1/notifications/connections`;
};

/**
 * @summary List the notification connections, each with whether it is sending
 */
export const listNotificationConnections = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<ConnectionsOutputBody> => {
  return apiFetch<ConnectionsOutputBody>(getListNotificationConnectionsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListNotificationConnectionsQueryKey = () => {
  return [`/api/v1/notifications/connections`] as const;
};

export const getListNotificationConnectionsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listNotificationConnections>>,
  TError = ErrorType<
    | ListNotificationConnections401
    | ListNotificationConnections403
    | ListNotificationConnections409
    | ListNotificationConnections500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationConnections>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListNotificationConnectionsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listNotificationConnections>>> = ({
    signal,
  }) => listNotificationConnections({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listNotificationConnections>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListNotificationConnectionsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listNotificationConnections>>
>;
export type ListNotificationConnectionsSuspenseQueryError = ErrorType<
  | ListNotificationConnections401
  | ListNotificationConnections403
  | ListNotificationConnections409
  | ListNotificationConnections500
>;

export function useListNotificationConnectionsSuspense<
  TData = Awaited<ReturnType<typeof listNotificationConnections>>,
  TError = ErrorType<
    | ListNotificationConnections401
    | ListNotificationConnections403
    | ListNotificationConnections409
    | ListNotificationConnections500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<
        Awaited<ReturnType<typeof listNotificationConnections>>,
        TError,
        TData
      >
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNotificationConnectionsSuspense<
  TData = Awaited<ReturnType<typeof listNotificationConnections>>,
  TError = ErrorType<
    | ListNotificationConnections401
    | ListNotificationConnections403
    | ListNotificationConnections409
    | ListNotificationConnections500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<
        Awaited<ReturnType<typeof listNotificationConnections>>,
        TError,
        TData
      >
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNotificationConnectionsSuspense<
  TData = Awaited<ReturnType<typeof listNotificationConnections>>,
  TError = ErrorType<
    | ListNotificationConnections401
    | ListNotificationConnections403
    | ListNotificationConnections409
    | ListNotificationConnections500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<
        Awaited<ReturnType<typeof listNotificationConnections>>,
        TError,
        TData
      >
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the notification connections, each with whether it is sending
 */

export function useListNotificationConnectionsSuspense<
  TData = Awaited<ReturnType<typeof listNotificationConnections>>,
  TError = ErrorType<
    | ListNotificationConnections401
    | ListNotificationConnections403
    | ListNotificationConnections409
    | ListNotificationConnections500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<
        Awaited<ReturnType<typeof listNotificationConnections>>,
        TError,
        TData
      >
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListNotificationConnectionsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getAddNotificationConnectionUrl = () => {
  return `/api/v1/notifications/connections`;
};

/**
 * @summary Add a notification connection
 */
export const addNotificationConnection = async (
  notificationsDraftBody: NotificationsDraftBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NotificationsConnection> => {
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
  return apiFetch<NotificationsConnection>(getAddNotificationConnectionUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(notificationsDraftBody),
  });
};

export const getAddNotificationConnectionMutationKey = () => ["addNotificationConnection"] as const;

export const getAddNotificationConnectionMutationOptions = <
  TError = ErrorType<
    | AddNotificationConnection401
    | AddNotificationConnection403
    | AddNotificationConnection409
    | AddNotificationConnection422
    | AddNotificationConnection500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof addNotificationConnection>>,
    TError,
    AddNotificationConnectionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof addNotificationConnection>>,
  TError,
  AddNotificationConnectionMutationVariables,
  TContext
> => {
  const mutationKey = getAddNotificationConnectionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof addNotificationConnection>>,
    AddNotificationConnectionMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return addNotificationConnection(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type AddNotificationConnectionMutationResult = NonNullable<
  Awaited<ReturnType<typeof addNotificationConnection>>
>;
export type AddNotificationConnectionMutationBody = NotificationsDraftBody;
export type AddNotificationConnectionMutationError = ErrorType<
  | AddNotificationConnection401
  | AddNotificationConnection403
  | AddNotificationConnection409
  | AddNotificationConnection422
  | AddNotificationConnection500
>;
export type AddNotificationConnectionMutationVariables = { data: NotificationsDraftBody };

/**
 * @summary Add a notification connection
 */
export const useAddNotificationConnection = <
  TError = ErrorType<
    | AddNotificationConnection401
    | AddNotificationConnection403
    | AddNotificationConnection409
    | AddNotificationConnection422
    | AddNotificationConnection500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof addNotificationConnection>>,
      TError,
      AddNotificationConnectionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof addNotificationConnection>>,
  TError,
  AddNotificationConnectionMutationVariables,
  TContext
> => {
  return useMutation(getAddNotificationConnectionMutationOptions(options), queryClient);
};
export const getTestNotificationDraftUrl = () => {
  return `/api/v1/notifications/connections/test`;
};

/**
 * @summary Send a Test message to a connection that isn't saved yet
 */
export const testNotificationDraft = async (
  notificationsDraftBody: NotificationsDraftBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NotificationsTestPassed> => {
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
  return apiFetch<NotificationsTestPassed>(getTestNotificationDraftUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(notificationsDraftBody),
  });
};

export const getTestNotificationDraftMutationKey = () => ["testNotificationDraft"] as const;

export const getTestNotificationDraftMutationOptions = <
  TError = ErrorType<
    | TestNotificationDraft401
    | TestNotificationDraft403
    | TestNotificationDraft409
    | TestNotificationDraft422
    | TestNotificationDraft500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof testNotificationDraft>>,
    TError,
    TestNotificationDraftMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof testNotificationDraft>>,
  TError,
  TestNotificationDraftMutationVariables,
  TContext
> => {
  const mutationKey = getTestNotificationDraftMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof testNotificationDraft>>,
    TestNotificationDraftMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return testNotificationDraft(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type TestNotificationDraftMutationResult = NonNullable<
  Awaited<ReturnType<typeof testNotificationDraft>>
>;
export type TestNotificationDraftMutationBody = NotificationsDraftBody;
export type TestNotificationDraftMutationError = ErrorType<
  | TestNotificationDraft401
  | TestNotificationDraft403
  | TestNotificationDraft409
  | TestNotificationDraft422
  | TestNotificationDraft500
>;
export type TestNotificationDraftMutationVariables = { data: NotificationsDraftBody };

/**
 * @summary Send a Test message to a connection that isn't saved yet
 */
export const useTestNotificationDraft = <
  TError = ErrorType<
    | TestNotificationDraft401
    | TestNotificationDraft403
    | TestNotificationDraft409
    | TestNotificationDraft422
    | TestNotificationDraft500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof testNotificationDraft>>,
      TError,
      TestNotificationDraftMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof testNotificationDraft>>,
  TError,
  TestNotificationDraftMutationVariables,
  TContext
> => {
  return useMutation(getTestNotificationDraftMutationOptions(options), queryClient);
};
export const getDeleteNotificationConnectionUrl = (id: string) => {
  return `/api/v1/notifications/connections/${id}`;
};

/**
 * @summary Delete a notification connection, whose waiting messages aren't sent
 */
export const deleteNotificationConnection = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeleteNotificationConnectionUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteNotificationConnectionMutationKey = () =>
  ["deleteNotificationConnection"] as const;

export const getDeleteNotificationConnectionMutationOptions = <
  TError = ErrorType<
    | DeleteNotificationConnection401
    | DeleteNotificationConnection403
    | DeleteNotificationConnection404
    | DeleteNotificationConnection409
    | DeleteNotificationConnection422
    | DeleteNotificationConnection500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteNotificationConnection>>,
    TError,
    DeleteNotificationConnectionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteNotificationConnection>>,
  TError,
  DeleteNotificationConnectionMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteNotificationConnectionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteNotificationConnection>>,
    DeleteNotificationConnectionMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return deleteNotificationConnection(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteNotificationConnectionMutationResult = NonNullable<
  Awaited<ReturnType<typeof deleteNotificationConnection>>
>;

export type DeleteNotificationConnectionMutationError = ErrorType<
  | DeleteNotificationConnection401
  | DeleteNotificationConnection403
  | DeleteNotificationConnection404
  | DeleteNotificationConnection409
  | DeleteNotificationConnection422
  | DeleteNotificationConnection500
>;
export type DeleteNotificationConnectionMutationVariables = { id: string };

/**
 * @summary Delete a notification connection, whose waiting messages aren't sent
 */
export const useDeleteNotificationConnection = <
  TError = ErrorType<
    | DeleteNotificationConnection401
    | DeleteNotificationConnection403
    | DeleteNotificationConnection404
    | DeleteNotificationConnection409
    | DeleteNotificationConnection422
    | DeleteNotificationConnection500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteNotificationConnection>>,
      TError,
      DeleteNotificationConnectionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteNotificationConnection>>,
  TError,
  DeleteNotificationConnectionMutationVariables,
  TContext
> => {
  return useMutation(getDeleteNotificationConnectionMutationOptions(options), queryClient);
};
export const getGetNotificationConnectionUrl = (id: string) => {
  return `/api/v1/notifications/connections/${id}`;
};

/**
 * @summary Get a notification connection with whether it is sending
 */
export const getNotificationConnection = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NotificationsConnection> => {
  return apiFetch<NotificationsConnection>(getGetNotificationConnectionUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetNotificationConnectionQueryKey = (id: string) => {
  return [`/api/v1/notifications/connections/${id}`] as const;
};

export const getGetNotificationConnectionSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getNotificationConnection>>,
  TError = ErrorType<
    | GetNotificationConnection401
    | GetNotificationConnection403
    | GetNotificationConnection404
    | GetNotificationConnection409
    | GetNotificationConnection422
    | GetNotificationConnection500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationConnection>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetNotificationConnectionQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getNotificationConnection>>> = ({
    signal,
  }) => getNotificationConnection(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof getNotificationConnection>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetNotificationConnectionSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getNotificationConnection>>
>;
export type GetNotificationConnectionSuspenseQueryError = ErrorType<
  | GetNotificationConnection401
  | GetNotificationConnection403
  | GetNotificationConnection404
  | GetNotificationConnection409
  | GetNotificationConnection422
  | GetNotificationConnection500
>;

export function useGetNotificationConnectionSuspense<
  TData = Awaited<ReturnType<typeof getNotificationConnection>>,
  TError = ErrorType<
    | GetNotificationConnection401
    | GetNotificationConnection403
    | GetNotificationConnection404
    | GetNotificationConnection409
    | GetNotificationConnection422
    | GetNotificationConnection500
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationConnection>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNotificationConnectionSuspense<
  TData = Awaited<ReturnType<typeof getNotificationConnection>>,
  TError = ErrorType<
    | GetNotificationConnection401
    | GetNotificationConnection403
    | GetNotificationConnection404
    | GetNotificationConnection409
    | GetNotificationConnection422
    | GetNotificationConnection500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationConnection>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNotificationConnectionSuspense<
  TData = Awaited<ReturnType<typeof getNotificationConnection>>,
  TError = ErrorType<
    | GetNotificationConnection401
    | GetNotificationConnection403
    | GetNotificationConnection404
    | GetNotificationConnection409
    | GetNotificationConnection422
    | GetNotificationConnection500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationConnection>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a notification connection with whether it is sending
 */

export function useGetNotificationConnectionSuspense<
  TData = Awaited<ReturnType<typeof getNotificationConnection>>,
  TError = ErrorType<
    | GetNotificationConnection401
    | GetNotificationConnection403
    | GetNotificationConnection404
    | GetNotificationConnection409
    | GetNotificationConnection422
    | GetNotificationConnection500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationConnection>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetNotificationConnectionSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateNotificationConnectionUrl = (id: string) => {
  return `/api/v1/notifications/connections/${id}`;
};

/**
 * @summary Change a notification connection, which also sends the messages it held while paused
 */
export const updateNotificationConnection = async (
  id: string,
  notificationsEditBody: NotificationsEditBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NotificationsConnection> => {
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
  return apiFetch<NotificationsConnection>(getUpdateNotificationConnectionUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(notificationsEditBody),
  });
};

export const getUpdateNotificationConnectionMutationKey = () =>
  ["updateNotificationConnection"] as const;

export const getUpdateNotificationConnectionMutationOptions = <
  TError = ErrorType<
    | UpdateNotificationConnection401
    | UpdateNotificationConnection403
    | UpdateNotificationConnection404
    | UpdateNotificationConnection409
    | UpdateNotificationConnection422
    | UpdateNotificationConnection500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateNotificationConnection>>,
    TError,
    UpdateNotificationConnectionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateNotificationConnection>>,
  TError,
  UpdateNotificationConnectionMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateNotificationConnectionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateNotificationConnection>>,
    UpdateNotificationConnectionMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return updateNotificationConnection(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateNotificationConnectionMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateNotificationConnection>>
>;
export type UpdateNotificationConnectionMutationBody = NotificationsEditBody;
export type UpdateNotificationConnectionMutationError = ErrorType<
  | UpdateNotificationConnection401
  | UpdateNotificationConnection403
  | UpdateNotificationConnection404
  | UpdateNotificationConnection409
  | UpdateNotificationConnection422
  | UpdateNotificationConnection500
>;
export type UpdateNotificationConnectionMutationVariables = {
  id: string;
  data: NotificationsEditBody;
};

/**
 * @summary Change a notification connection, which also sends the messages it held while paused
 */
export const useUpdateNotificationConnection = <
  TError = ErrorType<
    | UpdateNotificationConnection401
    | UpdateNotificationConnection403
    | UpdateNotificationConnection404
    | UpdateNotificationConnection409
    | UpdateNotificationConnection422
    | UpdateNotificationConnection500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateNotificationConnection>>,
      TError,
      UpdateNotificationConnectionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateNotificationConnection>>,
  TError,
  UpdateNotificationConnectionMutationVariables,
  TContext
> => {
  return useMutation(getUpdateNotificationConnectionMutationOptions(options), queryClient);
};
export const getTestNotificationConnectionUrl = (id: string) => {
  return `/api/v1/notifications/connections/${id}/test`;
};

/**
 * @summary Send a Test message to a saved connection, with any unsaved edits applied. A passing Test of the saved values also sends the messages held while it was paused
 */
export const testNotificationConnection = async (
  id: string,
  notificationsEditBody: NotificationsEditBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NotificationsTestPassed> => {
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
  return apiFetch<NotificationsTestPassed>(getTestNotificationConnectionUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(notificationsEditBody),
  });
};

export const getTestNotificationConnectionMutationKey = () =>
  ["testNotificationConnection"] as const;

export const getTestNotificationConnectionMutationOptions = <
  TError = ErrorType<
    | TestNotificationConnection401
    | TestNotificationConnection403
    | TestNotificationConnection404
    | TestNotificationConnection409
    | TestNotificationConnection422
    | TestNotificationConnection500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof testNotificationConnection>>,
    TError,
    TestNotificationConnectionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof testNotificationConnection>>,
  TError,
  TestNotificationConnectionMutationVariables,
  TContext
> => {
  const mutationKey = getTestNotificationConnectionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof testNotificationConnection>>,
    TestNotificationConnectionMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return testNotificationConnection(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type TestNotificationConnectionMutationResult = NonNullable<
  Awaited<ReturnType<typeof testNotificationConnection>>
>;
export type TestNotificationConnectionMutationBody = NotificationsEditBody;
export type TestNotificationConnectionMutationError = ErrorType<
  | TestNotificationConnection401
  | TestNotificationConnection403
  | TestNotificationConnection404
  | TestNotificationConnection409
  | TestNotificationConnection422
  | TestNotificationConnection500
>;
export type TestNotificationConnectionMutationVariables = {
  id: string;
  data: NotificationsEditBody;
};

/**
 * @summary Send a Test message to a saved connection, with any unsaved edits applied. A passing Test of the saved values also sends the messages held while it was paused
 */
export const useTestNotificationConnection = <
  TError = ErrorType<
    | TestNotificationConnection401
    | TestNotificationConnection403
    | TestNotificationConnection404
    | TestNotificationConnection409
    | TestNotificationConnection422
    | TestNotificationConnection500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof testNotificationConnection>>,
      TError,
      TestNotificationConnectionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof testNotificationConnection>>,
  TError,
  TestNotificationConnectionMutationVariables,
  TContext
> => {
  return useMutation(getTestNotificationConnectionMutationOptions(options), queryClient);
};
export const getListNotificationEventsUrl = () => {
  return `/api/v1/notifications/events`;
};

/**
 * @summary List the events a connection or the bell can get
 */
export const listNotificationEvents = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<EventsOutputBody> => {
  return apiFetch<EventsOutputBody>(getListNotificationEventsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListNotificationEventsQueryKey = () => {
  return [`/api/v1/notifications/events`] as const;
};

export const getListNotificationEventsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listNotificationEvents>>,
  TError = ErrorType<
    | ListNotificationEvents401
    | ListNotificationEvents403
    | ListNotificationEvents409
    | ListNotificationEvents500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationEvents>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListNotificationEventsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listNotificationEvents>>> = ({ signal }) =>
    listNotificationEvents({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listNotificationEvents>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListNotificationEventsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listNotificationEvents>>
>;
export type ListNotificationEventsSuspenseQueryError = ErrorType<
  | ListNotificationEvents401
  | ListNotificationEvents403
  | ListNotificationEvents409
  | ListNotificationEvents500
>;

export function useListNotificationEventsSuspense<
  TData = Awaited<ReturnType<typeof listNotificationEvents>>,
  TError = ErrorType<
    | ListNotificationEvents401
    | ListNotificationEvents403
    | ListNotificationEvents409
    | ListNotificationEvents500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationEvents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNotificationEventsSuspense<
  TData = Awaited<ReturnType<typeof listNotificationEvents>>,
  TError = ErrorType<
    | ListNotificationEvents401
    | ListNotificationEvents403
    | ListNotificationEvents409
    | ListNotificationEvents500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationEvents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNotificationEventsSuspense<
  TData = Awaited<ReturnType<typeof listNotificationEvents>>,
  TError = ErrorType<
    | ListNotificationEvents401
    | ListNotificationEvents403
    | ListNotificationEvents409
    | ListNotificationEvents500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationEvents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the events a connection or the bell can get
 */

export function useListNotificationEventsSuspense<
  TData = Awaited<ReturnType<typeof listNotificationEvents>>,
  TError = ErrorType<
    | ListNotificationEvents401
    | ListNotificationEvents403
    | ListNotificationEvents409
    | ListNotificationEvents500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationEvents>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListNotificationEventsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getListNotificationsUrl = (params?: ListNotificationsParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/notifications/feed?${stringifiedParams}`
    : `/api/v1/notifications/feed`;
};

/**
 * @summary List what the bell showed, newest first, with how many rows are unread
 */
export const listNotifications = async (
  params?: ListNotificationsParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<NotificationsFeed> => {
  return apiFetch<NotificationsFeed>(getListNotificationsUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListNotificationsQueryKey = (params?: ListNotificationsParams) => {
  return [`/api/v1/notifications/feed`, ...(params ? [params] : [])] as const;
};

export const getListNotificationsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listNotifications>>,
  TError = ErrorType<
    | ListNotifications401
    | ListNotifications403
    | ListNotifications409
    | ListNotifications422
    | ListNotifications500
  >,
>(
  params?: ListNotificationsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotifications>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListNotificationsQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listNotifications>>> = ({ signal }) =>
    listNotifications(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotifications>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListNotificationsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listNotifications>>
>;
export type ListNotificationsSuspenseQueryError = ErrorType<
  | ListNotifications401
  | ListNotifications403
  | ListNotifications409
  | ListNotifications422
  | ListNotifications500
>;

export function useListNotificationsSuspense<
  TData = Awaited<ReturnType<typeof listNotifications>>,
  TError = ErrorType<
    | ListNotifications401
    | ListNotifications403
    | ListNotifications409
    | ListNotifications422
    | ListNotifications500
  >,
>(
  params: undefined | ListNotificationsParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotifications>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNotificationsSuspense<
  TData = Awaited<ReturnType<typeof listNotifications>>,
  TError = ErrorType<
    | ListNotifications401
    | ListNotifications403
    | ListNotifications409
    | ListNotifications422
    | ListNotifications500
  >,
>(
  params?: ListNotificationsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotifications>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNotificationsSuspense<
  TData = Awaited<ReturnType<typeof listNotifications>>,
  TError = ErrorType<
    | ListNotifications401
    | ListNotifications403
    | ListNotifications409
    | ListNotifications422
    | ListNotifications500
  >,
>(
  params?: ListNotificationsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotifications>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List what the bell showed, newest first, with how many rows are unread
 */

export function useListNotificationsSuspense<
  TData = Awaited<ReturnType<typeof listNotifications>>,
  TError = ErrorType<
    | ListNotifications401
    | ListNotifications403
    | ListNotifications409
    | ListNotifications422
    | ListNotifications500
  >,
>(
  params?: ListNotificationsParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotifications>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListNotificationsSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getMarkNotificationsSeenUrl = () => {
  return `/api/v1/notifications/feed/seen`;
};

/**
 * @summary Mark the bell's rows up to through as read for the signed-in Admin
 */
export const markNotificationsSeen = async (
  seenInputBody: SeenInputBody,
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
  return apiFetch<void>(getMarkNotificationsSeenUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(seenInputBody),
  });
};

export const getMarkNotificationsSeenMutationKey = () => ["markNotificationsSeen"] as const;

export const getMarkNotificationsSeenMutationOptions = <
  TError = ErrorType<
    | MarkNotificationsSeen401
    | MarkNotificationsSeen403
    | MarkNotificationsSeen409
    | MarkNotificationsSeen422
    | MarkNotificationsSeen500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof markNotificationsSeen>>,
    TError,
    MarkNotificationsSeenMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof markNotificationsSeen>>,
  TError,
  MarkNotificationsSeenMutationVariables,
  TContext
> => {
  const mutationKey = getMarkNotificationsSeenMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof markNotificationsSeen>>,
    MarkNotificationsSeenMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return markNotificationsSeen(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MarkNotificationsSeenMutationResult = NonNullable<
  Awaited<ReturnType<typeof markNotificationsSeen>>
>;
export type MarkNotificationsSeenMutationBody = SeenInputBody;
export type MarkNotificationsSeenMutationError = ErrorType<
  | MarkNotificationsSeen401
  | MarkNotificationsSeen403
  | MarkNotificationsSeen409
  | MarkNotificationsSeen422
  | MarkNotificationsSeen500
>;
export type MarkNotificationsSeenMutationVariables = { data: SeenInputBody };

/**
 * @summary Mark the bell's rows up to through as read for the signed-in Admin
 */
export const useMarkNotificationsSeen = <
  TError = ErrorType<
    | MarkNotificationsSeen401
    | MarkNotificationsSeen403
    | MarkNotificationsSeen409
    | MarkNotificationsSeen422
    | MarkNotificationsSeen500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof markNotificationsSeen>>,
      TError,
      MarkNotificationsSeenMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof markNotificationsSeen>>,
  TError,
  MarkNotificationsSeenMutationVariables,
  TContext
> => {
  return useMutation(getMarkNotificationsSeenMutationOptions(options), queryClient);
};
export const getListNotificationProvidersUrl = () => {
  return `/api/v1/notifications/providers`;
};

/**
 * @summary List the providers a notification connection can send through, each with its fields
 */
export const listNotificationProviders = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<ProvidersOutputBody> => {
  return apiFetch<ProvidersOutputBody>(getListNotificationProvidersUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListNotificationProvidersQueryKey = () => {
  return [`/api/v1/notifications/providers`] as const;
};

export const getListNotificationProvidersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listNotificationProviders>>,
  TError = ErrorType<
    | ListNotificationProviders401
    | ListNotificationProviders403
    | ListNotificationProviders409
    | ListNotificationProviders500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationProviders>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListNotificationProvidersQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listNotificationProviders>>> = ({
    signal,
  }) => listNotificationProviders({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listNotificationProviders>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListNotificationProvidersSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listNotificationProviders>>
>;
export type ListNotificationProvidersSuspenseQueryError = ErrorType<
  | ListNotificationProviders401
  | ListNotificationProviders403
  | ListNotificationProviders409
  | ListNotificationProviders500
>;

export function useListNotificationProvidersSuspense<
  TData = Awaited<ReturnType<typeof listNotificationProviders>>,
  TError = ErrorType<
    | ListNotificationProviders401
    | ListNotificationProviders403
    | ListNotificationProviders409
    | ListNotificationProviders500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNotificationProvidersSuspense<
  TData = Awaited<ReturnType<typeof listNotificationProviders>>,
  TError = ErrorType<
    | ListNotificationProviders401
    | ListNotificationProviders403
    | ListNotificationProviders409
    | ListNotificationProviders500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListNotificationProvidersSuspense<
  TData = Awaited<ReturnType<typeof listNotificationProviders>>,
  TError = ErrorType<
    | ListNotificationProviders401
    | ListNotificationProviders403
    | ListNotificationProviders409
    | ListNotificationProviders500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the providers a notification connection can send through, each with its fields
 */

export function useListNotificationProvidersSuspense<
  TData = Awaited<ReturnType<typeof listNotificationProviders>>,
  TError = ErrorType<
    | ListNotificationProviders401
    | ListNotificationProviders403
    | ListNotificationProviders409
    | ListNotificationProviders500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listNotificationProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListNotificationProvidersSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetNotificationSettingsUrl = () => {
  return `/api/v1/notifications/settings`;
};

/**
 * @summary Get the Notifications settings
 */
export const getNotificationSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<SettingsOutputBody> => {
  return apiFetch<SettingsOutputBody>(getGetNotificationSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetNotificationSettingsQueryKey = () => {
  return [`/api/v1/notifications/settings`] as const;
};

export const getGetNotificationSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getNotificationSettings>>,
  TError = ErrorType<
    | GetNotificationSettings401
    | GetNotificationSettings403
    | GetNotificationSettings409
    | GetNotificationSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetNotificationSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getNotificationSettings>>> = ({
    signal,
  }) => getNotificationSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof getNotificationSettings>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetNotificationSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getNotificationSettings>>
>;
export type GetNotificationSettingsSuspenseQueryError = ErrorType<
  | GetNotificationSettings401
  | GetNotificationSettings403
  | GetNotificationSettings409
  | GetNotificationSettings500
>;

export function useGetNotificationSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNotificationSettings>>,
  TError = ErrorType<
    | GetNotificationSettings401
    | GetNotificationSettings403
    | GetNotificationSettings409
    | GetNotificationSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNotificationSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNotificationSettings>>,
  TError = ErrorType<
    | GetNotificationSettings401
    | GetNotificationSettings403
    | GetNotificationSettings409
    | GetNotificationSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetNotificationSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNotificationSettings>>,
  TError = ErrorType<
    | GetNotificationSettings401
    | GetNotificationSettings403
    | GetNotificationSettings409
    | GetNotificationSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Notifications settings
 */

export function useGetNotificationSettingsSuspense<
  TData = Awaited<ReturnType<typeof getNotificationSettings>>,
  TError = ErrorType<
    | GetNotificationSettings401
    | GetNotificationSettings403
    | GetNotificationSettings409
    | GetNotificationSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getNotificationSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetNotificationSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateNotificationSettingsUrl = () => {
  return `/api/v1/notifications/settings`;
};

/**
 * @summary Change the Notifications settings by key, all or none
 */
export const updateNotificationSettings = async (
  settingsInputBody: SettingsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SettingsOutputBody> => {
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
  return apiFetch<SettingsOutputBody>(getUpdateNotificationSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(settingsInputBody),
  });
};

export const getUpdateNotificationSettingsMutationKey = () =>
  ["updateNotificationSettings"] as const;

export const getUpdateNotificationSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateNotificationSettings401
    | UpdateNotificationSettings403
    | UpdateNotificationSettings409
    | UpdateNotificationSettings422
    | UpdateNotificationSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateNotificationSettings>>,
    TError,
    UpdateNotificationSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateNotificationSettings>>,
  TError,
  UpdateNotificationSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateNotificationSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateNotificationSettings>>,
    UpdateNotificationSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateNotificationSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateNotificationSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateNotificationSettings>>
>;
export type UpdateNotificationSettingsMutationBody = SettingsInputBody;
export type UpdateNotificationSettingsMutationError = ErrorType<
  | UpdateNotificationSettings401
  | UpdateNotificationSettings403
  | UpdateNotificationSettings409
  | UpdateNotificationSettings422
  | UpdateNotificationSettings500
>;
export type UpdateNotificationSettingsMutationVariables = { data: SettingsInputBody };

/**
 * @summary Change the Notifications settings by key, all or none
 */
export const useUpdateNotificationSettings = <
  TError = ErrorType<
    | UpdateNotificationSettings401
    | UpdateNotificationSettings403
    | UpdateNotificationSettings409
    | UpdateNotificationSettings422
    | UpdateNotificationSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateNotificationSettings>>,
      TError,
      UpdateNotificationSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateNotificationSettings>>,
  TError,
  UpdateNotificationSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateNotificationSettingsMutationOptions(options), queryClient);
};
