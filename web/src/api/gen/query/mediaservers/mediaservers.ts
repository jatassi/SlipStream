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
  AddMediaServer401,
  AddMediaServer403,
  AddMediaServer404,
  AddMediaServer409,
  AddMediaServer422,
  AddMediaServer500,
  AddMediaServer502,
  ConfirmMediaServerIdentity401,
  ConfirmMediaServerIdentity403,
  ConfirmMediaServerIdentity404,
  ConfirmMediaServerIdentity409,
  ConfirmMediaServerIdentity422,
  ConfirmMediaServerIdentity500,
  ConfirmMediaServerIdentity502,
  GetMediaServer401,
  GetMediaServer403,
  GetMediaServer404,
  GetMediaServer409,
  GetMediaServer422,
  GetMediaServer500,
  GetMediaServerSettings401,
  GetMediaServerSettings403,
  GetMediaServerSettings409,
  GetMediaServerSettings500,
  GetPlexAccount401,
  GetPlexAccount403,
  GetPlexAccount409,
  GetPlexAccount500,
  GetPlexSignIn401,
  GetPlexSignIn403,
  GetPlexSignIn404,
  GetPlexSignIn409,
  GetPlexSignIn422,
  GetPlexSignIn500,
  GetPlexSignIn502,
  HoldersOutputBody,
  KindsOutputBody,
  ListMediaServerChanges401,
  ListMediaServerChanges403,
  ListMediaServerChanges404,
  ListMediaServerChanges409,
  ListMediaServerChanges422,
  ListMediaServerChanges500,
  ListMediaServerChanges501,
  ListMediaServerChangesParams,
  ListMediaServerCredentialHolders401,
  ListMediaServerCredentialHolders403,
  ListMediaServerCredentialHolders404,
  ListMediaServerCredentialHolders409,
  ListMediaServerCredentialHolders422,
  ListMediaServerCredentialHolders500,
  ListMediaServerKinds401,
  ListMediaServerKinds403,
  ListMediaServerKinds409,
  ListMediaServerKinds500,
  ListMediaServerLibraries401,
  ListMediaServerLibraries403,
  ListMediaServerLibraries404,
  ListMediaServerLibraries409,
  ListMediaServerLibraries422,
  ListMediaServerLibraries500,
  ListMediaServerLibraries502,
  ListMediaServerSessions401,
  ListMediaServerSessions403,
  ListMediaServerSessions404,
  ListMediaServerSessions409,
  ListMediaServerSessions422,
  ListMediaServerSessions500,
  ListMediaServerSessions501,
  ListMediaServers401,
  ListMediaServers403,
  ListMediaServers409,
  ListMediaServers500,
  ListPlexServers401,
  ListPlexServers403,
  ListPlexServers409,
  ListPlexServers422,
  ListPlexServers500,
  ListPlexServers502,
  MappingsOutputBody,
  MediaserversAccountView,
  MediaserversChanges,
  MediaserversDraftBody,
  MediaserversLibraries,
  MediaserversMappingsBody,
  MediaserversPatchBody,
  MediaserversPinBody,
  MediaserversPlexPin,
  MediaserversPlexPinState,
  MediaserversServerView,
  MediaserversSessions,
  MediaserversSettings,
  MediaserversSignInBody,
  MediaserversSignedIn,
  MediaserversTested,
  PlexServersOutputBody,
  RefreshMediaServer401,
  RefreshMediaServer403,
  RefreshMediaServer404,
  RefreshMediaServer409,
  RefreshMediaServer422,
  RefreshMediaServer500,
  RefreshMediaServer501,
  RemoveMediaServer401,
  RemoveMediaServer403,
  RemoveMediaServer404,
  RemoveMediaServer409,
  RemoveMediaServer422,
  RemoveMediaServer500,
  SaveMediaServerMappings401,
  SaveMediaServerMappings403,
  SaveMediaServerMappings404,
  SaveMediaServerMappings409,
  SaveMediaServerMappings422,
  SaveMediaServerMappings500,
  SaveMediaServerMappings502,
  ServersOutputBody,
  SettingsInputBody,
  SignInToMediaServer401,
  SignInToMediaServer403,
  SignInToMediaServer404,
  SignInToMediaServer409,
  SignInToMediaServer422,
  SignInToMediaServer500,
  SignInToMediaServer502,
  SignOutOfPlex401,
  SignOutOfPlex403,
  SignOutOfPlex409,
  SignOutOfPlex500,
  StartPlexSignIn401,
  StartPlexSignIn403,
  StartPlexSignIn409,
  StartPlexSignIn422,
  StartPlexSignIn500,
  StartPlexSignIn502,
  TestMediaServer401,
  TestMediaServer403,
  TestMediaServer404,
  TestMediaServer409,
  TestMediaServer422,
  TestMediaServer500,
  UpdateMediaServer401,
  UpdateMediaServer403,
  UpdateMediaServer404,
  UpdateMediaServer409,
  UpdateMediaServer422,
  UpdateMediaServer500,
  UpdateMediaServer502,
  UpdateMediaServerSettings401,
  UpdateMediaServerSettings403,
  UpdateMediaServerSettings409,
  UpdateMediaServerSettings422,
  UpdateMediaServerSettings500,
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

export const getListMediaServersUrl = () => {
  return `/api/v1/media-servers`;
};

/**
 * @summary List the Media Servers SlipStream keeps up to date, each with its state
 */
