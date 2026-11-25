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
  CheckMediaServerAccess401,
  CheckMediaServerAccess403,
  CheckMediaServerAccess404,
  CheckMediaServerAccess409,
  CheckMediaServerAccess422,
  CheckMediaServerAccess500,
  CheckMediaServerAccess501,
  CheckMediaServerAccess502,
  CheckMyMediaServerAccess401,
  CheckMyMediaServerAccess403,
  CheckMyMediaServerAccess404,
  CheckMyMediaServerAccess409,
  CheckMyMediaServerAccess422,
  CheckMyMediaServerAccess500,
  CheckMyMediaServerAccess501,
  CheckMyMediaServerAccess502,
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
  GetMemberPlexSignIn401,
  GetMemberPlexSignIn403,
  GetMemberPlexSignIn404,
  GetMemberPlexSignIn409,
  GetMemberPlexSignIn422,
  GetMemberPlexSignIn500,
  GetMemberPlexSignIn501,
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
  LinkMediaServerUser401,
  LinkMediaServerUser403,
  LinkMediaServerUser404,
  LinkMediaServerUser409,
  LinkMediaServerUser422,
  LinkMediaServerUser500,
  LinkMediaServerUser501,
  LinkMediaServerUser502,
  LinkUserInputBody,
  ListMediaServerAccess401,
  ListMediaServerAccess403,
  ListMediaServerAccess404,
  ListMediaServerAccess409,
  ListMediaServerAccess422,
  ListMediaServerAccess500,
  ListMediaServerAccess501,
  ListMediaServerAccessParams,
  ListMediaServerChanges401,
  ListMediaServerChanges403,
  ListMediaServerChanges404,
  ListMediaServerChanges409,
  ListMediaServerChanges422,
  ListMediaServerChanges500,
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
  ListMediaServerUsers401,
  ListMediaServerUsers403,
  ListMediaServerUsers404,
  ListMediaServerUsers409,
  ListMediaServerUsers422,
  ListMediaServerUsers500,
  ListMediaServerUsers501,
  ListMediaServerUsers502,
  ListMediaServers401,
  ListMediaServers403,
  ListMediaServers409,
  ListMediaServers500,
  ListMyMediaServerAccess401,
  ListMyMediaServerAccess403,
  ListMyMediaServerAccess409,
  ListMyMediaServerAccess500,
  ListMyMediaServerAccess501,
  ListPlexServers401,
  ListPlexServers403,
  ListPlexServers409,
  ListPlexServers422,
  ListPlexServers500,
  ListPlexServers502,
  MakeAccountInputBody,
  MakeMediaServerAccount401,
  MakeMediaServerAccount403,
  MakeMediaServerAccount404,
  MakeMediaServerAccount409,
  MakeMediaServerAccount422,
  MakeMediaServerAccount500,
  MakeMediaServerAccount501,
  MakeMediaServerAccount502,
  MappingsOutputBody,
  MarkMediaServerAccessAdded401,
  MarkMediaServerAccessAdded403,
  MarkMediaServerAccessAdded404,
  MarkMediaServerAccessAdded409,
  MarkMediaServerAccessAdded422,
  MarkMediaServerAccessAdded500,
  MarkMediaServerAccessAdded501,
  MediaserversAccessList,
  MediaserversAccountView,
  MediaserversActed,
  MediaserversChanges,
  MediaserversDraftBody,
  MediaserversLibraries,
  MediaserversMappingsBody,
  MediaserversMemberPin,
  MediaserversMemberPinState,
  MediaserversMyAccess,
  MediaserversMyAccessRow,
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
  OfferInputBody,
  OfferMediaServer401,
  OfferMediaServer403,
  OfferMediaServer404,
  OfferMediaServer409,
  OfferMediaServer422,
  OfferMediaServer500,
  OfferMediaServer501,
  OfferMediaServer502,
  PlexServersOutputBody,
  RefreshMediaServer401,
  RefreshMediaServer403,
  RefreshMediaServer404,
  RefreshMediaServer409,
  RefreshMediaServer422,
  RefreshMediaServer500,
  RemoveMediaServer401,
  RemoveMediaServer403,
  RemoveMediaServer404,
  RemoveMediaServer409,
  RemoveMediaServer422,
  RemoveMediaServer500,
  RemoveMediaServerAccess401,
  RemoveMediaServerAccess403,
  RemoveMediaServerAccess404,
  RemoveMediaServerAccess409,
  RemoveMediaServerAccess422,
  RemoveMediaServerAccess500,
  RemoveMediaServerAccess501,
  ReplacePlexAccount401,
  ReplacePlexAccount403,
  ReplacePlexAccount404,
  ReplacePlexAccount409,
  ReplacePlexAccount422,
  ReplacePlexAccount500,
  ReplacePlexAccount501,
  SaveMediaServerMappings401,
  SaveMediaServerMappings403,
  SaveMediaServerMappings404,
  SaveMediaServerMappings409,
  SaveMediaServerMappings422,
  SaveMediaServerMappings500,
  SaveMediaServerMappings502,
  ServerUsersOutputBody,
  ServersOutputBody,
  SettingsInputBody,
  ShareAccessInputBody,
  ShareMediaServerAccess401,
  ShareMediaServerAccess403,
  ShareMediaServerAccess404,
  ShareMediaServerAccess409,
  ShareMediaServerAccess422,
  ShareMediaServerAccess500,
  ShareMediaServerAccess501,
  ShareMediaServerAccess502,
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
  StartMemberPlexSignIn401,
  StartMemberPlexSignIn403,
  StartMemberPlexSignIn409,
  StartMemberPlexSignIn500,
  StartMemberPlexSignIn501,
  StartMemberPlexSignIn502,
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
  UpdateMediaServer501,
  UpdateMediaServer502,
  UpdateMediaServerSettings401,
  UpdateMediaServerSettings403,
  UpdateMediaServerSettings409,
  UpdateMediaServerSettings422,
  UpdateMediaServerSettings500,
  UseAnotherPlexAccount401,
  UseAnotherPlexAccount403,
  UseAnotherPlexAccount409,
  UseAnotherPlexAccount500,
  UseAnotherPlexAccount501,
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