export const listMediaServers = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<ServersOutputBody> => {
  return apiFetch<ServersOutputBody>(getListMediaServersUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListMediaServersQueryKey = () => {
  return [`/api/v1/media-servers`] as const;
};

export const getListMediaServersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMediaServers>>,
  TError = ErrorType<
    ListMediaServers401 | ListMediaServers403 | ListMediaServers409 | ListMediaServers500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServers>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMediaServersQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMediaServers>>> = ({ signal }) =>
    listMediaServers({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServers>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMediaServersSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listMediaServers>>
>;
export type ListMediaServersSuspenseQueryError = ErrorType<
  ListMediaServers401 | ListMediaServers403 | ListMediaServers409 | ListMediaServers500
>;

export function useListMediaServersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServers>>,
  TError = ErrorType<
    ListMediaServers401 | ListMediaServers403 | ListMediaServers409 | ListMediaServers500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServers>>,
  TError = ErrorType<
    ListMediaServers401 | ListMediaServers403 | ListMediaServers409 | ListMediaServers500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServers>>,
  TError = ErrorType<
    ListMediaServers401 | ListMediaServers403 | ListMediaServers409 | ListMediaServers500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Media Servers SlipStream keeps up to date, each with its state
 */

export function useListMediaServersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServers>>,
  TError = ErrorType<
    ListMediaServers401 | ListMediaServers403 | ListMediaServers409 | ListMediaServers500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMediaServersSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getAddMediaServerUrl = () => {
  return `/api/v1/media-servers`;
};

/**
 * @summary Add a Media Server once it responds from its reaching Node and accepts SlipStream's sign-in
 */
export const addMediaServer = async (
  mediaserversDraftBody: MediaserversDraftBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversServerView> => {
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
  return apiFetch<MediaserversServerView>(getAddMediaServerUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(mediaserversDraftBody),
  });
};

export const getAddMediaServerMutationKey = () => ["addMediaServer"] as const;

export const getAddMediaServerMutationOptions = <
  TError = ErrorType<
    | AddMediaServer401
    | AddMediaServer403
    | AddMediaServer404
    | AddMediaServer409
    | AddMediaServer422
    | AddMediaServer500
    | AddMediaServer502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof addMediaServer>>,
    TError,
    AddMediaServerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof addMediaServer>>,
  TError,
  AddMediaServerMutationVariables,
  TContext
> => {
  const mutationKey = getAddMediaServerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof addMediaServer>>,
    AddMediaServerMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return addMediaServer(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type AddMediaServerMutationResult = NonNullable<Awaited<ReturnType<typeof addMediaServer>>>;
export type AddMediaServerMutationBody = MediaserversDraftBody;
export type AddMediaServerMutationError = ErrorType<
  | AddMediaServer401
  | AddMediaServer403
  | AddMediaServer404
  | AddMediaServer409
  | AddMediaServer422
  | AddMediaServer500
  | AddMediaServer502
>;
export type AddMediaServerMutationVariables = { data: MediaserversDraftBody };

/**
 * @summary Add a Media Server once it responds from its reaching Node and accepts SlipStream's sign-in
 */
export const useAddMediaServer = <
  TError = ErrorType<
    | AddMediaServer401
    | AddMediaServer403
    | AddMediaServer404
    | AddMediaServer409
    | AddMediaServer422
    | AddMediaServer500
    | AddMediaServer502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof addMediaServer>>,
      TError,
      AddMediaServerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof addMediaServer>>,
  TError,
  AddMediaServerMutationVariables,
  TContext
> => {
  return useMutation(getAddMediaServerMutationOptions(options), queryClient);
};
export const getListMediaServerKindsUrl = () => {
  return `/api/v1/media-servers/kinds`;
};

/**
 * @summary List the kinds of Media Server SlipStream supports, with the sign-in each takes and its oldest supported version
 */
export const listMediaServerKinds = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<KindsOutputBody> => {
  return apiFetch<KindsOutputBody>(getListMediaServerKindsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListMediaServerKindsQueryKey = () => {
  return [`/api/v1/media-servers/kinds`] as const;
};

export const getListMediaServerKindsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMediaServerKinds>>,
  TError = ErrorType<
    | ListMediaServerKinds401
    | ListMediaServerKinds403
    | ListMediaServerKinds409
    | ListMediaServerKinds500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerKinds>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMediaServerKindsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMediaServerKinds>>> = ({ signal }) =>
    listMediaServerKinds({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerKinds>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMediaServerKindsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listMediaServerKinds>>
>;
export type ListMediaServerKindsSuspenseQueryError = ErrorType<
  | ListMediaServerKinds401
  | ListMediaServerKinds403
  | ListMediaServerKinds409
  | ListMediaServerKinds500
>;

export function useListMediaServerKindsSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerKinds>>,
  TError = ErrorType<
    | ListMediaServerKinds401
    | ListMediaServerKinds403
    | ListMediaServerKinds409
    | ListMediaServerKinds500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerKinds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerKindsSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerKinds>>,
  TError = ErrorType<
    | ListMediaServerKinds401
    | ListMediaServerKinds403
    | ListMediaServerKinds409
    | ListMediaServerKinds500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerKinds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerKindsSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerKinds>>,
  TError = ErrorType<
    | ListMediaServerKinds401
    | ListMediaServerKinds403
    | ListMediaServerKinds409
    | ListMediaServerKinds500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerKinds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the kinds of Media Server SlipStream supports, with the sign-in each takes and its oldest supported version
 */

export function useListMediaServerKindsSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerKinds>>,
  TError = ErrorType<
    | ListMediaServerKinds401
    | ListMediaServerKinds403
    | ListMediaServerKinds409
    | ListMediaServerKinds500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerKinds>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMediaServerKindsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSignOutOfPlexUrl = () => {
  return `/api/v1/media-servers/plex/account`;
};

/**
 * @summary Sign out of Plex, which also removes every Plex server
 */
export const signOutOfPlex = async (options?: Parameters<typeof apiFetch>[1]): Promise<void> => {
  return apiFetch<void>(getSignOutOfPlexUrl(), {
    ...options,
    method: "DELETE",
  });
};

export const getSignOutOfPlexMutationKey = () => ["signOutOfPlex"] as const;

export const getSignOutOfPlexMutationOptions = <
  TError = ErrorType<SignOutOfPlex401 | SignOutOfPlex403 | SignOutOfPlex409 | SignOutOfPlex500>,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<Awaited<ReturnType<typeof signOutOfPlex>>, TError, void, TContext>;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof signOutOfPlex>>, TError, void, TContext> => {
  const mutationKey = getSignOutOfPlexMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof signOutOfPlex>>, void> = () => {
    return signOutOfPlex(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SignOutOfPlexMutationResult = NonNullable<Awaited<ReturnType<typeof signOutOfPlex>>>;

export type SignOutOfPlexMutationError = ErrorType<
  SignOutOfPlex401 | SignOutOfPlex403 | SignOutOfPlex409 | SignOutOfPlex500
>;

/**
 * @summary Sign out of Plex, which also removes every Plex server
 */
export const useSignOutOfPlex = <
  TError = ErrorType<SignOutOfPlex401 | SignOutOfPlex403 | SignOutOfPlex409 | SignOutOfPlex500>,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof signOutOfPlex>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof signOutOfPlex>>, TError, void, TContext> => {
  return useMutation(getSignOutOfPlexMutationOptions(options), queryClient);
};
export const getGetPlexAccountUrl = () => {
  return `/api/v1/media-servers/plex/account`;
};

/**
 * @summary Get the Plex account SlipStream is signed in to and when its sign-in expires
 */
export const getPlexAccount = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversAccountView> => {
  return apiFetch<MediaserversAccountView>(getGetPlexAccountUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetPlexAccountQueryKey = () => {
  return [`/api/v1/media-servers/plex/account`] as const;
};

export const getGetPlexAccountSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getPlexAccount>>,
  TError = ErrorType<GetPlexAccount401 | GetPlexAccount403 | GetPlexAccount409 | GetPlexAccount500>,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexAccount>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetPlexAccountQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getPlexAccount>>> = ({ signal }) =>
    getPlexAccount({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexAccount>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetPlexAccountSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getPlexAccount>>
>;
export type GetPlexAccountSuspenseQueryError = ErrorType<
  GetPlexAccount401 | GetPlexAccount403 | GetPlexAccount409 | GetPlexAccount500
>;

export function useGetPlexAccountSuspense<
  TData = Awaited<ReturnType<typeof getPlexAccount>>,
  TError = ErrorType<GetPlexAccount401 | GetPlexAccount403 | GetPlexAccount409 | GetPlexAccount500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexAccount>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetPlexAccountSuspense<
  TData = Awaited<ReturnType<typeof getPlexAccount>>,
  TError = ErrorType<GetPlexAccount401 | GetPlexAccount403 | GetPlexAccount409 | GetPlexAccount500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexAccount>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetPlexAccountSuspense<
  TData = Awaited<ReturnType<typeof getPlexAccount>>,
  TError = ErrorType<GetPlexAccount401 | GetPlexAccount403 | GetPlexAccount409 | GetPlexAccount500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexAccount>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Plex account SlipStream is signed in to and when its sign-in expires
 */

export function useGetPlexAccountSuspense<
  TData = Awaited<ReturnType<typeof getPlexAccount>>,
  TError = ErrorType<GetPlexAccount401 | GetPlexAccount403 | GetPlexAccount409 | GetPlexAccount500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexAccount>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetPlexAccountSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getStartPlexSignInUrl = () => {
  return `/api/v1/media-servers/plex/pin`;
};

/**
 * @summary Start a Plex sign-in and get the plex.tv page the Admin signs in on
 */
export const startPlexSignIn = async (
  mediaserversPinBody: MediaserversPinBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversPlexPin> => {
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
  return apiFetch<MediaserversPlexPin>(getStartPlexSignInUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(mediaserversPinBody),
  });
};

export const getStartPlexSignInMutationKey = () => ["startPlexSignIn"] as const;

export const getStartPlexSignInMutationOptions = <
  TError = ErrorType<
    | StartPlexSignIn401
    | StartPlexSignIn403
    | StartPlexSignIn409
    | StartPlexSignIn422
    | StartPlexSignIn500
    | StartPlexSignIn502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof startPlexSignIn>>,
    TError,
    StartPlexSignInMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof startPlexSignIn>>,
  TError,
  StartPlexSignInMutationVariables,
  TContext
> => {
  const mutationKey = getStartPlexSignInMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof startPlexSignIn>>,
    StartPlexSignInMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return startPlexSignIn(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type StartPlexSignInMutationResult = NonNullable<
  Awaited<ReturnType<typeof startPlexSignIn>>
>;
export type StartPlexSignInMutationBody = MediaserversPinBody;
export type StartPlexSignInMutationError = ErrorType<
  | StartPlexSignIn401
  | StartPlexSignIn403
  | StartPlexSignIn409
  | StartPlexSignIn422
  | StartPlexSignIn500
  | StartPlexSignIn502
>;
export type StartPlexSignInMutationVariables = { data: MediaserversPinBody };

/**
 * @summary Start a Plex sign-in and get the plex.tv page the Admin signs in on
 */
export const useStartPlexSignIn = <
  TError = ErrorType<
    | StartPlexSignIn401
    | StartPlexSignIn403
    | StartPlexSignIn409
    | StartPlexSignIn422
    | StartPlexSignIn500
    | StartPlexSignIn502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof startPlexSignIn>>,
      TError,
      StartPlexSignInMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof startPlexSignIn>>,
  TError,
  StartPlexSignInMutationVariables,
  TContext
> => {
  return useMutation(getStartPlexSignInMutationOptions(options), queryClient);
};
export const getGetPlexSignInUrl = (id: number) => {
  return `/api/v1/media-servers/plex/pin/${id}`;
};

/**
 * @summary Get whether a Plex sign-in is still waiting, has expired or signed in
 */
export const getPlexSignIn = async (
  id: number,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversPlexPinState> => {
  return apiFetch<MediaserversPlexPinState>(getGetPlexSignInUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetPlexSignInQueryKey = (id: number) => {
  return [`/api/v1/media-servers/plex/pin/${id}`] as const;
};

export const getGetPlexSignInSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getPlexSignIn>>,
  TError = ErrorType<
    | GetPlexSignIn401
    | GetPlexSignIn403
    | GetPlexSignIn404
    | GetPlexSignIn409
    | GetPlexSignIn422
    | GetPlexSignIn500
    | GetPlexSignIn502
  >,
>(
  id: number,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexSignIn>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetPlexSignInQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getPlexSignIn>>> = ({ signal }) =>
    getPlexSignIn(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexSignIn>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetPlexSignInSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getPlexSignIn>>
>;
export type GetPlexSignInSuspenseQueryError = ErrorType<
  | GetPlexSignIn401
  | GetPlexSignIn403
  | GetPlexSignIn404
  | GetPlexSignIn409
  | GetPlexSignIn422
  | GetPlexSignIn500
  | GetPlexSignIn502
>;

export function useGetPlexSignInSuspense<
  TData = Awaited<ReturnType<typeof getPlexSignIn>>,
  TError = ErrorType<
    | GetPlexSignIn401
    | GetPlexSignIn403
    | GetPlexSignIn404
    | GetPlexSignIn409
    | GetPlexSignIn422
    | GetPlexSignIn500
    | GetPlexSignIn502
  >,
>(
  id: number,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexSignIn>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetPlexSignInSuspense<
  TData = Awaited<ReturnType<typeof getPlexSignIn>>,
  TError = ErrorType<
    | GetPlexSignIn401
    | GetPlexSignIn403
    | GetPlexSignIn404
    | GetPlexSignIn409
    | GetPlexSignIn422
    | GetPlexSignIn500
    | GetPlexSignIn502
  >,
>(
  id: number,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexSignIn>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetPlexSignInSuspense<
  TData = Awaited<ReturnType<typeof getPlexSignIn>>,
  TError = ErrorType<
    | GetPlexSignIn401
    | GetPlexSignIn403
    | GetPlexSignIn404
    | GetPlexSignIn409
    | GetPlexSignIn422
    | GetPlexSignIn500
    | GetPlexSignIn502
  >,
>(
  id: number,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexSignIn>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get whether a Plex sign-in is still waiting, has expired or signed in
 */

export function useGetPlexSignInSuspense<
  TData = Awaited<ReturnType<typeof getPlexSignIn>>,
  TError = ErrorType<
    | GetPlexSignIn401
    | GetPlexSignIn403
    | GetPlexSignIn404
    | GetPlexSignIn409
    | GetPlexSignIn422
    | GetPlexSignIn500
    | GetPlexSignIn502
  >,
>(
  id: number,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getPlexSignIn>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetPlexSignInSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getListPlexServersUrl = () => {
  return `/api/v1/media-servers/plex/servers`;
};

/**
 * @summary List the Plex account's servers and which Nodes reach each of their connections
 */
export const listPlexServers = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<PlexServersOutputBody> => {
  return apiFetch<PlexServersOutputBody>(getListPlexServersUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListPlexServersQueryKey = () => {
  return [`/api/v1/media-servers/plex/servers`] as const;
};

export const getListPlexServersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listPlexServers>>,
  TError = ErrorType<
    | ListPlexServers401
    | ListPlexServers403
    | ListPlexServers409
    | ListPlexServers422
    | ListPlexServers500
    | ListPlexServers502
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPlexServers>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListPlexServersQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listPlexServers>>> = ({ signal }) =>
    listPlexServers({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPlexServers>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListPlexServersSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listPlexServers>>
>;
export type ListPlexServersSuspenseQueryError = ErrorType<
  | ListPlexServers401
  | ListPlexServers403
  | ListPlexServers409
  | ListPlexServers422
  | ListPlexServers500
  | ListPlexServers502
>;

export function useListPlexServersSuspense<
  TData = Awaited<ReturnType<typeof listPlexServers>>,
  TError = ErrorType<
    | ListPlexServers401
    | ListPlexServers403
    | ListPlexServers409
    | ListPlexServers422
    | ListPlexServers500
    | ListPlexServers502
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPlexServers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListPlexServersSuspense<
  TData = Awaited<ReturnType<typeof listPlexServers>>,
  TError = ErrorType<
    | ListPlexServers401
    | ListPlexServers403
    | ListPlexServers409
    | ListPlexServers422
    | ListPlexServers500
    | ListPlexServers502
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPlexServers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListPlexServersSuspense<
  TData = Awaited<ReturnType<typeof listPlexServers>>,
  TError = ErrorType<
    | ListPlexServers401
    | ListPlexServers403
    | ListPlexServers409
    | ListPlexServers422
    | ListPlexServers500
    | ListPlexServers502
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPlexServers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Plex account's servers and which Nodes reach each of their connections
 */

export function useListPlexServersSuspense<
  TData = Awaited<ReturnType<typeof listPlexServers>>,
  TError = ErrorType<
    | ListPlexServers401
    | ListPlexServers403
    | ListPlexServers409
    | ListPlexServers422
    | ListPlexServers500
    | ListPlexServers502
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPlexServers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListPlexServersSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getGetMediaServerSettingsUrl = () => {
  return `/api/v1/media-servers/settings`;
};

/**
 * @summary Get the Media Servers settings
 */
export const getMediaServerSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversSettings> => {
  return apiFetch<MediaserversSettings>(getGetMediaServerSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetMediaServerSettingsQueryKey = () => {
  return [`/api/v1/media-servers/settings`] as const;
};

export const getGetMediaServerSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getMediaServerSettings>>,
  TError = ErrorType<
    | GetMediaServerSettings401
    | GetMediaServerSettings403
    | GetMediaServerSettings409
    | GetMediaServerSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServerSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetMediaServerSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getMediaServerSettings>>> = ({ signal }) =>
    getMediaServerSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof getMediaServerSettings>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetMediaServerSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getMediaServerSettings>>
>;
export type GetMediaServerSettingsSuspenseQueryError = ErrorType<
  | GetMediaServerSettings401
  | GetMediaServerSettings403
  | GetMediaServerSettings409
  | GetMediaServerSettings500
>;

export function useGetMediaServerSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMediaServerSettings>>,
  TError = ErrorType<
    | GetMediaServerSettings401
    | GetMediaServerSettings403
    | GetMediaServerSettings409
    | GetMediaServerSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServerSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMediaServerSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMediaServerSettings>>,
  TError = ErrorType<
    | GetMediaServerSettings401
    | GetMediaServerSettings403
    | GetMediaServerSettings409
    | GetMediaServerSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServerSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMediaServerSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMediaServerSettings>>,
  TError = ErrorType<
    | GetMediaServerSettings401
    | GetMediaServerSettings403
    | GetMediaServerSettings409
    | GetMediaServerSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServerSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Media Servers settings
 */

export function useGetMediaServerSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMediaServerSettings>>,
  TError = ErrorType<
    | GetMediaServerSettings401
    | GetMediaServerSettings403
    | GetMediaServerSettings409
    | GetMediaServerSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServerSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetMediaServerSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateMediaServerSettingsUrl = () => {
  return `/api/v1/media-servers/settings`;
};

/**
 * @summary Change the Media Servers settings by key, all or none
 */
export const updateMediaServerSettings = async (
  settingsInputBody: SettingsInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversSettings> => {
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
  return apiFetch<MediaserversSettings>(getUpdateMediaServerSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(settingsInputBody),
  });
};

export const getUpdateMediaServerSettingsMutationKey = () => ["updateMediaServerSettings"] as const;

export const getUpdateMediaServerSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateMediaServerSettings401
    | UpdateMediaServerSettings403
    | UpdateMediaServerSettings409
    | UpdateMediaServerSettings422
    | UpdateMediaServerSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateMediaServerSettings>>,
    TError,
    UpdateMediaServerSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateMediaServerSettings>>,
  TError,
  UpdateMediaServerSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateMediaServerSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateMediaServerSettings>>,
    UpdateMediaServerSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateMediaServerSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateMediaServerSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateMediaServerSettings>>
>;
export type UpdateMediaServerSettingsMutationBody = SettingsInputBody;
export type UpdateMediaServerSettingsMutationError = ErrorType<
  | UpdateMediaServerSettings401
  | UpdateMediaServerSettings403
  | UpdateMediaServerSettings409
  | UpdateMediaServerSettings422
  | UpdateMediaServerSettings500
>;
export type UpdateMediaServerSettingsMutationVariables = { data: SettingsInputBody };

/**
 * @summary Change the Media Servers settings by key, all or none
 */
export const useUpdateMediaServerSettings = <
  TError = ErrorType<
    | UpdateMediaServerSettings401
    | UpdateMediaServerSettings403
    | UpdateMediaServerSettings409
    | UpdateMediaServerSettings422
    | UpdateMediaServerSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateMediaServerSettings>>,
      TError,
      UpdateMediaServerSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateMediaServerSettings>>,
  TError,
  UpdateMediaServerSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateMediaServerSettingsMutationOptions(options), queryClient);
};
export const getSignInToMediaServerUrl = () => {
  return `/api/v1/media-servers/sign-in`;
};

/**
 * @summary Sign in to a Jellyfin or Emby server as an administrator once, creating an API key named SlipStream for a draft server
 */
export const signInToMediaServer = async (
  mediaserversSignInBody: MediaserversSignInBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversSignedIn> => {
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
  return apiFetch<MediaserversSignedIn>(getSignInToMediaServerUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(mediaserversSignInBody),
  });
};

export const getSignInToMediaServerMutationKey = () => ["signInToMediaServer"] as const;

export const getSignInToMediaServerMutationOptions = <
  TError = ErrorType<
    | SignInToMediaServer401
    | SignInToMediaServer403
    | SignInToMediaServer404
    | SignInToMediaServer409
    | SignInToMediaServer422
    | SignInToMediaServer500
    | SignInToMediaServer502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof signInToMediaServer>>,
    TError,
    SignInToMediaServerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof signInToMediaServer>>,
  TError,
  SignInToMediaServerMutationVariables,
  TContext
> => {
  const mutationKey = getSignInToMediaServerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof signInToMediaServer>>,
    SignInToMediaServerMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return signInToMediaServer(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SignInToMediaServerMutationResult = NonNullable<
  Awaited<ReturnType<typeof signInToMediaServer>>
>;
export type SignInToMediaServerMutationBody = MediaserversSignInBody;
export type SignInToMediaServerMutationError = ErrorType<
  | SignInToMediaServer401
  | SignInToMediaServer403
  | SignInToMediaServer404
  | SignInToMediaServer409
  | SignInToMediaServer422
  | SignInToMediaServer500
  | SignInToMediaServer502
>;
export type SignInToMediaServerMutationVariables = { data: MediaserversSignInBody };

/**
 * @summary Sign in to a Jellyfin or Emby server as an administrator once, creating an API key named SlipStream for a draft server
 */
export const useSignInToMediaServer = <
  TError = ErrorType<
    | SignInToMediaServer401
    | SignInToMediaServer403
    | SignInToMediaServer404
    | SignInToMediaServer409
    | SignInToMediaServer422
    | SignInToMediaServer500
    | SignInToMediaServer502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof signInToMediaServer>>,
      TError,
      SignInToMediaServerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof signInToMediaServer>>,
  TError,
  SignInToMediaServerMutationVariables,
  TContext
> => {
  return useMutation(getSignInToMediaServerMutationOptions(options), queryClient);
};
export const getRemoveMediaServerUrl = (id: string) => {
  return `/api/v1/media-servers/${id}`;
};

/**
 * @summary Remove a Media Server, which also deletes its queued changes, its path index and its play history
 */
export const removeMediaServer = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getRemoveMediaServerUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getRemoveMediaServerMutationKey = () => ["removeMediaServer"] as const;

export const getRemoveMediaServerMutationOptions = <
  TError = ErrorType<
    | RemoveMediaServer401
    | RemoveMediaServer403
    | RemoveMediaServer404
    | RemoveMediaServer409
    | RemoveMediaServer422
    | RemoveMediaServer500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof removeMediaServer>>,
    TError,
    RemoveMediaServerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof removeMediaServer>>,
  TError,
  RemoveMediaServerMutationVariables,
  TContext
> => {
  const mutationKey = getRemoveMediaServerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof removeMediaServer>>,
    RemoveMediaServerMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return removeMediaServer(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RemoveMediaServerMutationResult = NonNullable<
  Awaited<ReturnType<typeof removeMediaServer>>
>;

export type RemoveMediaServerMutationError = ErrorType<
  | RemoveMediaServer401
  | RemoveMediaServer403
  | RemoveMediaServer404
  | RemoveMediaServer409
  | RemoveMediaServer422
  | RemoveMediaServer500
>;
export type RemoveMediaServerMutationVariables = { id: string };

/**
 * @summary Remove a Media Server, which also deletes its queued changes, its path index and its play history
 */
export const useRemoveMediaServer = <
  TError = ErrorType<
    | RemoveMediaServer401
    | RemoveMediaServer403
    | RemoveMediaServer404
    | RemoveMediaServer409
    | RemoveMediaServer422
    | RemoveMediaServer500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof removeMediaServer>>,
      TError,
      RemoveMediaServerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof removeMediaServer>>,
  TError,
  RemoveMediaServerMutationVariables,
  TContext
> => {
  return useMutation(getRemoveMediaServerMutationOptions(options), queryClient);
};
export const getGetMediaServerUrl = (id: string) => {
  return `/api/v1/media-servers/${id}`;
};

/**
 * @summary Get a Media Server with its state
 */
export const getMediaServer = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversServerView> => {
  return apiFetch<MediaserversServerView>(getGetMediaServerUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetMediaServerQueryKey = (id: string) => {
  return [`/api/v1/media-servers/${id}`] as const;
};

export const getGetMediaServerSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getMediaServer>>,
  TError = ErrorType<
    | GetMediaServer401
    | GetMediaServer403
    | GetMediaServer404
    | GetMediaServer409
    | GetMediaServer422
    | GetMediaServer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServer>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetMediaServerQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getMediaServer>>> = ({ signal }) =>
    getMediaServer(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServer>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetMediaServerSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getMediaServer>>
>;
export type GetMediaServerSuspenseQueryError = ErrorType<
  | GetMediaServer401
  | GetMediaServer403
  | GetMediaServer404
  | GetMediaServer409
  | GetMediaServer422
  | GetMediaServer500
>;

export function useGetMediaServerSuspense<
  TData = Awaited<ReturnType<typeof getMediaServer>>,
  TError = ErrorType<
    | GetMediaServer401
    | GetMediaServer403
    | GetMediaServer404
    | GetMediaServer409
    | GetMediaServer422
    | GetMediaServer500
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServer>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMediaServerSuspense<
  TData = Awaited<ReturnType<typeof getMediaServer>>,
  TError = ErrorType<
    | GetMediaServer401
    | GetMediaServer403
    | GetMediaServer404
    | GetMediaServer409
    | GetMediaServer422
    | GetMediaServer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServer>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMediaServerSuspense<
  TData = Awaited<ReturnType<typeof getMediaServer>>,
  TError = ErrorType<
    | GetMediaServer401
    | GetMediaServer403
    | GetMediaServer404
    | GetMediaServer409
    | GetMediaServer422
    | GetMediaServer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServer>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Media Server with its state
 */

export function useGetMediaServerSuspense<
  TData = Awaited<ReturnType<typeof getMediaServer>>,
  TError = ErrorType<
    | GetMediaServer401
    | GetMediaServer403
    | GetMediaServer404
    | GetMediaServer409
    | GetMediaServer422
    | GetMediaServer500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMediaServer>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetMediaServerSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateMediaServerUrl = (id: string) => {
  return `/api/v1/media-servers/${id}`;
};

/**
 * @summary Change a Media Server, testing it again when the change is to its address, reaching Node or API key
 */
export const updateMediaServer = async (
  id: string,
  mediaserversPatchBody: MediaserversPatchBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversServerView> => {
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
  return apiFetch<MediaserversServerView>(getUpdateMediaServerUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(mediaserversPatchBody),
  });
};

export const getUpdateMediaServerMutationKey = () => ["updateMediaServer"] as const;

export const getUpdateMediaServerMutationOptions = <
  TError = ErrorType<
    | UpdateMediaServer401
    | UpdateMediaServer403
    | UpdateMediaServer404
    | UpdateMediaServer409
    | UpdateMediaServer422
    | UpdateMediaServer500
    | UpdateMediaServer502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateMediaServer>>,
    TError,
    UpdateMediaServerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateMediaServer>>,
  TError,
  UpdateMediaServerMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateMediaServerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateMediaServer>>,
    UpdateMediaServerMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return updateMediaServer(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateMediaServerMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateMediaServer>>
>;
export type UpdateMediaServerMutationBody = MediaserversPatchBody;
export type UpdateMediaServerMutationError = ErrorType<
  | UpdateMediaServer401
  | UpdateMediaServer403
  | UpdateMediaServer404
  | UpdateMediaServer409
  | UpdateMediaServer422
  | UpdateMediaServer500
  | UpdateMediaServer502
>;
export type UpdateMediaServerMutationVariables = { id: string; data: MediaserversPatchBody };

/**
 * @summary Change a Media Server, testing it again when the change is to its address, reaching Node or API key
 */
export const useUpdateMediaServer = <
  TError = ErrorType<
    | UpdateMediaServer401
    | UpdateMediaServer403
    | UpdateMediaServer404
    | UpdateMediaServer409
    | UpdateMediaServer422
    | UpdateMediaServer500
    | UpdateMediaServer502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateMediaServer>>,
      TError,
      UpdateMediaServerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateMediaServer>>,
  TError,
  UpdateMediaServerMutationVariables,
  TContext
> => {
  return useMutation(getUpdateMediaServerMutationOptions(options), queryClient);
};
export const getListMediaServerChangesUrl = (id: string, params?: ListMediaServerChangesParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/media-servers/${id}/changes?${stringifiedParams}`
    : `/api/v1/media-servers/${id}/changes`;
};

/**
 * @summary List a Media Server's recent changes, newest first, with their results and what each move couldn't carry
 */
export const listMediaServerChanges = async (
  id: string,
  params?: ListMediaServerChangesParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversChanges> => {
  return apiFetch<MediaserversChanges>(getListMediaServerChangesUrl(id, params), {
    ...options,
    method: "GET",
  });
};

export const getListMediaServerChangesQueryKey = (
  id: string,
  params?: ListMediaServerChangesParams,
) => {
  return [`/api/v1/media-servers/${id}/changes`, ...(params ? [params] : [])] as const;
};

export const getListMediaServerChangesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMediaServerChanges>>,
  TError = ErrorType<
    | ListMediaServerChanges401
    | ListMediaServerChanges403
    | ListMediaServerChanges404
    | ListMediaServerChanges409
    | ListMediaServerChanges422
    | ListMediaServerChanges500
    | ListMediaServerChanges501
  >,
>(
  id: string,
  params?: ListMediaServerChangesParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerChanges>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMediaServerChangesQueryKey(id, params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMediaServerChanges>>> = ({ signal }) =>
    listMediaServerChanges(id, params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listMediaServerChanges>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMediaServerChangesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listMediaServerChanges>>
>;
export type ListMediaServerChangesSuspenseQueryError = ErrorType<
  | ListMediaServerChanges401
  | ListMediaServerChanges403
  | ListMediaServerChanges404
  | ListMediaServerChanges409
  | ListMediaServerChanges422
  | ListMediaServerChanges500
  | ListMediaServerChanges501
>;

export function useListMediaServerChangesSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerChanges>>,
  TError = ErrorType<
    | ListMediaServerChanges401
    | ListMediaServerChanges403
    | ListMediaServerChanges404
    | ListMediaServerChanges409
    | ListMediaServerChanges422
    | ListMediaServerChanges500
    | ListMediaServerChanges501
  >,
>(
  id: string,
  params: undefined | ListMediaServerChangesParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerChanges>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerChangesSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerChanges>>,
  TError = ErrorType<
    | ListMediaServerChanges401
    | ListMediaServerChanges403
    | ListMediaServerChanges404
    | ListMediaServerChanges409
    | ListMediaServerChanges422
    | ListMediaServerChanges500
    | ListMediaServerChanges501
  >,
>(
  id: string,
  params?: ListMediaServerChangesParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerChanges>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerChangesSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerChanges>>,
  TError = ErrorType<
    | ListMediaServerChanges401
    | ListMediaServerChanges403
    | ListMediaServerChanges404
    | ListMediaServerChanges409
    | ListMediaServerChanges422
    | ListMediaServerChanges500
    | ListMediaServerChanges501
  >,
>(
  id: string,
  params?: ListMediaServerChangesParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerChanges>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List a Media Server's recent changes, newest first, with their results and what each move couldn't carry
 */

export function useListMediaServerChangesSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerChanges>>,
  TError = ErrorType<
    | ListMediaServerChanges401
    | ListMediaServerChanges403
    | ListMediaServerChanges404
    | ListMediaServerChanges409
    | ListMediaServerChanges422
    | ListMediaServerChanges500
    | ListMediaServerChanges501
  >,
>(
  id: string,
  params?: ListMediaServerChangesParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerChanges>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMediaServerChangesSuspenseQueryOptions(id, params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getConfirmMediaServerIdentityUrl = (id: string) => {
  return `/api/v1/media-servers/${id}/confirm-identity`;
};

/**
 * @summary Accept the server that responds at a Media Server's address now as that Media Server
 */
export const confirmMediaServerIdentity = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversServerView> => {
  return apiFetch<MediaserversServerView>(getConfirmMediaServerIdentityUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getConfirmMediaServerIdentityMutationKey = () =>
  ["confirmMediaServerIdentity"] as const;

export const getConfirmMediaServerIdentityMutationOptions = <
  TError = ErrorType<
    | ConfirmMediaServerIdentity401
    | ConfirmMediaServerIdentity403
    | ConfirmMediaServerIdentity404
    | ConfirmMediaServerIdentity409
    | ConfirmMediaServerIdentity422
    | ConfirmMediaServerIdentity500
    | ConfirmMediaServerIdentity502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof confirmMediaServerIdentity>>,
    TError,
    ConfirmMediaServerIdentityMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof confirmMediaServerIdentity>>,
  TError,
  ConfirmMediaServerIdentityMutationVariables,
  TContext
> => {
  const mutationKey = getConfirmMediaServerIdentityMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof confirmMediaServerIdentity>>,
    ConfirmMediaServerIdentityMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return confirmMediaServerIdentity(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ConfirmMediaServerIdentityMutationResult = NonNullable<
  Awaited<ReturnType<typeof confirmMediaServerIdentity>>
>;

export type ConfirmMediaServerIdentityMutationError = ErrorType<
  | ConfirmMediaServerIdentity401
  | ConfirmMediaServerIdentity403
  | ConfirmMediaServerIdentity404
  | ConfirmMediaServerIdentity409
  | ConfirmMediaServerIdentity422
  | ConfirmMediaServerIdentity500
  | ConfirmMediaServerIdentity502
>;
export type ConfirmMediaServerIdentityMutationVariables = { id: string };

/**
 * @summary Accept the server that responds at a Media Server's address now as that Media Server
 */
export const useConfirmMediaServerIdentity = <
  TError = ErrorType<
    | ConfirmMediaServerIdentity401
    | ConfirmMediaServerIdentity403
    | ConfirmMediaServerIdentity404
    | ConfirmMediaServerIdentity409
    | ConfirmMediaServerIdentity422
    | ConfirmMediaServerIdentity500
    | ConfirmMediaServerIdentity502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof confirmMediaServerIdentity>>,
      TError,
      ConfirmMediaServerIdentityMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof confirmMediaServerIdentity>>,
  TError,
  ConfirmMediaServerIdentityMutationVariables,
  TContext
> => {
  return useMutation(getConfirmMediaServerIdentityMutationOptions(options), queryClient);
};
export const getListMediaServerCredentialHoldersUrl = (id: string) => {
  return `/api/v1/media-servers/${id}/credential-holders`;
};

/**
 * @summary List every Node that fetched a Media Server's credential
 */
export const listMediaServerCredentialHolders = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<HoldersOutputBody> => {
  return apiFetch<HoldersOutputBody>(getListMediaServerCredentialHoldersUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getListMediaServerCredentialHoldersQueryKey = (id: string) => {
  return [`/api/v1/media-servers/${id}/credential-holders`] as const;
};

export const getListMediaServerCredentialHoldersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
  TError = ErrorType<
    | ListMediaServerCredentialHolders401
    | ListMediaServerCredentialHolders403
    | ListMediaServerCredentialHolders404
    | ListMediaServerCredentialHolders409
    | ListMediaServerCredentialHolders422
    | ListMediaServerCredentialHolders500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<
        Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
        TError,
        TData
      >
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMediaServerCredentialHoldersQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMediaServerCredentialHolders>>> = ({
    signal,
  }) => listMediaServerCredentialHolders(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMediaServerCredentialHoldersSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listMediaServerCredentialHolders>>
>;
export type ListMediaServerCredentialHoldersSuspenseQueryError = ErrorType<
  | ListMediaServerCredentialHolders401
  | ListMediaServerCredentialHolders403
  | ListMediaServerCredentialHolders404
  | ListMediaServerCredentialHolders409
  | ListMediaServerCredentialHolders422
  | ListMediaServerCredentialHolders500
>;

export function useListMediaServerCredentialHoldersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
  TError = ErrorType<
    | ListMediaServerCredentialHolders401
    | ListMediaServerCredentialHolders403
    | ListMediaServerCredentialHolders404
    | ListMediaServerCredentialHolders409
    | ListMediaServerCredentialHolders422
    | ListMediaServerCredentialHolders500
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<
        Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
        TError,
        TData
      >
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerCredentialHoldersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
  TError = ErrorType<
    | ListMediaServerCredentialHolders401
    | ListMediaServerCredentialHolders403
    | ListMediaServerCredentialHolders404
    | ListMediaServerCredentialHolders409
    | ListMediaServerCredentialHolders422
    | ListMediaServerCredentialHolders500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<
        Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
        TError,
        TData
      >
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerCredentialHoldersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
  TError = ErrorType<
    | ListMediaServerCredentialHolders401
    | ListMediaServerCredentialHolders403
    | ListMediaServerCredentialHolders404
    | ListMediaServerCredentialHolders409
    | ListMediaServerCredentialHolders422
    | ListMediaServerCredentialHolders500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<
        Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
        TError,
        TData
      >
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List every Node that fetched a Media Server's credential
 */

export function useListMediaServerCredentialHoldersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
  TError = ErrorType<
    | ListMediaServerCredentialHolders401
    | ListMediaServerCredentialHolders403
    | ListMediaServerCredentialHolders404
    | ListMediaServerCredentialHolders409
    | ListMediaServerCredentialHolders422
    | ListMediaServerCredentialHolders500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<
        Awaited<ReturnType<typeof listMediaServerCredentialHolders>>,
        TError,
        TData
      >
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMediaServerCredentialHoldersSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getListMediaServerLibrariesUrl = (id: string) => {
  return `/api/v1/media-servers/${id}/libraries`;
};

/**
 * @summary List a Media Server's library folders and every library root's Path mapping, pre-filled where none is saved
 */
export const listMediaServerLibraries = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversLibraries> => {
  return apiFetch<MediaserversLibraries>(getListMediaServerLibrariesUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getListMediaServerLibrariesQueryKey = (id: string) => {
  return [`/api/v1/media-servers/${id}/libraries`] as const;
};

export const getListMediaServerLibrariesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMediaServerLibraries>>,
  TError = ErrorType<
    | ListMediaServerLibraries401
    | ListMediaServerLibraries403
    | ListMediaServerLibraries404
    | ListMediaServerLibraries409
    | ListMediaServerLibraries422
    | ListMediaServerLibraries500
    | ListMediaServerLibraries502
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerLibraries>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMediaServerLibrariesQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMediaServerLibraries>>> = ({
    signal,
  }) => listMediaServerLibraries(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listMediaServerLibraries>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMediaServerLibrariesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listMediaServerLibraries>>
>;
export type ListMediaServerLibrariesSuspenseQueryError = ErrorType<
  | ListMediaServerLibraries401
  | ListMediaServerLibraries403
  | ListMediaServerLibraries404
  | ListMediaServerLibraries409
  | ListMediaServerLibraries422
  | ListMediaServerLibraries500
  | ListMediaServerLibraries502
>;

export function useListMediaServerLibrariesSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerLibraries>>,
  TError = ErrorType<
    | ListMediaServerLibraries401
    | ListMediaServerLibraries403
    | ListMediaServerLibraries404
    | ListMediaServerLibraries409
    | ListMediaServerLibraries422
    | ListMediaServerLibraries500
    | ListMediaServerLibraries502
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerLibraries>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerLibrariesSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerLibraries>>,
  TError = ErrorType<
    | ListMediaServerLibraries401
    | ListMediaServerLibraries403
    | ListMediaServerLibraries404
    | ListMediaServerLibraries409
    | ListMediaServerLibraries422
    | ListMediaServerLibraries500
    | ListMediaServerLibraries502
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerLibraries>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerLibrariesSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerLibraries>>,
  TError = ErrorType<
    | ListMediaServerLibraries401
    | ListMediaServerLibraries403
    | ListMediaServerLibraries404
    | ListMediaServerLibraries409
    | ListMediaServerLibraries422
    | ListMediaServerLibraries500
    | ListMediaServerLibraries502
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerLibraries>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List a Media Server's library folders and every library root's Path mapping, pre-filled where none is saved
 */

export function useListMediaServerLibrariesSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerLibraries>>,
  TError = ErrorType<
    | ListMediaServerLibraries401
    | ListMediaServerLibraries403
    | ListMediaServerLibraries404
    | ListMediaServerLibraries409
    | ListMediaServerLibraries422
    | ListMediaServerLibraries500
    | ListMediaServerLibraries502
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerLibraries>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMediaServerLibrariesSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getSaveMediaServerMappingsUrl = (id: string) => {
  return `/api/v1/media-servers/${id}/mappings`;
};

/**
 * @summary Save a Media Server's Path mappings once each shown folder sits inside one of its library folders
 */
export const saveMediaServerMappings = async (
  id: string,
  mediaserversMappingsBody: MediaserversMappingsBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MappingsOutputBody> => {
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
  return apiFetch<MappingsOutputBody>(getSaveMediaServerMappingsUrl(id), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(mediaserversMappingsBody),
  });
};

export const getSaveMediaServerMappingsMutationKey = () => ["saveMediaServerMappings"] as const;

export const getSaveMediaServerMappingsMutationOptions = <
  TError = ErrorType<
    | SaveMediaServerMappings401
    | SaveMediaServerMappings403
    | SaveMediaServerMappings404
    | SaveMediaServerMappings409
    | SaveMediaServerMappings422
    | SaveMediaServerMappings500
    | SaveMediaServerMappings502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof saveMediaServerMappings>>,
    TError,
    SaveMediaServerMappingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof saveMediaServerMappings>>,
  TError,
  SaveMediaServerMappingsMutationVariables,
  TContext
> => {
  const mutationKey = getSaveMediaServerMappingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof saveMediaServerMappings>>,
    SaveMediaServerMappingsMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return saveMediaServerMappings(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SaveMediaServerMappingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof saveMediaServerMappings>>
>;
export type SaveMediaServerMappingsMutationBody = MediaserversMappingsBody;
export type SaveMediaServerMappingsMutationError = ErrorType<
  | SaveMediaServerMappings401
  | SaveMediaServerMappings403
  | SaveMediaServerMappings404
  | SaveMediaServerMappings409
  | SaveMediaServerMappings422
  | SaveMediaServerMappings500
  | SaveMediaServerMappings502
>;
export type SaveMediaServerMappingsMutationVariables = {
  id: string;
  data: MediaserversMappingsBody;
};

/**
 * @summary Save a Media Server's Path mappings once each shown folder sits inside one of its library folders
 */
export const useSaveMediaServerMappings = <
  TError = ErrorType<
    | SaveMediaServerMappings401
    | SaveMediaServerMappings403
    | SaveMediaServerMappings404
    | SaveMediaServerMappings409
    | SaveMediaServerMappings422
    | SaveMediaServerMappings500
    | SaveMediaServerMappings502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof saveMediaServerMappings>>,
      TError,
      SaveMediaServerMappingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof saveMediaServerMappings>>,
  TError,
  SaveMediaServerMappingsMutationVariables,
  TContext
> => {
  return useMutation(getSaveMediaServerMappingsMutationOptions(options), queryClient);
};
export const getRefreshMediaServerUrl = (id: string) => {
  return `/api/v1/media-servers/${id}/refresh`;
};

/**
 * @summary Queue a whole library refresh of every library a Media Server's confirmed Path mappings lead into
 */
export const refreshMediaServer = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getRefreshMediaServerUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getRefreshMediaServerMutationKey = () => ["refreshMediaServer"] as const;

export const getRefreshMediaServerMutationOptions = <
  TError = ErrorType<
    | RefreshMediaServer401
    | RefreshMediaServer403
    | RefreshMediaServer404
    | RefreshMediaServer409
    | RefreshMediaServer422
    | RefreshMediaServer500
    | RefreshMediaServer501
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof refreshMediaServer>>,
    TError,
    RefreshMediaServerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof refreshMediaServer>>,
  TError,
  RefreshMediaServerMutationVariables,
  TContext
> => {
  const mutationKey = getRefreshMediaServerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof refreshMediaServer>>,
    RefreshMediaServerMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return refreshMediaServer(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RefreshMediaServerMutationResult = NonNullable<
  Awaited<ReturnType<typeof refreshMediaServer>>
>;

export type RefreshMediaServerMutationError = ErrorType<
  | RefreshMediaServer401
  | RefreshMediaServer403
  | RefreshMediaServer404
  | RefreshMediaServer409
  | RefreshMediaServer422
  | RefreshMediaServer500
  | RefreshMediaServer501
>;
export type RefreshMediaServerMutationVariables = { id: string };

/**
 * @summary Queue a whole library refresh of every library a Media Server's confirmed Path mappings lead into
 */
export const useRefreshMediaServer = <
  TError = ErrorType<
    | RefreshMediaServer401
    | RefreshMediaServer403
    | RefreshMediaServer404
    | RefreshMediaServer409
    | RefreshMediaServer422
    | RefreshMediaServer500
    | RefreshMediaServer501
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof refreshMediaServer>>,
      TError,
      RefreshMediaServerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof refreshMediaServer>>,
  TError,
  RefreshMediaServerMutationVariables,
  TContext
> => {
  return useMutation(getRefreshMediaServerMutationOptions(options), queryClient);
};
export const getListMediaServerSessionsUrl = (id: string) => {
  return `/api/v1/media-servers/${id}/sessions`;
};

/**
 * @summary List what plays on a Media Server now, as its reaching Node last saw it
 */
export const listMediaServerSessions = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversSessions> => {
  return apiFetch<MediaserversSessions>(getListMediaServerSessionsUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getListMediaServerSessionsQueryKey = (id: string) => {
  return [`/api/v1/media-servers/${id}/sessions`] as const;
};

export const getListMediaServerSessionsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMediaServerSessions>>,
  TError = ErrorType<
    | ListMediaServerSessions401
    | ListMediaServerSessions403
    | ListMediaServerSessions404
    | ListMediaServerSessions409
    | ListMediaServerSessions422
    | ListMediaServerSessions500
    | ListMediaServerSessions501
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMediaServerSessionsQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMediaServerSessions>>> = ({
    signal,
  }) => listMediaServerSessions(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listMediaServerSessions>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMediaServerSessionsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listMediaServerSessions>>
>;
export type ListMediaServerSessionsSuspenseQueryError = ErrorType<
  | ListMediaServerSessions401
  | ListMediaServerSessions403
  | ListMediaServerSessions404
  | ListMediaServerSessions409
  | ListMediaServerSessions422
  | ListMediaServerSessions500
  | ListMediaServerSessions501
>;

export function useListMediaServerSessionsSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerSessions>>,
  TError = ErrorType<
    | ListMediaServerSessions401
    | ListMediaServerSessions403
    | ListMediaServerSessions404
    | ListMediaServerSessions409
    | ListMediaServerSessions422
    | ListMediaServerSessions500
    | ListMediaServerSessions501
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerSessionsSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerSessions>>,
  TError = ErrorType<
    | ListMediaServerSessions401
    | ListMediaServerSessions403
    | ListMediaServerSessions404
    | ListMediaServerSessions409
    | ListMediaServerSessions422
    | ListMediaServerSessions500
    | ListMediaServerSessions501
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerSessionsSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerSessions>>,
  TError = ErrorType<
    | ListMediaServerSessions401
    | ListMediaServerSessions403
    | ListMediaServerSessions404
    | ListMediaServerSessions409
    | ListMediaServerSessions422
    | ListMediaServerSessions500
    | ListMediaServerSessions501
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List what plays on a Media Server now, as its reaching Node last saw it
 */

export function useListMediaServerSessionsSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerSessions>>,
  TError = ErrorType<
    | ListMediaServerSessions401
    | ListMediaServerSessions403
    | ListMediaServerSessions404
    | ListMediaServerSessions409
    | ListMediaServerSessions422
    | ListMediaServerSessions500
    | ListMediaServerSessions501
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMediaServerSessionsSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getTestMediaServerUrl = (id: string) => {
  return `/api/v1/media-servers/${id}/test`;
};

/**
 * @summary Test a Media Server from its reaching Node at its address and from the Controller at its fallback address
 */
export const testMediaServer = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversTested> => {
  return apiFetch<MediaserversTested>(getTestMediaServerUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getTestMediaServerMutationKey = () => ["testMediaServer"] as const;

export const getTestMediaServerMutationOptions = <
  TError = ErrorType<
    | TestMediaServer401
    | TestMediaServer403
    | TestMediaServer404
    | TestMediaServer409
    | TestMediaServer422
    | TestMediaServer500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof testMediaServer>>,
    TError,
    TestMediaServerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof testMediaServer>>,
  TError,
  TestMediaServerMutationVariables,
  TContext
> => {
  const mutationKey = getTestMediaServerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof testMediaServer>>,
    TestMediaServerMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return testMediaServer(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type TestMediaServerMutationResult = NonNullable<
  Awaited<ReturnType<typeof testMediaServer>>
>;

export type TestMediaServerMutationError = ErrorType<
  | TestMediaServer401
  | TestMediaServer403
  | TestMediaServer404
  | TestMediaServer409
  | TestMediaServer422
  | TestMediaServer500
>;
export type TestMediaServerMutationVariables = { id: string };

/**
 * @summary Test a Media Server from its reaching Node at its address and from the Controller at its fallback address
 */
export const useTestMediaServer = <
  TError = ErrorType<
    | TestMediaServer401
    | TestMediaServer403
    | TestMediaServer404
    | TestMediaServer409
    | TestMediaServer422
    | TestMediaServer500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof testMediaServer>>,
      TError,
      TestMediaServerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof testMediaServer>>,
  TError,
  TestMediaServerMutationVariables,
  TContext
> => {
  return useMutation(getTestMediaServerMutationOptions(options), queryClient);
};