export const getListMyMediaServerAccessUrl = () => {
  return `/api/v1/account/media-servers`;
};

/**
 * @summary List the Media Servers the Member was offered, with what each needs next and how to watch on it
 */
export const listMyMediaServerAccess = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversMyAccess> => {
  return apiFetch<MediaserversMyAccess>(getListMyMediaServerAccessUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListMyMediaServerAccessQueryKey = () => {
  return [`/api/v1/account/media-servers`] as const;
};

export const getListMyMediaServerAccessSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMyMediaServerAccess>>,
  TError = ErrorType<
    | ListMyMediaServerAccess401
    | ListMyMediaServerAccess403
    | ListMyMediaServerAccess409
    | ListMyMediaServerAccess500
    | ListMyMediaServerAccess501
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMyMediaServerAccess>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMyMediaServerAccessQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMyMediaServerAccess>>> = ({
    signal,
  }) => listMyMediaServerAccess({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listMyMediaServerAccess>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMyMediaServerAccessSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listMyMediaServerAccess>>
>;
export type ListMyMediaServerAccessSuspenseQueryError = ErrorType<
  | ListMyMediaServerAccess401
  | ListMyMediaServerAccess403
  | ListMyMediaServerAccess409
  | ListMyMediaServerAccess500
  | ListMyMediaServerAccess501
>;

export function useListMyMediaServerAccessSuspense<
  TData = Awaited<ReturnType<typeof listMyMediaServerAccess>>,
  TError = ErrorType<
    | ListMyMediaServerAccess401
    | ListMyMediaServerAccess403
    | ListMyMediaServerAccess409
    | ListMyMediaServerAccess500
    | ListMyMediaServerAccess501
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMyMediaServerAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMyMediaServerAccessSuspense<
  TData = Awaited<ReturnType<typeof listMyMediaServerAccess>>,
  TError = ErrorType<
    | ListMyMediaServerAccess401
    | ListMyMediaServerAccess403
    | ListMyMediaServerAccess409
    | ListMyMediaServerAccess500
    | ListMyMediaServerAccess501
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMyMediaServerAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMyMediaServerAccessSuspense<
  TData = Awaited<ReturnType<typeof listMyMediaServerAccess>>,
  TError = ErrorType<
    | ListMyMediaServerAccess401
    | ListMyMediaServerAccess403
    | ListMyMediaServerAccess409
    | ListMyMediaServerAccess500
    | ListMyMediaServerAccess501
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMyMediaServerAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Media Servers the Member was offered, with what each needs next and how to watch on it
 */

export function useListMyMediaServerAccessSuspense<
  TData = Awaited<ReturnType<typeof listMyMediaServerAccess>>,
  TError = ErrorType<
    | ListMyMediaServerAccess401
    | ListMyMediaServerAccess403
    | ListMyMediaServerAccess409
    | ListMyMediaServerAccess500
    | ListMyMediaServerAccess501
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMyMediaServerAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMyMediaServerAccessSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUseAnotherPlexAccountUrl = () => {
  return `/api/v1/account/media-servers/plex`;
};

/**
 * @summary Forget the Plex account the Member signed in to Plex as, while no server is shared with it yet
 */
export const useAnotherPlexAccount = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getUseAnotherPlexAccountUrl(), {
    ...options,
    method: "DELETE",
  });
};

export const getUseAnotherPlexAccountMutationKey = () => ["useAnotherPlexAccount"] as const;

export const getUseAnotherPlexAccountMutationOptions = <
  TError = ErrorType<
    | UseAnotherPlexAccount401
    | UseAnotherPlexAccount403
    | UseAnotherPlexAccount409
    | UseAnotherPlexAccount500
    | UseAnotherPlexAccount501
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof useAnotherPlexAccount>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof useAnotherPlexAccount>>,
  TError,
  void,
  TContext
> => {
  const mutationKey = getUseAnotherPlexAccountMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof useAnotherPlexAccount>>,
    void
  > = () => {
    return useAnotherPlexAccount(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UseAnotherPlexAccountMutationResult = NonNullable<
  Awaited<ReturnType<typeof useAnotherPlexAccount>>
>;

export type UseAnotherPlexAccountMutationError = ErrorType<
  | UseAnotherPlexAccount401
  | UseAnotherPlexAccount403
  | UseAnotherPlexAccount409
  | UseAnotherPlexAccount500
  | UseAnotherPlexAccount501
>;

/**
 * @summary Forget the Plex account the Member signed in to Plex as, while no server is shared with it yet
 */
export const useUseAnotherPlexAccount = <
  TError = ErrorType<
    | UseAnotherPlexAccount401
    | UseAnotherPlexAccount403
    | UseAnotherPlexAccount409
    | UseAnotherPlexAccount500
    | UseAnotherPlexAccount501
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof useAnotherPlexAccount>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof useAnotherPlexAccount>>, TError, void, TContext> => {
  return useMutation(getUseAnotherPlexAccountMutationOptions(options), queryClient);
};
export const getStartMemberPlexSignInUrl = () => {
  return `/api/v1/account/media-servers/plex/pin`;
};

/**
 * @summary Start the Member's Plex sign-in and get the plex.tv page they sign in on
 */
export const startMemberPlexSignIn = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversMemberPin> => {
  return apiFetch<MediaserversMemberPin>(getStartMemberPlexSignInUrl(), {
    ...options,
    method: "POST",
  });
};

export const getStartMemberPlexSignInMutationKey = () => ["startMemberPlexSignIn"] as const;

export const getStartMemberPlexSignInMutationOptions = <
  TError = ErrorType<
    | StartMemberPlexSignIn401
    | StartMemberPlexSignIn403
    | StartMemberPlexSignIn409
    | StartMemberPlexSignIn500
    | StartMemberPlexSignIn501
    | StartMemberPlexSignIn502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof startMemberPlexSignIn>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof startMemberPlexSignIn>>,
  TError,
  void,
  TContext
> => {
  const mutationKey = getStartMemberPlexSignInMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof startMemberPlexSignIn>>,
    void
  > = () => {
    return startMemberPlexSignIn(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type StartMemberPlexSignInMutationResult = NonNullable<
  Awaited<ReturnType<typeof startMemberPlexSignIn>>
>;

export type StartMemberPlexSignInMutationError = ErrorType<
  | StartMemberPlexSignIn401
  | StartMemberPlexSignIn403
  | StartMemberPlexSignIn409
  | StartMemberPlexSignIn500
  | StartMemberPlexSignIn501
  | StartMemberPlexSignIn502
>;

/**
 * @summary Start the Member's Plex sign-in and get the plex.tv page they sign in on
 */
export const useStartMemberPlexSignIn = <
  TError = ErrorType<
    | StartMemberPlexSignIn401
    | StartMemberPlexSignIn403
    | StartMemberPlexSignIn409
    | StartMemberPlexSignIn500
    | StartMemberPlexSignIn501
    | StartMemberPlexSignIn502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof startMemberPlexSignIn>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof startMemberPlexSignIn>>, TError, void, TContext> => {
  return useMutation(getStartMemberPlexSignInMutationOptions(options), queryClient);
};
export const getGetMemberPlexSignInUrl = (id: number) => {
  return `/api/v1/account/media-servers/plex/pin/${id}`;
};

/**
 * @summary Get whether the Member's Plex sign-in is still waiting, has expired, signed in or was refused
 */
export const getMemberPlexSignIn = async (
  id: number,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversMemberPinState> => {
  return apiFetch<MediaserversMemberPinState>(getGetMemberPlexSignInUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetMemberPlexSignInQueryKey = (id: number) => {
  return [`/api/v1/account/media-servers/plex/pin/${id}`] as const;
};

export const getGetMemberPlexSignInSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getMemberPlexSignIn>>,
  TError = ErrorType<
    | GetMemberPlexSignIn401
    | GetMemberPlexSignIn403
    | GetMemberPlexSignIn404
    | GetMemberPlexSignIn409
    | GetMemberPlexSignIn422
    | GetMemberPlexSignIn500
    | GetMemberPlexSignIn501
  >,
>(
  id: number,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMemberPlexSignIn>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetMemberPlexSignInQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getMemberPlexSignIn>>> = ({ signal }) =>
    getMemberPlexSignIn(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMemberPlexSignIn>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetMemberPlexSignInSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getMemberPlexSignIn>>
>;
export type GetMemberPlexSignInSuspenseQueryError = ErrorType<
  | GetMemberPlexSignIn401
  | GetMemberPlexSignIn403
  | GetMemberPlexSignIn404
  | GetMemberPlexSignIn409
  | GetMemberPlexSignIn422
  | GetMemberPlexSignIn500
  | GetMemberPlexSignIn501
>;

export function useGetMemberPlexSignInSuspense<
  TData = Awaited<ReturnType<typeof getMemberPlexSignIn>>,
  TError = ErrorType<
    | GetMemberPlexSignIn401
    | GetMemberPlexSignIn403
    | GetMemberPlexSignIn404
    | GetMemberPlexSignIn409
    | GetMemberPlexSignIn422
    | GetMemberPlexSignIn500
    | GetMemberPlexSignIn501
  >,
>(
  id: number,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMemberPlexSignIn>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMemberPlexSignInSuspense<
  TData = Awaited<ReturnType<typeof getMemberPlexSignIn>>,
  TError = ErrorType<
    | GetMemberPlexSignIn401
    | GetMemberPlexSignIn403
    | GetMemberPlexSignIn404
    | GetMemberPlexSignIn409
    | GetMemberPlexSignIn422
    | GetMemberPlexSignIn500
    | GetMemberPlexSignIn501
  >,
>(
  id: number,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMemberPlexSignIn>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMemberPlexSignInSuspense<
  TData = Awaited<ReturnType<typeof getMemberPlexSignIn>>,
  TError = ErrorType<
    | GetMemberPlexSignIn401
    | GetMemberPlexSignIn403
    | GetMemberPlexSignIn404
    | GetMemberPlexSignIn409
    | GetMemberPlexSignIn422
    | GetMemberPlexSignIn500
    | GetMemberPlexSignIn501
  >,
>(
  id: number,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMemberPlexSignIn>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get whether the Member's Plex sign-in is still waiting, has expired, signed in or was refused
 */

export function useGetMemberPlexSignInSuspense<
  TData = Awaited<ReturnType<typeof getMemberPlexSignIn>>,
  TError = ErrorType<
    | GetMemberPlexSignIn401
    | GetMemberPlexSignIn403
    | GetMemberPlexSignIn404
    | GetMemberPlexSignIn409
    | GetMemberPlexSignIn422
    | GetMemberPlexSignIn500
    | GetMemberPlexSignIn501
  >,
>(
  id: number,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMemberPlexSignIn>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetMemberPlexSignInSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getMakeMediaServerAccountUrl = (accessId: string) => {
  return `/api/v1/account/media-servers/${accessId}/account`;
};

/**
 * @summary Make the Member's own user on a Jellyfin or Emby server they were offered
 */
export const makeMediaServerAccount = async (
  accessId: string,
  makeAccountInputBody: MakeAccountInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversMyAccessRow> => {
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
  return apiFetch<MediaserversMyAccessRow>(getMakeMediaServerAccountUrl(accessId), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(makeAccountInputBody),
  });
};

export const getMakeMediaServerAccountMutationKey = () => ["makeMediaServerAccount"] as const;

export const getMakeMediaServerAccountMutationOptions = <
  TError = ErrorType<
    | MakeMediaServerAccount401
    | MakeMediaServerAccount403
    | MakeMediaServerAccount404
    | MakeMediaServerAccount409
    | MakeMediaServerAccount422
    | MakeMediaServerAccount500
    | MakeMediaServerAccount501
    | MakeMediaServerAccount502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof makeMediaServerAccount>>,
    TError,
    MakeMediaServerAccountMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof makeMediaServerAccount>>,
  TError,
  MakeMediaServerAccountMutationVariables,
  TContext
> => {
  const mutationKey = getMakeMediaServerAccountMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof makeMediaServerAccount>>,
    MakeMediaServerAccountMutationVariables
  > = (props) => {
    const { accessId, data } = props ?? {};

    return makeMediaServerAccount(accessId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MakeMediaServerAccountMutationResult = NonNullable<
  Awaited<ReturnType<typeof makeMediaServerAccount>>
>;
export type MakeMediaServerAccountMutationBody = MakeAccountInputBody;
export type MakeMediaServerAccountMutationError = ErrorType<
  | MakeMediaServerAccount401
  | MakeMediaServerAccount403
  | MakeMediaServerAccount404
  | MakeMediaServerAccount409
  | MakeMediaServerAccount422
  | MakeMediaServerAccount500
  | MakeMediaServerAccount501
  | MakeMediaServerAccount502
>;
export type MakeMediaServerAccountMutationVariables = {
  accessId: string;
  data: MakeAccountInputBody;
};

/**
 * @summary Make the Member's own user on a Jellyfin or Emby server they were offered
 */
export const useMakeMediaServerAccount = <
  TError = ErrorType<
    | MakeMediaServerAccount401
    | MakeMediaServerAccount403
    | MakeMediaServerAccount404
    | MakeMediaServerAccount409
    | MakeMediaServerAccount422
    | MakeMediaServerAccount500
    | MakeMediaServerAccount501
    | MakeMediaServerAccount502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof makeMediaServerAccount>>,
      TError,
      MakeMediaServerAccountMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof makeMediaServerAccount>>,
  TError,
  MakeMediaServerAccountMutationVariables,
  TContext
> => {
  return useMutation(getMakeMediaServerAccountMutationOptions(options), queryClient);
};
export const getCheckMyMediaServerAccessUrl = (accessId: string) => {
  return `/api/v1/account/media-servers/${accessId}/check`;
};

/**
 * @summary Check whether the Member's Plex share is accepted, accepting it when SlipStream can
 */
export const checkMyMediaServerAccess = async (
  accessId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversMyAccessRow> => {
  return apiFetch<MediaserversMyAccessRow>(getCheckMyMediaServerAccessUrl(accessId), {
    ...options,
    method: "POST",
  });
};

export const getCheckMyMediaServerAccessMutationKey = () => ["checkMyMediaServerAccess"] as const;

export const getCheckMyMediaServerAccessMutationOptions = <
  TError = ErrorType<
    | CheckMyMediaServerAccess401
    | CheckMyMediaServerAccess403
    | CheckMyMediaServerAccess404
    | CheckMyMediaServerAccess409
    | CheckMyMediaServerAccess422
    | CheckMyMediaServerAccess500
    | CheckMyMediaServerAccess501
    | CheckMyMediaServerAccess502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof checkMyMediaServerAccess>>,
    TError,
    CheckMyMediaServerAccessMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof checkMyMediaServerAccess>>,
  TError,
  CheckMyMediaServerAccessMutationVariables,
  TContext
> => {
  const mutationKey = getCheckMyMediaServerAccessMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof checkMyMediaServerAccess>>,
    CheckMyMediaServerAccessMutationVariables
  > = (props) => {
    const { accessId } = props ?? {};

    return checkMyMediaServerAccess(accessId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CheckMyMediaServerAccessMutationResult = NonNullable<
  Awaited<ReturnType<typeof checkMyMediaServerAccess>>
>;

export type CheckMyMediaServerAccessMutationError = ErrorType<
  | CheckMyMediaServerAccess401
  | CheckMyMediaServerAccess403
  | CheckMyMediaServerAccess404
  | CheckMyMediaServerAccess409
  | CheckMyMediaServerAccess422
  | CheckMyMediaServerAccess500
  | CheckMyMediaServerAccess501
  | CheckMyMediaServerAccess502
>;
export type CheckMyMediaServerAccessMutationVariables = { accessId: string };

/**
 * @summary Check whether the Member's Plex share is accepted, accepting it when SlipStream can
 */
export const useCheckMyMediaServerAccess = <
  TError = ErrorType<
    | CheckMyMediaServerAccess401
    | CheckMyMediaServerAccess403
    | CheckMyMediaServerAccess404
    | CheckMyMediaServerAccess409
    | CheckMyMediaServerAccess422
    | CheckMyMediaServerAccess500
    | CheckMyMediaServerAccess501
    | CheckMyMediaServerAccess502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof checkMyMediaServerAccess>>,
      TError,
      CheckMyMediaServerAccessMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof checkMyMediaServerAccess>>,
  TError,
  CheckMyMediaServerAccessMutationVariables,
  TContext
> => {
  return useMutation(getCheckMyMediaServerAccessMutationOptions(options), queryClient);
};
export const getOfferMediaServerUrl = (id: string, serverId: string) => {
  return `/api/v1/accounts/${id}/media-servers/${serverId}`;
};

/**
 * @summary Offer a Member access to a Media Server, or offer it again after the access was removed
 */
export const offerMediaServer = async (
  id: string,
  serverId: string,
  offerInputBody: OfferInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversActed> => {
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
  return apiFetch<MediaserversActed>(getOfferMediaServerUrl(id, serverId), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(offerInputBody),
  });
};

export const getOfferMediaServerMutationKey = () => ["offerMediaServer"] as const;

export const getOfferMediaServerMutationOptions = <
  TError = ErrorType<
    | OfferMediaServer401
    | OfferMediaServer403
    | OfferMediaServer404
    | OfferMediaServer409
    | OfferMediaServer422
    | OfferMediaServer500
    | OfferMediaServer501
    | OfferMediaServer502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof offerMediaServer>>,
    TError,
    OfferMediaServerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof offerMediaServer>>,
  TError,
  OfferMediaServerMutationVariables,
  TContext
> => {
  const mutationKey = getOfferMediaServerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof offerMediaServer>>,
    OfferMediaServerMutationVariables
  > = (props) => {
    const { id, serverId, data } = props ?? {};

    return offerMediaServer(id, serverId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type OfferMediaServerMutationResult = NonNullable<
  Awaited<ReturnType<typeof offerMediaServer>>
>;
export type OfferMediaServerMutationBody = OfferInputBody;
export type OfferMediaServerMutationError = ErrorType<
  | OfferMediaServer401
  | OfferMediaServer403
  | OfferMediaServer404
  | OfferMediaServer409
  | OfferMediaServer422
  | OfferMediaServer500
  | OfferMediaServer501
  | OfferMediaServer502
>;
export type OfferMediaServerMutationVariables = {
  id: string;
  serverId: string;
  data: OfferInputBody;
};

/**
 * @summary Offer a Member access to a Media Server, or offer it again after the access was removed
 */
export const useOfferMediaServer = <
  TError = ErrorType<
    | OfferMediaServer401
    | OfferMediaServer403
    | OfferMediaServer404
    | OfferMediaServer409
    | OfferMediaServer422
    | OfferMediaServer500
    | OfferMediaServer501
    | OfferMediaServer502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof offerMediaServer>>,
      TError,
      OfferMediaServerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof offerMediaServer>>,
  TError,
  OfferMediaServerMutationVariables,
  TContext
> => {
  return useMutation(getOfferMediaServerMutationOptions(options), queryClient);
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
 * @summary Add a Media Server once it responds from its Connect from Node and accepts SlipStream's sign-in
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
 * @summary Add a Media Server once it responds from its Connect from Node and accepts SlipStream's sign-in
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
export const getListMediaServerAccessUrl = (params?: ListMediaServerAccessParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/media-servers/access?${stringifiedParams}`
    : `/api/v1/media-servers/access`;
};

/**
 * @summary List Members' access to Media Servers, the rows waiting on an Admin first
 */
export const listMediaServerAccess = async (
  params?: ListMediaServerAccessParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversAccessList> => {
  return apiFetch<MediaserversAccessList>(getListMediaServerAccessUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getListMediaServerAccessQueryKey = (params?: ListMediaServerAccessParams) => {
  return [`/api/v1/media-servers/access`, ...(params ? [params] : [])] as const;
};

export const getListMediaServerAccessSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMediaServerAccess>>,
  TError = ErrorType<
    | ListMediaServerAccess401
    | ListMediaServerAccess403
    | ListMediaServerAccess404
    | ListMediaServerAccess409
    | ListMediaServerAccess422
    | ListMediaServerAccess500
    | ListMediaServerAccess501
  >,
>(
  params?: ListMediaServerAccessParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMediaServerAccessQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMediaServerAccess>>> = ({ signal }) =>
    listMediaServerAccess(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listMediaServerAccess>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMediaServerAccessSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listMediaServerAccess>>
>;
export type ListMediaServerAccessSuspenseQueryError = ErrorType<
  | ListMediaServerAccess401
  | ListMediaServerAccess403
  | ListMediaServerAccess404
  | ListMediaServerAccess409
  | ListMediaServerAccess422
  | ListMediaServerAccess500
  | ListMediaServerAccess501
>;

export function useListMediaServerAccessSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerAccess>>,
  TError = ErrorType<
    | ListMediaServerAccess401
    | ListMediaServerAccess403
    | ListMediaServerAccess404
    | ListMediaServerAccess409
    | ListMediaServerAccess422
    | ListMediaServerAccess500
    | ListMediaServerAccess501
  >,
>(
  params: undefined | ListMediaServerAccessParams,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerAccessSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerAccess>>,
  TError = ErrorType<
    | ListMediaServerAccess401
    | ListMediaServerAccess403
    | ListMediaServerAccess404
    | ListMediaServerAccess409
    | ListMediaServerAccess422
    | ListMediaServerAccess500
    | ListMediaServerAccess501
  >,
>(
  params?: ListMediaServerAccessParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerAccessSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerAccess>>,
  TError = ErrorType<
    | ListMediaServerAccess401
    | ListMediaServerAccess403
    | ListMediaServerAccess404
    | ListMediaServerAccess409
    | ListMediaServerAccess422
    | ListMediaServerAccess500
    | ListMediaServerAccess501
  >,
>(
  params?: ListMediaServerAccessParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List Members' access to Media Servers, the rows waiting on an Admin first
 */

export function useListMediaServerAccessSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerAccess>>,
  TError = ErrorType<
    | ListMediaServerAccess401
    | ListMediaServerAccess403
    | ListMediaServerAccess404
    | ListMediaServerAccess409
    | ListMediaServerAccess422
    | ListMediaServerAccess500
    | ListMediaServerAccess501
  >,
>(
  params?: ListMediaServerAccessParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerAccess>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMediaServerAccessSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getRemoveMediaServerAccessUrl = (id: string) => {
  return `/api/v1/media-servers/access/${id}`;
};

/**
 * @summary Remove a Member's access to a Media Server, on the server or plex.tv too
 */
export const removeMediaServerAccess = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversActed> => {
  return apiFetch<MediaserversActed>(getRemoveMediaServerAccessUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getRemoveMediaServerAccessMutationKey = () => ["removeMediaServerAccess"] as const;

export const getRemoveMediaServerAccessMutationOptions = <
  TError = ErrorType<
    | RemoveMediaServerAccess401
    | RemoveMediaServerAccess403
    | RemoveMediaServerAccess404
    | RemoveMediaServerAccess409
    | RemoveMediaServerAccess422
    | RemoveMediaServerAccess500
    | RemoveMediaServerAccess501
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof removeMediaServerAccess>>,
    TError,
    RemoveMediaServerAccessMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof removeMediaServerAccess>>,
  TError,
  RemoveMediaServerAccessMutationVariables,
  TContext
> => {
  const mutationKey = getRemoveMediaServerAccessMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof removeMediaServerAccess>>,
    RemoveMediaServerAccessMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return removeMediaServerAccess(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RemoveMediaServerAccessMutationResult = NonNullable<
  Awaited<ReturnType<typeof removeMediaServerAccess>>
>;

export type RemoveMediaServerAccessMutationError = ErrorType<
  | RemoveMediaServerAccess401
  | RemoveMediaServerAccess403
  | RemoveMediaServerAccess404
  | RemoveMediaServerAccess409
  | RemoveMediaServerAccess422
  | RemoveMediaServerAccess500
  | RemoveMediaServerAccess501
>;
export type RemoveMediaServerAccessMutationVariables = { id: string };

/**
 * @summary Remove a Member's access to a Media Server, on the server or plex.tv too
 */
export const useRemoveMediaServerAccess = <
  TError = ErrorType<
    | RemoveMediaServerAccess401
    | RemoveMediaServerAccess403
    | RemoveMediaServerAccess404
    | RemoveMediaServerAccess409
    | RemoveMediaServerAccess422
    | RemoveMediaServerAccess500
    | RemoveMediaServerAccess501
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof removeMediaServerAccess>>,
      TError,
      RemoveMediaServerAccessMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof removeMediaServerAccess>>,
  TError,
  RemoveMediaServerAccessMutationVariables,
  TContext
> => {
  return useMutation(getRemoveMediaServerAccessMutationOptions(options), queryClient);
};
export const getCheckMediaServerAccessUrl = (id: string) => {
  return `/api/v1/media-servers/access/${id}/check`;
};

/**
 * @summary Check whether a Member accepted a Plex share
 */
export const checkMediaServerAccess = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversActed> => {
  return apiFetch<MediaserversActed>(getCheckMediaServerAccessUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getCheckMediaServerAccessMutationKey = () => ["checkMediaServerAccess"] as const;

export const getCheckMediaServerAccessMutationOptions = <
  TError = ErrorType<
    | CheckMediaServerAccess401
    | CheckMediaServerAccess403
    | CheckMediaServerAccess404
    | CheckMediaServerAccess409
    | CheckMediaServerAccess422
    | CheckMediaServerAccess500
    | CheckMediaServerAccess501
    | CheckMediaServerAccess502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof checkMediaServerAccess>>,
    TError,
    CheckMediaServerAccessMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof checkMediaServerAccess>>,
  TError,
  CheckMediaServerAccessMutationVariables,
  TContext
> => {
  const mutationKey = getCheckMediaServerAccessMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof checkMediaServerAccess>>,
    CheckMediaServerAccessMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return checkMediaServerAccess(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CheckMediaServerAccessMutationResult = NonNullable<
  Awaited<ReturnType<typeof checkMediaServerAccess>>
>;

export type CheckMediaServerAccessMutationError = ErrorType<
  | CheckMediaServerAccess401
  | CheckMediaServerAccess403
  | CheckMediaServerAccess404
  | CheckMediaServerAccess409
  | CheckMediaServerAccess422
  | CheckMediaServerAccess500
  | CheckMediaServerAccess501
  | CheckMediaServerAccess502
>;
export type CheckMediaServerAccessMutationVariables = { id: string };

/**
 * @summary Check whether a Member accepted a Plex share
 */
export const useCheckMediaServerAccess = <
  TError = ErrorType<
    | CheckMediaServerAccess401
    | CheckMediaServerAccess403
    | CheckMediaServerAccess404
    | CheckMediaServerAccess409
    | CheckMediaServerAccess422
    | CheckMediaServerAccess500
    | CheckMediaServerAccess501
    | CheckMediaServerAccess502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof checkMediaServerAccess>>,
      TError,
      CheckMediaServerAccessMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof checkMediaServerAccess>>,
  TError,
  CheckMediaServerAccessMutationVariables,
  TContext
> => {
  return useMutation(getCheckMediaServerAccessMutationOptions(options), queryClient);
};
export const getLinkMediaServerUserUrl = (id: string) => {
  return `/api/v1/media-servers/access/${id}/link`;
};

/**
 * @summary Link a Member to an existing user on a Jellyfin or Emby server
 */
export const linkMediaServerUser = async (
  id: string,
  linkUserInputBody: LinkUserInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversActed> => {
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
  return apiFetch<MediaserversActed>(getLinkMediaServerUserUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(linkUserInputBody),
  });
};

export const getLinkMediaServerUserMutationKey = () => ["linkMediaServerUser"] as const;

export const getLinkMediaServerUserMutationOptions = <
  TError = ErrorType<
    | LinkMediaServerUser401
    | LinkMediaServerUser403
    | LinkMediaServerUser404
    | LinkMediaServerUser409
    | LinkMediaServerUser422
    | LinkMediaServerUser500
    | LinkMediaServerUser501
    | LinkMediaServerUser502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof linkMediaServerUser>>,
    TError,
    LinkMediaServerUserMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof linkMediaServerUser>>,
  TError,
  LinkMediaServerUserMutationVariables,
  TContext
> => {
  const mutationKey = getLinkMediaServerUserMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof linkMediaServerUser>>,
    LinkMediaServerUserMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return linkMediaServerUser(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type LinkMediaServerUserMutationResult = NonNullable<
  Awaited<ReturnType<typeof linkMediaServerUser>>
>;
export type LinkMediaServerUserMutationBody = LinkUserInputBody;
export type LinkMediaServerUserMutationError = ErrorType<
  | LinkMediaServerUser401
  | LinkMediaServerUser403
  | LinkMediaServerUser404
  | LinkMediaServerUser409
  | LinkMediaServerUser422
  | LinkMediaServerUser500
  | LinkMediaServerUser501
  | LinkMediaServerUser502
>;
export type LinkMediaServerUserMutationVariables = { id: string; data: LinkUserInputBody };

/**
 * @summary Link a Member to an existing user on a Jellyfin or Emby server
 */
export const useLinkMediaServerUser = <
  TError = ErrorType<
    | LinkMediaServerUser401
    | LinkMediaServerUser403
    | LinkMediaServerUser404
    | LinkMediaServerUser409
    | LinkMediaServerUser422
    | LinkMediaServerUser500
    | LinkMediaServerUser501
    | LinkMediaServerUser502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof linkMediaServerUser>>,
      TError,
      LinkMediaServerUserMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof linkMediaServerUser>>,
  TError,
  LinkMediaServerUserMutationVariables,
  TContext
> => {
  return useMutation(getLinkMediaServerUserMutationOptions(options), queryClient);
};
export const getMarkMediaServerAccessAddedUrl = (id: string) => {
  return `/api/v1/media-servers/access/${id}/mark-added`;
};

/**
 * @summary Record that an Admin shared a Plex server with a Member in Plex itself
 */
export const markMediaServerAccessAdded = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversActed> => {
  return apiFetch<MediaserversActed>(getMarkMediaServerAccessAddedUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getMarkMediaServerAccessAddedMutationKey = () =>
  ["markMediaServerAccessAdded"] as const;

export const getMarkMediaServerAccessAddedMutationOptions = <
  TError = ErrorType<
    | MarkMediaServerAccessAdded401
    | MarkMediaServerAccessAdded403
    | MarkMediaServerAccessAdded404
    | MarkMediaServerAccessAdded409
    | MarkMediaServerAccessAdded422
    | MarkMediaServerAccessAdded500
    | MarkMediaServerAccessAdded501
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof markMediaServerAccessAdded>>,
    TError,
    MarkMediaServerAccessAddedMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof markMediaServerAccessAdded>>,
  TError,
  MarkMediaServerAccessAddedMutationVariables,
  TContext
> => {
  const mutationKey = getMarkMediaServerAccessAddedMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof markMediaServerAccessAdded>>,
    MarkMediaServerAccessAddedMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return markMediaServerAccessAdded(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MarkMediaServerAccessAddedMutationResult = NonNullable<
  Awaited<ReturnType<typeof markMediaServerAccessAdded>>
>;

export type MarkMediaServerAccessAddedMutationError = ErrorType<
  | MarkMediaServerAccessAdded401
  | MarkMediaServerAccessAdded403
  | MarkMediaServerAccessAdded404
  | MarkMediaServerAccessAdded409
  | MarkMediaServerAccessAdded422
  | MarkMediaServerAccessAdded500
  | MarkMediaServerAccessAdded501
>;
export type MarkMediaServerAccessAddedMutationVariables = { id: string };

/**
 * @summary Record that an Admin shared a Plex server with a Member in Plex itself
 */
export const useMarkMediaServerAccessAdded = <
  TError = ErrorType<
    | MarkMediaServerAccessAdded401
    | MarkMediaServerAccessAdded403
    | MarkMediaServerAccessAdded404
    | MarkMediaServerAccessAdded409
    | MarkMediaServerAccessAdded422
    | MarkMediaServerAccessAdded500
    | MarkMediaServerAccessAdded501
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof markMediaServerAccessAdded>>,
      TError,
      MarkMediaServerAccessAddedMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof markMediaServerAccessAdded>>,
  TError,
  MarkMediaServerAccessAddedMutationVariables,
  TContext
> => {
  return useMutation(getMarkMediaServerAccessAddedMutationOptions(options), queryClient);
};
export const getShareMediaServerAccessUrl = (id: string) => {
  return `/api/v1/media-servers/access/${id}/share`;
};

/**
 * @summary Share a Plex server with a Member who signed in to Plex, as a friend or in the Plex Home
 */
export const shareMediaServerAccess = async (
  id: string,
  shareAccessInputBody: ShareAccessInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversActed> => {
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
  return apiFetch<MediaserversActed>(getShareMediaServerAccessUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(shareAccessInputBody),
  });
};

export const getShareMediaServerAccessMutationKey = () => ["shareMediaServerAccess"] as const;

export const getShareMediaServerAccessMutationOptions = <
  TError = ErrorType<
    | ShareMediaServerAccess401
    | ShareMediaServerAccess403
    | ShareMediaServerAccess404
    | ShareMediaServerAccess409
    | ShareMediaServerAccess422
    | ShareMediaServerAccess500
    | ShareMediaServerAccess501
    | ShareMediaServerAccess502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof shareMediaServerAccess>>,
    TError,
    ShareMediaServerAccessMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof shareMediaServerAccess>>,
  TError,
  ShareMediaServerAccessMutationVariables,
  TContext
> => {
  const mutationKey = getShareMediaServerAccessMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof shareMediaServerAccess>>,
    ShareMediaServerAccessMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return shareMediaServerAccess(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ShareMediaServerAccessMutationResult = NonNullable<
  Awaited<ReturnType<typeof shareMediaServerAccess>>
>;
export type ShareMediaServerAccessMutationBody = ShareAccessInputBody;
export type ShareMediaServerAccessMutationError = ErrorType<
  | ShareMediaServerAccess401
  | ShareMediaServerAccess403
  | ShareMediaServerAccess404
  | ShareMediaServerAccess409
  | ShareMediaServerAccess422
  | ShareMediaServerAccess500
  | ShareMediaServerAccess501
  | ShareMediaServerAccess502
>;
export type ShareMediaServerAccessMutationVariables = { id: string; data: ShareAccessInputBody };

/**
 * @summary Share a Plex server with a Member who signed in to Plex, as a friend or in the Plex Home
 */
export const useShareMediaServerAccess = <
  TError = ErrorType<
    | ShareMediaServerAccess401
    | ShareMediaServerAccess403
    | ShareMediaServerAccess404
    | ShareMediaServerAccess409
    | ShareMediaServerAccess422
    | ShareMediaServerAccess500
    | ShareMediaServerAccess501
    | ShareMediaServerAccess502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof shareMediaServerAccess>>,
      TError,
      ShareMediaServerAccessMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof shareMediaServerAccess>>,
  TError,
  ShareMediaServerAccessMutationVariables,
  TContext
> => {
  return useMutation(getShareMediaServerAccessMutationOptions(options), queryClient);
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

export const getReplacePlexAccountUrl = (id: number) => {
  return `/api/v1/media-servers/plex/pin/${id}/replace`;
};

/**
 * @summary Sign SlipStream in to the different Plex account a Plex sign-in names, which removes every Plex server and Members' access to them
 */
export const replacePlexAccount = async (
  id: number,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MediaserversAccountView> => {
  return apiFetch<MediaserversAccountView>(getReplacePlexAccountUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getReplacePlexAccountMutationKey = () => ["replacePlexAccount"] as const;

export const getReplacePlexAccountMutationOptions = <
  TError = ErrorType<
    | ReplacePlexAccount401
    | ReplacePlexAccount403
    | ReplacePlexAccount404
    | ReplacePlexAccount409
    | ReplacePlexAccount422
    | ReplacePlexAccount500
    | ReplacePlexAccount501
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof replacePlexAccount>>,
    TError,
    ReplacePlexAccountMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof replacePlexAccount>>,
  TError,
  ReplacePlexAccountMutationVariables,
  TContext
> => {
  const mutationKey = getReplacePlexAccountMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof replacePlexAccount>>,
    ReplacePlexAccountMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return replacePlexAccount(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ReplacePlexAccountMutationResult = NonNullable<
  Awaited<ReturnType<typeof replacePlexAccount>>
>;

export type ReplacePlexAccountMutationError = ErrorType<
  | ReplacePlexAccount401
  | ReplacePlexAccount403
  | ReplacePlexAccount404
  | ReplacePlexAccount409
  | ReplacePlexAccount422
  | ReplacePlexAccount500
  | ReplacePlexAccount501
>;
export type ReplacePlexAccountMutationVariables = { id: number };

/**
 * @summary Sign SlipStream in to the different Plex account a Plex sign-in names, which removes every Plex server and Members' access to them
 */
export const useReplacePlexAccount = <
  TError = ErrorType<
    | ReplacePlexAccount401
    | ReplacePlexAccount403
    | ReplacePlexAccount404
    | ReplacePlexAccount409
    | ReplacePlexAccount422
    | ReplacePlexAccount500
    | ReplacePlexAccount501
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof replacePlexAccount>>,
      TError,
      ReplacePlexAccountMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof replacePlexAccount>>,
  TError,
  ReplacePlexAccountMutationVariables,
  TContext
> => {
  return useMutation(getReplacePlexAccountMutationOptions(options), queryClient);
};
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
 * @summary Change a Media Server, testing it again when the change is to its address, Connect from Node or API key
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
    | UpdateMediaServer501
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
  | UpdateMediaServer501
  | UpdateMediaServer502
>;
export type UpdateMediaServerMutationVariables = { id: string; data: MediaserversPatchBody };

/**
 * @summary Change a Media Server, testing it again when the change is to its address, Connect from Node or API key
 */
export const useUpdateMediaServer = <
  TError = ErrorType<
    | UpdateMediaServer401
    | UpdateMediaServer403
    | UpdateMediaServer404
    | UpdateMediaServer409
    | UpdateMediaServer422
    | UpdateMediaServer500
    | UpdateMediaServer501
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
 * @summary Confirm that the server now responding at a Media Server's address is that Media Server
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
 * @summary Confirm that the server now responding at a Media Server's address is that Media Server
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
 * @summary List what plays on a Media Server now, as its Connect from Node last saw it
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
 * @summary List what plays on a Media Server now, as its Connect from Node last saw it
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
 * @summary Test a Media Server from its Connect from Node at its address and from the Controller at its fallback address
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
 * @summary Test a Media Server from its Connect from Node at its address and from the Controller at its fallback address
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
export const getListMediaServerUsersUrl = (id: string) => {
  return `/api/v1/media-servers/${id}/users`;
};

/**
 * @summary List the users of a Jellyfin or Emby server that an Admin may link to a Member
 */
export const listMediaServerUsers = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<ServerUsersOutputBody> => {
  return apiFetch<ServerUsersOutputBody>(getListMediaServerUsersUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getListMediaServerUsersQueryKey = (id: string) => {
  return [`/api/v1/media-servers/${id}/users`] as const;
};

export const getListMediaServerUsersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listMediaServerUsers>>,
  TError = ErrorType<
    | ListMediaServerUsers401
    | ListMediaServerUsers403
    | ListMediaServerUsers404
    | ListMediaServerUsers409
    | ListMediaServerUsers422
    | ListMediaServerUsers500
    | ListMediaServerUsers501
    | ListMediaServerUsers502
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerUsers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListMediaServerUsersQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listMediaServerUsers>>> = ({ signal }) =>
    listMediaServerUsers(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerUsers>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListMediaServerUsersSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listMediaServerUsers>>
>;
export type ListMediaServerUsersSuspenseQueryError = ErrorType<
  | ListMediaServerUsers401
  | ListMediaServerUsers403
  | ListMediaServerUsers404
  | ListMediaServerUsers409
  | ListMediaServerUsers422
  | ListMediaServerUsers500
  | ListMediaServerUsers501
  | ListMediaServerUsers502
>;

export function useListMediaServerUsersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerUsers>>,
  TError = ErrorType<
    | ListMediaServerUsers401
    | ListMediaServerUsers403
    | ListMediaServerUsers404
    | ListMediaServerUsers409
    | ListMediaServerUsers422
    | ListMediaServerUsers500
    | ListMediaServerUsers501
    | ListMediaServerUsers502
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerUsers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerUsersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerUsers>>,
  TError = ErrorType<
    | ListMediaServerUsers401
    | ListMediaServerUsers403
    | ListMediaServerUsers404
    | ListMediaServerUsers409
    | ListMediaServerUsers422
    | ListMediaServerUsers500
    | ListMediaServerUsers501
    | ListMediaServerUsers502
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerUsers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListMediaServerUsersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerUsers>>,
  TError = ErrorType<
    | ListMediaServerUsers401
    | ListMediaServerUsers403
    | ListMediaServerUsers404
    | ListMediaServerUsers409
    | ListMediaServerUsers422
    | ListMediaServerUsers500
    | ListMediaServerUsers501
    | ListMediaServerUsers502
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerUsers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the users of a Jellyfin or Emby server that an Admin may link to a Member
 */

export function useListMediaServerUsersSuspense<
  TData = Awaited<ReturnType<typeof listMediaServerUsers>>,
  TError = ErrorType<
    | ListMediaServerUsers401
    | ListMediaServerUsers403
    | ListMediaServerUsers404
    | ListMediaServerUsers409
    | ListMediaServerUsers422
    | ListMediaServerUsers500
    | ListMediaServerUsers501
    | ListMediaServerUsers502
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listMediaServerUsers>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListMediaServerUsersSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
