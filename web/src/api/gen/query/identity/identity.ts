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
  AccountPatchInputBody,
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
  ConfirmEmail403,
  ConfirmEmail409,
  ConfirmEmail410,
  ConfirmEmail422,
  ConfirmEmail500,
  CreateResetLink401,
  CreateResetLink403,
  CreateResetLink404,
  CreateResetLink409,
  CreateResetLink422,
  CreateResetLink500,
  CreateSession401,
  CreateSession403,
  CreateSession422,
  CreateSession429,
  CreateSession500,
  CreateSignupLink401,
  CreateSignupLink403,
  CreateSignupLink409,
  CreateSignupLink422,
  CreateSignupLink500,
  DeleteAccount401,
  DeleteAccount403,
  DeleteAccount404,
  DeleteAccount409,
  DeleteAccount422,
  DeleteAccount500,
  DeletePasskey401,
  DeletePasskey403,
  DeletePasskey404,
  DeletePasskey409,
  DeletePasskey422,
  DeletePasskey500,
  DeleteSession401,
  DeleteSession403,
  DeleteSession409,
  DeleteSession500,
  DeleteSignupLink401,
  DeleteSignupLink403,
  DeleteSignupLink404,
  DeleteSignupLink409,
  DeleteSignupLink422,
  DeleteSignupLink500,
  EmailChangeInputBody,
  EndAccountSession401,
  EndAccountSession403,
  EndAccountSession404,
  EndAccountSession409,
  EndAccountSession422,
  EndAccountSession500,
  EndAccountSessions401,
  EndAccountSessions403,
  EndAccountSessions404,
  EndAccountSessions409,
  EndAccountSessions422,
  EndAccountSessions500,
  EndOtherSessions401,
  EndOtherSessions403,
  EndOtherSessions409,
  EndOtherSessions422,
  EndOtherSessions500,
  EndOtherSessionsParams,
  EndSession401,
  EndSession403,
  EndSession404,
  EndSession409,
  EndSession422,
  EndSession500,
  FinishPasskeyRegistration401,
  FinishPasskeyRegistration403,
  FinishPasskeyRegistration409,
  FinishPasskeyRegistration422,
  FinishPasskeyRegistration500,
  GetAPIKey401,
  GetAPIKey403,
  GetAPIKey409,
  GetAPIKey500,
  GetAccount401,
  GetAccount403,
  GetAccount404,
  GetAccount409,
  GetAccount422,
  GetAccount500,
  GetEmailLinkByToken422,
  GetEmailLinkByToken500,
  GetIdentitySettings401,
  GetIdentitySettings403,
  GetIdentitySettings409,
  GetIdentitySettings500,
  GetResetLinkByToken422,
  GetResetLinkByToken500,
  GetSession401,
  GetSession403,
  GetSession409,
  GetSession500,
  GetSignupLink401,
  GetSignupLink403,
  GetSignupLink404,
  GetSignupLink409,
  GetSignupLink422,
  GetSignupLink500,
  GetSignupLinkByToken422,
  GetSignupLinkByToken500,
  IdentityAccountDetail,
  IdentityAccountRow,
  IdentityAccountRows,
  IdentityApiKeyMade,
  IdentityCeremony,
  IdentityClaimBody,
  IdentityClaimed,
  IdentityEmailChangeStarted,
  IdentityIssuedResetLink,
  IdentityIssuedSignupLink,
  IdentityLinkView,
  IdentityPasskey,
  IdentityPasskeyRows,
  IdentityPendingLinks,
  IdentityPinChange,
  IdentityReconfirmation,
  IdentityRegeneratedAPIKey,
  IdentitySession,
  IdentitySessionRows,
  IdentitySettings,
  IdentitySettingsPatch,
  IdentitySignInBody,
  IdentitySignedIn,
  IdentitySignupLinkURL,
  JoinInputBody,
  ListAccountSessions401,
  ListAccountSessions403,
  ListAccountSessions404,
  ListAccountSessions409,
  ListAccountSessions422,
  ListAccountSessions500,
  ListAccounts401,
  ListAccounts403,
  ListAccounts409,
  ListAccounts500,
  ListPasskeys401,
  ListPasskeys403,
  ListPasskeys409,
  ListPasskeys500,
  ListSessions401,
  ListSessions403,
  ListSessions409,
  ListSessions500,
  ListSignupLinks401,
  ListSignupLinks403,
  ListSignupLinks409,
  ListSignupLinks500,
  NewPINInputBody,
  NewSignupLinkInputBody,
  OwnPatchInputBody,
  PasskeyRenameInputBody,
  ReconfirmSession401,
  ReconfirmSession403,
  ReconfirmSession409,
  ReconfirmSession422,
  ReconfirmSession429,
  ReconfirmSession500,
  RedeemResetLink403,
  RedeemResetLink410,
  RedeemResetLink422,
  RedeemResetLink500,
  RedeemSignupLink409,
  RedeemSignupLink410,
  RedeemSignupLink422,
  RedeemSignupLink500,
  RegenerateAPIKey401,
  RegenerateAPIKey403,
  RegenerateAPIKey409,
  RegenerateAPIKey500,
  ResendSignupLink401,
  ResendSignupLink403,
  ResendSignupLink404,
  ResendSignupLink409,
  ResendSignupLink422,
  ResendSignupLink500,
  ResetLinkInputBody,
  StartEmailChange401,
  StartEmailChange403,
  StartEmailChange409,
  StartEmailChange422,
  StartEmailChange500,
  StartPasskeyRegistration401,
  StartPasskeyRegistration403,
  StartPasskeyRegistration409,
  StartPasskeyRegistration500,
  UpdateAccount401,
  UpdateAccount403,
  UpdateAccount404,
  UpdateAccount409,
  UpdateAccount422,
  UpdateAccount500,
  UpdateIdentitySettings401,
  UpdateIdentitySettings403,
  UpdateIdentitySettings409,
  UpdateIdentitySettings422,
  UpdateIdentitySettings500,
  UpdateOwnAccount401,
  UpdateOwnAccount403,
  UpdateOwnAccount409,
  UpdateOwnAccount422,
  UpdateOwnAccount500,
  UpdatePasskey401,
  UpdatePasskey403,
  UpdatePasskey404,
  UpdatePasskey409,
  UpdatePasskey422,
  UpdatePasskey500,
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

export const getUpdateOwnAccountUrl = () => {
  return `/api/v1/account`;
};

/**
 * @summary Change your display name, or whether the passkey prompt may show
 */
export const updateOwnAccount = async (
  ownPatchInputBody: OwnPatchInputBody,
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
  return apiFetch<IdentitySession>(getUpdateOwnAccountUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(ownPatchInputBody),
  });
};

export const getUpdateOwnAccountMutationKey = () => ["updateOwnAccount"] as const;

export const getUpdateOwnAccountMutationOptions = <
  TError = ErrorType<
    | UpdateOwnAccount401
    | UpdateOwnAccount403
    | UpdateOwnAccount409
    | UpdateOwnAccount422
    | UpdateOwnAccount500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateOwnAccount>>,
    TError,
    UpdateOwnAccountMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateOwnAccount>>,
  TError,
  UpdateOwnAccountMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateOwnAccountMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateOwnAccount>>,
    UpdateOwnAccountMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateOwnAccount(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateOwnAccountMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateOwnAccount>>
>;
export type UpdateOwnAccountMutationBody = OwnPatchInputBody;
export type UpdateOwnAccountMutationError = ErrorType<
  | UpdateOwnAccount401
  | UpdateOwnAccount403
  | UpdateOwnAccount409
  | UpdateOwnAccount422
  | UpdateOwnAccount500
>;
export type UpdateOwnAccountMutationVariables = { data: OwnPatchInputBody };

/**
 * @summary Change your display name, or whether the passkey prompt may show
 */
export const useUpdateOwnAccount = <
  TError = ErrorType<
    | UpdateOwnAccount401
    | UpdateOwnAccount403
    | UpdateOwnAccount409
    | UpdateOwnAccount422
    | UpdateOwnAccount500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateOwnAccount>>,
      TError,
      UpdateOwnAccountMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateOwnAccount>>,
  TError,
  UpdateOwnAccountMutationVariables,
  TContext
> => {
  return useMutation(getUpdateOwnAccountMutationOptions(options), queryClient);
};
export const getStartEmailChangeUrl = () => {
  return `/api/v1/account/email`;
};

/**
 * @summary Change your email: a confirm link goes to the new address, and the email changes when it is followed
 */
export const startEmailChange = async (
  emailChangeInputBody: EmailChangeInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityEmailChangeStarted> => {
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
  return apiFetch<IdentityEmailChangeStarted>(getStartEmailChangeUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(emailChangeInputBody),
  });
};

export const getStartEmailChangeMutationKey = () => ["startEmailChange"] as const;

export const getStartEmailChangeMutationOptions = <
  TError = ErrorType<
    | StartEmailChange401
    | StartEmailChange403
    | StartEmailChange409
    | StartEmailChange422
    | StartEmailChange500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof startEmailChange>>,
    TError,
    StartEmailChangeMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof startEmailChange>>,
  TError,
  StartEmailChangeMutationVariables,
  TContext
> => {
  const mutationKey = getStartEmailChangeMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof startEmailChange>>,
    StartEmailChangeMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return startEmailChange(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type StartEmailChangeMutationResult = NonNullable<
  Awaited<ReturnType<typeof startEmailChange>>
>;
export type StartEmailChangeMutationBody = EmailChangeInputBody;
export type StartEmailChangeMutationError = ErrorType<
  | StartEmailChange401
  | StartEmailChange403
  | StartEmailChange409
  | StartEmailChange422
  | StartEmailChange500
>;
export type StartEmailChangeMutationVariables = { data: EmailChangeInputBody };

/**
 * @summary Change your email: a confirm link goes to the new address, and the email changes when it is followed
 */
export const useStartEmailChange = <
  TError = ErrorType<
    | StartEmailChange401
    | StartEmailChange403
    | StartEmailChange409
    | StartEmailChange422
    | StartEmailChange500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof startEmailChange>>,
      TError,
      StartEmailChangeMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof startEmailChange>>,
  TError,
  StartEmailChangeMutationVariables,
  TContext
> => {
  return useMutation(getStartEmailChangeMutationOptions(options), queryClient);
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
export const getListAccountsUrl = () => {
  return `/api/v1/accounts`;
};

/**
 * @summary List every Account
 */
export const listAccounts = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityAccountRows> => {
  return apiFetch<IdentityAccountRows>(getListAccountsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListAccountsQueryKey = () => {
  return [`/api/v1/accounts`] as const;
};

export const getListAccountsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listAccounts>>,
  TError = ErrorType<ListAccounts401 | ListAccounts403 | ListAccounts409 | ListAccounts500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccounts>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListAccountsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listAccounts>>> = ({ signal }) =>
    listAccounts({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccounts>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListAccountsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listAccounts>>>;
export type ListAccountsSuspenseQueryError = ErrorType<
  ListAccounts401 | ListAccounts403 | ListAccounts409 | ListAccounts500
>;

export function useListAccountsSuspense<
  TData = Awaited<ReturnType<typeof listAccounts>>,
  TError = ErrorType<ListAccounts401 | ListAccounts403 | ListAccounts409 | ListAccounts500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccounts>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListAccountsSuspense<
  TData = Awaited<ReturnType<typeof listAccounts>>,
  TError = ErrorType<ListAccounts401 | ListAccounts403 | ListAccounts409 | ListAccounts500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccounts>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListAccountsSuspense<
  TData = Awaited<ReturnType<typeof listAccounts>>,
  TError = ErrorType<ListAccounts401 | ListAccounts403 | ListAccounts409 | ListAccounts500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccounts>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List every Account
 */

export function useListAccountsSuspense<
  TData = Awaited<ReturnType<typeof listAccounts>>,
  TError = ErrorType<ListAccounts401 | ListAccounts403 | ListAccounts409 | ListAccounts500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccounts>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListAccountsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getDeleteAccountUrl = (id: string) => {
  return `/api/v1/accounts/${id}`;
};

/**
 * @summary Delete an Account with its sessions, passkeys and links
 */
export const deleteAccount = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeleteAccountUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteAccountMutationKey = () => ["deleteAccount"] as const;

export const getDeleteAccountMutationOptions = <
  TError = ErrorType<
    | DeleteAccount401
    | DeleteAccount403
    | DeleteAccount404
    | DeleteAccount409
    | DeleteAccount422
    | DeleteAccount500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteAccount>>,
    TError,
    DeleteAccountMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteAccount>>,
  TError,
  DeleteAccountMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteAccountMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteAccount>>,
    DeleteAccountMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return deleteAccount(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteAccountMutationResult = NonNullable<Awaited<ReturnType<typeof deleteAccount>>>;

export type DeleteAccountMutationError = ErrorType<
  | DeleteAccount401
  | DeleteAccount403
  | DeleteAccount404
  | DeleteAccount409
  | DeleteAccount422
  | DeleteAccount500
>;
export type DeleteAccountMutationVariables = { id: string };

/**
 * @summary Delete an Account with its sessions, passkeys and links
 */
export const useDeleteAccount = <
  TError = ErrorType<
    | DeleteAccount401
    | DeleteAccount403
    | DeleteAccount404
    | DeleteAccount409
    | DeleteAccount422
    | DeleteAccount500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteAccount>>,
      TError,
      DeleteAccountMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteAccount>>,
  TError,
  DeleteAccountMutationVariables,
  TContext
> => {
  return useMutation(getDeleteAccountMutationOptions(options), queryClient);
};
export const getGetAccountUrl = (id: string) => {
  return `/api/v1/accounts/${id}`;
};

/**
 * @summary Get an Account with its passkeys and sessions
 */
export const getAccount = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityAccountDetail> => {
  return apiFetch<IdentityAccountDetail>(getGetAccountUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetAccountQueryKey = (id: string) => {
  return [`/api/v1/accounts/${id}`] as const;
};

export const getGetAccountSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getAccount>>,
  TError = ErrorType<
    GetAccount401 | GetAccount403 | GetAccount404 | GetAccount409 | GetAccount422 | GetAccount500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAccount>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetAccountQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getAccount>>> = ({ signal }) =>
    getAccount(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAccount>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetAccountSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getAccount>>>;
export type GetAccountSuspenseQueryError = ErrorType<
  GetAccount401 | GetAccount403 | GetAccount404 | GetAccount409 | GetAccount422 | GetAccount500
>;

export function useGetAccountSuspense<
  TData = Awaited<ReturnType<typeof getAccount>>,
  TError = ErrorType<
    GetAccount401 | GetAccount403 | GetAccount404 | GetAccount409 | GetAccount422 | GetAccount500
  >,
>(
  id: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAccount>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetAccountSuspense<
  TData = Awaited<ReturnType<typeof getAccount>>,
  TError = ErrorType<
    GetAccount401 | GetAccount403 | GetAccount404 | GetAccount409 | GetAccount422 | GetAccount500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAccount>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetAccountSuspense<
  TData = Awaited<ReturnType<typeof getAccount>>,
  TError = ErrorType<
    GetAccount401 | GetAccount403 | GetAccount404 | GetAccount409 | GetAccount422 | GetAccount500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAccount>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get an Account with its passkeys and sessions
 */

export function useGetAccountSuspense<
  TData = Awaited<ReturnType<typeof getAccount>>,
  TError = ErrorType<
    GetAccount401 | GetAccount403 | GetAccount404 | GetAccount409 | GetAccount422 | GetAccount500
  >,
>(
  id: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getAccount>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetAccountSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateAccountUrl = (id: string) => {
  return `/api/v1/accounts/${id}`;
};

/**
 * @summary Change an Account's name, email or role, or turn it off or on
 */
export const updateAccount = async (
  id: string,
  accountPatchInputBody: AccountPatchInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityAccountRow> => {
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
  return apiFetch<IdentityAccountRow>(getUpdateAccountUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(accountPatchInputBody),
  });
};

export const getUpdateAccountMutationKey = () => ["updateAccount"] as const;

export const getUpdateAccountMutationOptions = <
  TError = ErrorType<
    | UpdateAccount401
    | UpdateAccount403
    | UpdateAccount404
    | UpdateAccount409
    | UpdateAccount422
    | UpdateAccount500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateAccount>>,
    TError,
    UpdateAccountMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateAccount>>,
  TError,
  UpdateAccountMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateAccountMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateAccount>>,
    UpdateAccountMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return updateAccount(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateAccountMutationResult = NonNullable<Awaited<ReturnType<typeof updateAccount>>>;
export type UpdateAccountMutationBody = AccountPatchInputBody;
export type UpdateAccountMutationError = ErrorType<
  | UpdateAccount401
  | UpdateAccount403
  | UpdateAccount404
  | UpdateAccount409
  | UpdateAccount422
  | UpdateAccount500
>;
export type UpdateAccountMutationVariables = { id: string; data: AccountPatchInputBody };

/**
 * @summary Change an Account's name, email or role, or turn it off or on
 */
export const useUpdateAccount = <
  TError = ErrorType<
    | UpdateAccount401
    | UpdateAccount403
    | UpdateAccount404
    | UpdateAccount409
    | UpdateAccount422
    | UpdateAccount500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateAccount>>,
      TError,
      UpdateAccountMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateAccount>>,
  TError,
  UpdateAccountMutationVariables,
  TContext
> => {
  return useMutation(getUpdateAccountMutationOptions(options), queryClient);
};
export const getCreateResetLinkUrl = (id: string) => {
  return `/api/v1/accounts/${id}/reset-link`;
};

/**
 * @summary Issue a Reset link for an Account, ending its sessions; the link is shown only in this answer
 */
export const createResetLink = async (
  id: string,
  resetLinkInputBody: ResetLinkInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityIssuedResetLink> => {
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
  return apiFetch<IdentityIssuedResetLink>(getCreateResetLinkUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(resetLinkInputBody),
  });
};

export const getCreateResetLinkMutationKey = () => ["createResetLink"] as const;

export const getCreateResetLinkMutationOptions = <
  TError = ErrorType<
    | CreateResetLink401
    | CreateResetLink403
    | CreateResetLink404
    | CreateResetLink409
    | CreateResetLink422
    | CreateResetLink500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof createResetLink>>,
    TError,
    CreateResetLinkMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof createResetLink>>,
  TError,
  CreateResetLinkMutationVariables,
  TContext
> => {
  const mutationKey = getCreateResetLinkMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof createResetLink>>,
    CreateResetLinkMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return createResetLink(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateResetLinkMutationResult = NonNullable<
  Awaited<ReturnType<typeof createResetLink>>
>;
export type CreateResetLinkMutationBody = ResetLinkInputBody;
export type CreateResetLinkMutationError = ErrorType<
  | CreateResetLink401
  | CreateResetLink403
  | CreateResetLink404
  | CreateResetLink409
  | CreateResetLink422
  | CreateResetLink500
>;
export type CreateResetLinkMutationVariables = { id: string; data: ResetLinkInputBody };

/**
 * @summary Issue a Reset link for an Account, ending its sessions; the link is shown only in this answer
 */
export const useCreateResetLink = <
  TError = ErrorType<
    | CreateResetLink401
    | CreateResetLink403
    | CreateResetLink404
    | CreateResetLink409
    | CreateResetLink422
    | CreateResetLink500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createResetLink>>,
      TError,
      CreateResetLinkMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof createResetLink>>,
  TError,
  CreateResetLinkMutationVariables,
  TContext
> => {
  return useMutation(getCreateResetLinkMutationOptions(options), queryClient);
};
export const getEndAccountSessionsUrl = (id: string) => {
  return `/api/v1/accounts/${id}/sessions`;
};

/**
 * @summary Sign an Account out everywhere
 */
export const endAccountSessions = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getEndAccountSessionsUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getEndAccountSessionsMutationKey = () => ["endAccountSessions"] as const;

export const getEndAccountSessionsMutationOptions = <
  TError = ErrorType<
    | EndAccountSessions401
    | EndAccountSessions403
    | EndAccountSessions404
    | EndAccountSessions409
    | EndAccountSessions422
    | EndAccountSessions500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof endAccountSessions>>,
    TError,
    EndAccountSessionsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof endAccountSessions>>,
  TError,
  EndAccountSessionsMutationVariables,
  TContext
> => {
  const mutationKey = getEndAccountSessionsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof endAccountSessions>>,
    EndAccountSessionsMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return endAccountSessions(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type EndAccountSessionsMutationResult = NonNullable<
  Awaited<ReturnType<typeof endAccountSessions>>
>;

export type EndAccountSessionsMutationError = ErrorType<
  | EndAccountSessions401
  | EndAccountSessions403
  | EndAccountSessions404
  | EndAccountSessions409
  | EndAccountSessions422
  | EndAccountSessions500
>;
export type EndAccountSessionsMutationVariables = { id: string };

/**
 * @summary Sign an Account out everywhere
 */
export const useEndAccountSessions = <
  TError = ErrorType<
    | EndAccountSessions401
    | EndAccountSessions403
    | EndAccountSessions404
    | EndAccountSessions409
    | EndAccountSessions422
    | EndAccountSessions500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof endAccountSessions>>,
      TError,
      EndAccountSessionsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof endAccountSessions>>,
  TError,
  EndAccountSessionsMutationVariables,
  TContext
> => {
  return useMutation(getEndAccountSessionsMutationOptions(options), queryClient);
};
export const getListAccountSessionsUrl = (id: string) => {
  return `/api/v1/accounts/${id}/sessions`;
};

/**
 * @summary List the devices an Account is signed in on
 */
export const listAccountSessions = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentitySessionRows> => {
  return apiFetch<IdentitySessionRows>(getListAccountSessionsUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getListAccountSessionsQueryKey = (id: string) => {
  return [`/api/v1/accounts/${id}/sessions`] as const;
};

export const getListAccountSessionsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listAccountSessions>>,
  TError = ErrorType<
    | ListAccountSessions401
    | ListAccountSessions403
    | ListAccountSessions404
    | ListAccountSessions409
    | ListAccountSessions422
    | ListAccountSessions500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccountSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListAccountSessionsQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listAccountSessions>>> = ({ signal }) =>
    listAccountSessions(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccountSessions>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListAccountSessionsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listAccountSessions>>
>;
export type ListAccountSessionsSuspenseQueryError = ErrorType<
  | ListAccountSessions401
  | ListAccountSessions403
  | ListAccountSessions404
  | ListAccountSessions409
  | ListAccountSessions422
  | ListAccountSessions500
>;

export function useListAccountSessionsSuspense<
  TData = Awaited<ReturnType<typeof listAccountSessions>>,
  TError = ErrorType<
    | ListAccountSessions401
    | ListAccountSessions403
    | ListAccountSessions404
    | ListAccountSessions409
    | ListAccountSessions422
    | ListAccountSessions500
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccountSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListAccountSessionsSuspense<
  TData = Awaited<ReturnType<typeof listAccountSessions>>,
  TError = ErrorType<
    | ListAccountSessions401
    | ListAccountSessions403
    | ListAccountSessions404
    | ListAccountSessions409
    | ListAccountSessions422
    | ListAccountSessions500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccountSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListAccountSessionsSuspense<
  TData = Awaited<ReturnType<typeof listAccountSessions>>,
  TError = ErrorType<
    | ListAccountSessions401
    | ListAccountSessions403
    | ListAccountSessions404
    | ListAccountSessions409
    | ListAccountSessions422
    | ListAccountSessions500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccountSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the devices an Account is signed in on
 */

export function useListAccountSessionsSuspense<
  TData = Awaited<ReturnType<typeof listAccountSessions>>,
  TError = ErrorType<
    | ListAccountSessions401
    | ListAccountSessions403
    | ListAccountSessions404
    | ListAccountSessions409
    | ListAccountSessions422
    | ListAccountSessions500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listAccountSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListAccountSessionsSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getEndAccountSessionUrl = (id: string, session: string) => {
  return `/api/v1/accounts/${id}/sessions/${session}`;
};

/**
 * @summary Sign one of an Account's devices out
 */
export const endAccountSession = async (
  id: string,
  session: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getEndAccountSessionUrl(id, session), {
    ...options,
    method: "DELETE",
  });
};

export const getEndAccountSessionMutationKey = () => ["endAccountSession"] as const;

export const getEndAccountSessionMutationOptions = <
  TError = ErrorType<
    | EndAccountSession401
    | EndAccountSession403
    | EndAccountSession404
    | EndAccountSession409
    | EndAccountSession422
    | EndAccountSession500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof endAccountSession>>,
    TError,
    EndAccountSessionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof endAccountSession>>,
  TError,
  EndAccountSessionMutationVariables,
  TContext
> => {
  const mutationKey = getEndAccountSessionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof endAccountSession>>,
    EndAccountSessionMutationVariables
  > = (props) => {
    const { id, session } = props ?? {};

    return endAccountSession(id, session, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type EndAccountSessionMutationResult = NonNullable<
  Awaited<ReturnType<typeof endAccountSession>>
>;

export type EndAccountSessionMutationError = ErrorType<
  | EndAccountSession401
  | EndAccountSession403
  | EndAccountSession404
  | EndAccountSession409
  | EndAccountSession422
  | EndAccountSession500
>;
export type EndAccountSessionMutationVariables = { id: string; session: string };

/**
 * @summary Sign one of an Account's devices out
 */
export const useEndAccountSession = <
  TError = ErrorType<
    | EndAccountSession401
    | EndAccountSession403
    | EndAccountSession404
    | EndAccountSession409
    | EndAccountSession422
    | EndAccountSession500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof endAccountSession>>,
      TError,
      EndAccountSessionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof endAccountSession>>,
  TError,
  EndAccountSessionMutationVariables,
  TContext
> => {
  return useMutation(getEndAccountSessionMutationOptions(options), queryClient);
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
export const getGetEmailLinkByTokenUrl = (token: string) => {
  return `/api/v1/email-links/by-token/${token}`;
};

/**
 * @summary Get what a confirm link's page shows: the new email, or why it no longer works
 */
export const getEmailLinkByToken = async (
  token: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityLinkView> => {
  return apiFetch<IdentityLinkView>(getGetEmailLinkByTokenUrl(token), {
    ...options,
    method: "GET",
  });
};

export const getGetEmailLinkByTokenQueryKey = (token: string) => {
  return [`/api/v1/email-links/by-token/${token}`] as const;
};

export const getGetEmailLinkByTokenSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getEmailLinkByToken>>,
  TError = ErrorType<GetEmailLinkByToken422 | GetEmailLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getEmailLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetEmailLinkByTokenQueryKey(token);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getEmailLinkByToken>>> = ({ signal }) =>
    getEmailLinkByToken(token, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getEmailLinkByToken>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetEmailLinkByTokenSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getEmailLinkByToken>>
>;
export type GetEmailLinkByTokenSuspenseQueryError = ErrorType<
  GetEmailLinkByToken422 | GetEmailLinkByToken500
>;

export function useGetEmailLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getEmailLinkByToken>>,
  TError = ErrorType<GetEmailLinkByToken422 | GetEmailLinkByToken500>,
>(
  token: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getEmailLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetEmailLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getEmailLinkByToken>>,
  TError = ErrorType<GetEmailLinkByToken422 | GetEmailLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getEmailLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetEmailLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getEmailLinkByToken>>,
  TError = ErrorType<GetEmailLinkByToken422 | GetEmailLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getEmailLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get what a confirm link's page shows: the new email, or why it no longer works
 */

export function useGetEmailLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getEmailLinkByToken>>,
  TError = ErrorType<GetEmailLinkByToken422 | GetEmailLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getEmailLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetEmailLinkByTokenSuspenseQueryOptions(token, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getConfirmEmailUrl = (token: string) => {
  return `/api/v1/email-links/by-token/${token}/confirm`;
};

/**
 * @summary Change an Account's email to the one a confirm link proved
 */
export const confirmEmail = async (
  token: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getConfirmEmailUrl(token), {
    ...options,
    method: "POST",
  });
};

export const getConfirmEmailMutationKey = () => ["confirmEmail"] as const;

export const getConfirmEmailMutationOptions = <
  TError = ErrorType<
    ConfirmEmail403 | ConfirmEmail409 | ConfirmEmail410 | ConfirmEmail422 | ConfirmEmail500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof confirmEmail>>,
    TError,
    ConfirmEmailMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof confirmEmail>>,
  TError,
  ConfirmEmailMutationVariables,
  TContext
> => {
  const mutationKey = getConfirmEmailMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof confirmEmail>>,
    ConfirmEmailMutationVariables
  > = (props) => {
    const { token } = props ?? {};

    return confirmEmail(token, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ConfirmEmailMutationResult = NonNullable<Awaited<ReturnType<typeof confirmEmail>>>;

export type ConfirmEmailMutationError = ErrorType<
  ConfirmEmail403 | ConfirmEmail409 | ConfirmEmail410 | ConfirmEmail422 | ConfirmEmail500
>;
export type ConfirmEmailMutationVariables = { token: string };

/**
 * @summary Change an Account's email to the one a confirm link proved
 */
export const useConfirmEmail = <
  TError = ErrorType<
    ConfirmEmail403 | ConfirmEmail409 | ConfirmEmail410 | ConfirmEmail422 | ConfirmEmail500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof confirmEmail>>,
      TError,
      ConfirmEmailMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof confirmEmail>>,
  TError,
  ConfirmEmailMutationVariables,
  TContext
> => {
  return useMutation(getConfirmEmailMutationOptions(options), queryClient);
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
export const getListPasskeysUrl = () => {
  return `/api/v1/passkeys`;
};

/**
 * @summary List your passkeys at every address
 */
export const listPasskeys = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityPasskeyRows> => {
  return apiFetch<IdentityPasskeyRows>(getListPasskeysUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListPasskeysQueryKey = () => {
  return [`/api/v1/passkeys`] as const;
};

export const getListPasskeysSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listPasskeys>>,
  TError = ErrorType<ListPasskeys401 | ListPasskeys403 | ListPasskeys409 | ListPasskeys500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPasskeys>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListPasskeysQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listPasskeys>>> = ({ signal }) =>
    listPasskeys({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPasskeys>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListPasskeysSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listPasskeys>>>;
export type ListPasskeysSuspenseQueryError = ErrorType<
  ListPasskeys401 | ListPasskeys403 | ListPasskeys409 | ListPasskeys500
>;

export function useListPasskeysSuspense<
  TData = Awaited<ReturnType<typeof listPasskeys>>,
  TError = ErrorType<ListPasskeys401 | ListPasskeys403 | ListPasskeys409 | ListPasskeys500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPasskeys>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListPasskeysSuspense<
  TData = Awaited<ReturnType<typeof listPasskeys>>,
  TError = ErrorType<ListPasskeys401 | ListPasskeys403 | ListPasskeys409 | ListPasskeys500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPasskeys>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListPasskeysSuspense<
  TData = Awaited<ReturnType<typeof listPasskeys>>,
  TError = ErrorType<ListPasskeys401 | ListPasskeys403 | ListPasskeys409 | ListPasskeys500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPasskeys>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List your passkeys at every address
 */

export function useListPasskeysSuspense<
  TData = Awaited<ReturnType<typeof listPasskeys>>,
  TError = ErrorType<ListPasskeys401 | ListPasskeys403 | ListPasskeys409 | ListPasskeys500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listPasskeys>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListPasskeysSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getStartPasskeyRegistrationUrl = () => {
  return `/api/v1/passkeys/registration`;
};

/**
 * @summary Start adding a passkey for this address
 */
export const startPasskeyRegistration = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityCeremony> => {
  return apiFetch<IdentityCeremony>(getStartPasskeyRegistrationUrl(), {
    ...options,
    method: "POST",
  });
};

export const getStartPasskeyRegistrationMutationKey = () => ["startPasskeyRegistration"] as const;

export const getStartPasskeyRegistrationMutationOptions = <
  TError = ErrorType<
    | StartPasskeyRegistration401
    | StartPasskeyRegistration403
    | StartPasskeyRegistration409
    | StartPasskeyRegistration500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof startPasskeyRegistration>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof startPasskeyRegistration>>,
  TError,
  void,
  TContext
> => {
  const mutationKey = getStartPasskeyRegistrationMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof startPasskeyRegistration>>,
    void
  > = () => {
    return startPasskeyRegistration(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type StartPasskeyRegistrationMutationResult = NonNullable<
  Awaited<ReturnType<typeof startPasskeyRegistration>>
>;

export type StartPasskeyRegistrationMutationError = ErrorType<
  | StartPasskeyRegistration401
  | StartPasskeyRegistration403
  | StartPasskeyRegistration409
  | StartPasskeyRegistration500
>;

/**
 * @summary Start adding a passkey for this address
 */
export const useStartPasskeyRegistration = <
  TError = ErrorType<
    | StartPasskeyRegistration401
    | StartPasskeyRegistration403
    | StartPasskeyRegistration409
    | StartPasskeyRegistration500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof startPasskeyRegistration>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof startPasskeyRegistration>>,
  TError,
  void,
  TContext
> => {
  return useMutation(getStartPasskeyRegistrationMutationOptions(options), queryClient);
};
export const getFinishPasskeyRegistrationUrl = (ceremony: string) => {
  return `/api/v1/passkeys/registration/${ceremony}`;
};

/**
 * @summary Finish adding a passkey with the browser's answer
 */
export const finishPasskeyRegistration = async (
  ceremony: string,
  finishPasskeyRegistrationBody: unknown,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityPasskey> => {
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
  return apiFetch<IdentityPasskey>(getFinishPasskeyRegistrationUrl(ceremony), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(finishPasskeyRegistrationBody),
  });
};

export const getFinishPasskeyRegistrationMutationKey = () => ["finishPasskeyRegistration"] as const;

export const getFinishPasskeyRegistrationMutationOptions = <
  TError = ErrorType<
    | FinishPasskeyRegistration401
    | FinishPasskeyRegistration403
    | FinishPasskeyRegistration409
    | FinishPasskeyRegistration422
    | FinishPasskeyRegistration500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof finishPasskeyRegistration>>,
    TError,
    FinishPasskeyRegistrationMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof finishPasskeyRegistration>>,
  TError,
  FinishPasskeyRegistrationMutationVariables,
  TContext
> => {
  const mutationKey = getFinishPasskeyRegistrationMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof finishPasskeyRegistration>>,
    FinishPasskeyRegistrationMutationVariables
  > = (props) => {
    const { ceremony, data } = props ?? {};

    return finishPasskeyRegistration(ceremony, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type FinishPasskeyRegistrationMutationResult = NonNullable<
  Awaited<ReturnType<typeof finishPasskeyRegistration>>
>;
export type FinishPasskeyRegistrationMutationBody = unknown;
export type FinishPasskeyRegistrationMutationError = ErrorType<
  | FinishPasskeyRegistration401
  | FinishPasskeyRegistration403
  | FinishPasskeyRegistration409
  | FinishPasskeyRegistration422
  | FinishPasskeyRegistration500
>;
export type FinishPasskeyRegistrationMutationVariables = { ceremony: string; data: unknown };

/**
 * @summary Finish adding a passkey with the browser's answer
 */
export const useFinishPasskeyRegistration = <
  TError = ErrorType<
    | FinishPasskeyRegistration401
    | FinishPasskeyRegistration403
    | FinishPasskeyRegistration409
    | FinishPasskeyRegistration422
    | FinishPasskeyRegistration500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof finishPasskeyRegistration>>,
      TError,
      FinishPasskeyRegistrationMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof finishPasskeyRegistration>>,
  TError,
  FinishPasskeyRegistrationMutationVariables,
  TContext
> => {
  return useMutation(getFinishPasskeyRegistrationMutationOptions(options), queryClient);
};
export const getDeletePasskeyUrl = (id: string) => {
  return `/api/v1/passkeys/${id}`;
};

/**
 * @summary Delete one of your passkeys
 */
export const deletePasskey = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeletePasskeyUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getDeletePasskeyMutationKey = () => ["deletePasskey"] as const;

export const getDeletePasskeyMutationOptions = <
  TError = ErrorType<
    | DeletePasskey401
    | DeletePasskey403
    | DeletePasskey404
    | DeletePasskey409
    | DeletePasskey422
    | DeletePasskey500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deletePasskey>>,
    TError,
    DeletePasskeyMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deletePasskey>>,
  TError,
  DeletePasskeyMutationVariables,
  TContext
> => {
  const mutationKey = getDeletePasskeyMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deletePasskey>>,
    DeletePasskeyMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return deletePasskey(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeletePasskeyMutationResult = NonNullable<Awaited<ReturnType<typeof deletePasskey>>>;

export type DeletePasskeyMutationError = ErrorType<
  | DeletePasskey401
  | DeletePasskey403
  | DeletePasskey404
  | DeletePasskey409
  | DeletePasskey422
  | DeletePasskey500
>;
export type DeletePasskeyMutationVariables = { id: string };

/**
 * @summary Delete one of your passkeys
 */
export const useDeletePasskey = <
  TError = ErrorType<
    | DeletePasskey401
    | DeletePasskey403
    | DeletePasskey404
    | DeletePasskey409
    | DeletePasskey422
    | DeletePasskey500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deletePasskey>>,
      TError,
      DeletePasskeyMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deletePasskey>>,
  TError,
  DeletePasskeyMutationVariables,
  TContext
> => {
  return useMutation(getDeletePasskeyMutationOptions(options), queryClient);
};
export const getUpdatePasskeyUrl = (id: string) => {
  return `/api/v1/passkeys/${id}`;
};

/**
 * @summary Rename one of your passkeys
 */
export const updatePasskey = async (
  id: string,
  passkeyRenameInputBody: PasskeyRenameInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityPasskey> => {
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
  return apiFetch<IdentityPasskey>(getUpdatePasskeyUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(passkeyRenameInputBody),
  });
};

export const getUpdatePasskeyMutationKey = () => ["updatePasskey"] as const;

export const getUpdatePasskeyMutationOptions = <
  TError = ErrorType<
    | UpdatePasskey401
    | UpdatePasskey403
    | UpdatePasskey404
    | UpdatePasskey409
    | UpdatePasskey422
    | UpdatePasskey500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updatePasskey>>,
    TError,
    UpdatePasskeyMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updatePasskey>>,
  TError,
  UpdatePasskeyMutationVariables,
  TContext
> => {
  const mutationKey = getUpdatePasskeyMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updatePasskey>>,
    UpdatePasskeyMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return updatePasskey(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdatePasskeyMutationResult = NonNullable<Awaited<ReturnType<typeof updatePasskey>>>;
export type UpdatePasskeyMutationBody = PasskeyRenameInputBody;
export type UpdatePasskeyMutationError = ErrorType<
  | UpdatePasskey401
  | UpdatePasskey403
  | UpdatePasskey404
  | UpdatePasskey409
  | UpdatePasskey422
  | UpdatePasskey500
>;
export type UpdatePasskeyMutationVariables = { id: string; data: PasskeyRenameInputBody };

/**
 * @summary Rename one of your passkeys
 */
export const useUpdatePasskey = <
  TError = ErrorType<
    | UpdatePasskey401
    | UpdatePasskey403
    | UpdatePasskey404
    | UpdatePasskey409
    | UpdatePasskey422
    | UpdatePasskey500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updatePasskey>>,
      TError,
      UpdatePasskeyMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updatePasskey>>,
  TError,
  UpdatePasskeyMutationVariables,
  TContext
> => {
  return useMutation(getUpdatePasskeyMutationOptions(options), queryClient);
};
export const getGetResetLinkByTokenUrl = (token: string) => {
  return `/api/v1/reset-links/by-token/${token}`;
};

/**
 * @summary Get what a Reset link's page shows: who it is from, or why it no longer works
 */
export const getResetLinkByToken = async (
  token: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityLinkView> => {
  return apiFetch<IdentityLinkView>(getGetResetLinkByTokenUrl(token), {
    ...options,
    method: "GET",
  });
};

export const getGetResetLinkByTokenQueryKey = (token: string) => {
  return [`/api/v1/reset-links/by-token/${token}`] as const;
};

export const getGetResetLinkByTokenSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getResetLinkByToken>>,
  TError = ErrorType<GetResetLinkByToken422 | GetResetLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getResetLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetResetLinkByTokenQueryKey(token);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getResetLinkByToken>>> = ({ signal }) =>
    getResetLinkByToken(token, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getResetLinkByToken>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetResetLinkByTokenSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getResetLinkByToken>>
>;
export type GetResetLinkByTokenSuspenseQueryError = ErrorType<
  GetResetLinkByToken422 | GetResetLinkByToken500
>;

export function useGetResetLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getResetLinkByToken>>,
  TError = ErrorType<GetResetLinkByToken422 | GetResetLinkByToken500>,
>(
  token: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getResetLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetResetLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getResetLinkByToken>>,
  TError = ErrorType<GetResetLinkByToken422 | GetResetLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getResetLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetResetLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getResetLinkByToken>>,
  TError = ErrorType<GetResetLinkByToken422 | GetResetLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getResetLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get what a Reset link's page shows: who it is from, or why it no longer works
 */

export function useGetResetLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getResetLinkByToken>>,
  TError = ErrorType<GetResetLinkByToken422 | GetResetLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getResetLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetResetLinkByTokenSuspenseQueryOptions(token, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getRedeemResetLinkUrl = (token: string) => {
  return `/api/v1/reset-links/by-token/${token}/pin`;
};

/**
 * @summary Set a new PIN with a Reset link, ending every session, and sign in
 */
export const redeemResetLink = async (
  token: string,
  newPINInputBody: NewPINInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentitySignedIn> => {
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
  return apiFetch<IdentitySignedIn>(getRedeemResetLinkUrl(token), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(newPINInputBody),
  });
};

export const getRedeemResetLinkMutationKey = () => ["redeemResetLink"] as const;

export const getRedeemResetLinkMutationOptions = <
  TError = ErrorType<
    RedeemResetLink403 | RedeemResetLink410 | RedeemResetLink422 | RedeemResetLink500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof redeemResetLink>>,
    TError,
    RedeemResetLinkMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof redeemResetLink>>,
  TError,
  RedeemResetLinkMutationVariables,
  TContext
> => {
  const mutationKey = getRedeemResetLinkMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof redeemResetLink>>,
    RedeemResetLinkMutationVariables
  > = (props) => {
    const { token, data } = props ?? {};

    return redeemResetLink(token, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RedeemResetLinkMutationResult = NonNullable<
  Awaited<ReturnType<typeof redeemResetLink>>
>;
export type RedeemResetLinkMutationBody = NewPINInputBody;
export type RedeemResetLinkMutationError = ErrorType<
  RedeemResetLink403 | RedeemResetLink410 | RedeemResetLink422 | RedeemResetLink500
>;
export type RedeemResetLinkMutationVariables = { token: string; data: NewPINInputBody };

/**
 * @summary Set a new PIN with a Reset link, ending every session, and sign in
 */
export const useRedeemResetLink = <
  TError = ErrorType<
    RedeemResetLink403 | RedeemResetLink410 | RedeemResetLink422 | RedeemResetLink500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof redeemResetLink>>,
      TError,
      RedeemResetLinkMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof redeemResetLink>>,
  TError,
  RedeemResetLinkMutationVariables,
  TContext
> => {
  return useMutation(getRedeemResetLinkMutationOptions(options), queryClient);
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
): Promise<IdentitySignedIn> => {
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
  return apiFetch<IdentitySignedIn>(getCreateSessionUrl(), {
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
export const getEndOtherSessionsUrl = (params: EndOtherSessionsParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/sessions?${stringifiedParams}`
    : `/api/v1/sessions`;
};

/**
 * @summary Sign out every device but this one
 */
export const endOtherSessions = async (
  params: EndOtherSessionsParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getEndOtherSessionsUrl(params), {
    ...options,
    method: "DELETE",
  });
};

export const getEndOtherSessionsMutationKey = () => ["endOtherSessions"] as const;

export const getEndOtherSessionsMutationOptions = <
  TError = ErrorType<
    | EndOtherSessions401
    | EndOtherSessions403
    | EndOtherSessions409
    | EndOtherSessions422
    | EndOtherSessions500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof endOtherSessions>>,
    TError,
    EndOtherSessionsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof endOtherSessions>>,
  TError,
  EndOtherSessionsMutationVariables,
  TContext
> => {
  const mutationKey = getEndOtherSessionsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof endOtherSessions>>,
    EndOtherSessionsMutationVariables
  > = (props) => {
    const { params } = props ?? {};

    return endOtherSessions(params, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type EndOtherSessionsMutationResult = NonNullable<
  Awaited<ReturnType<typeof endOtherSessions>>
>;

export type EndOtherSessionsMutationError = ErrorType<
  | EndOtherSessions401
  | EndOtherSessions403
  | EndOtherSessions409
  | EndOtherSessions422
  | EndOtherSessions500
>;
export type EndOtherSessionsMutationVariables = { params: EndOtherSessionsParams };

/**
 * @summary Sign out every device but this one
 */
export const useEndOtherSessions = <
  TError = ErrorType<
    | EndOtherSessions401
    | EndOtherSessions403
    | EndOtherSessions409
    | EndOtherSessions422
    | EndOtherSessions500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof endOtherSessions>>,
      TError,
      EndOtherSessionsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof endOtherSessions>>,
  TError,
  EndOtherSessionsMutationVariables,
  TContext
> => {
  return useMutation(getEndOtherSessionsMutationOptions(options), queryClient);
};
export const getListSessionsUrl = () => {
  return `/api/v1/sessions`;
};

/**
 * @summary List the devices your Account is signed in on
 */
export const listSessions = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentitySessionRows> => {
  return apiFetch<IdentitySessionRows>(getListSessionsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListSessionsQueryKey = () => {
  return [`/api/v1/sessions`] as const;
};

export const getListSessionsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listSessions>>,
  TError = ErrorType<ListSessions401 | ListSessions403 | ListSessions409 | ListSessions500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSessions>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListSessionsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listSessions>>> = ({ signal }) =>
    listSessions({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSessions>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListSessionsSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listSessions>>>;
export type ListSessionsSuspenseQueryError = ErrorType<
  ListSessions401 | ListSessions403 | ListSessions409 | ListSessions500
>;

export function useListSessionsSuspense<
  TData = Awaited<ReturnType<typeof listSessions>>,
  TError = ErrorType<ListSessions401 | ListSessions403 | ListSessions409 | ListSessions500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListSessionsSuspense<
  TData = Awaited<ReturnType<typeof listSessions>>,
  TError = ErrorType<ListSessions401 | ListSessions403 | ListSessions409 | ListSessions500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListSessionsSuspense<
  TData = Awaited<ReturnType<typeof listSessions>>,
  TError = ErrorType<ListSessions401 | ListSessions403 | ListSessions409 | ListSessions500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the devices your Account is signed in on
 */

export function useListSessionsSuspense<
  TData = Awaited<ReturnType<typeof listSessions>>,
  TError = ErrorType<ListSessions401 | ListSessions403 | ListSessions409 | ListSessions500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSessions>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListSessionsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getEndSessionUrl = (id: string) => {
  return `/api/v1/sessions/${id}`;
};

/**
 * @summary Sign one of your devices out
 */
export const endSession = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getEndSessionUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getEndSessionMutationKey = () => ["endSession"] as const;

export const getEndSessionMutationOptions = <
  TError = ErrorType<
    EndSession401 | EndSession403 | EndSession404 | EndSession409 | EndSession422 | EndSession500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof endSession>>,
    TError,
    EndSessionMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof endSession>>,
  TError,
  EndSessionMutationVariables,
  TContext
> => {
  const mutationKey = getEndSessionMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof endSession>>,
    EndSessionMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return endSession(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type EndSessionMutationResult = NonNullable<Awaited<ReturnType<typeof endSession>>>;

export type EndSessionMutationError = ErrorType<
  EndSession401 | EndSession403 | EndSession404 | EndSession409 | EndSession422 | EndSession500
>;
export type EndSessionMutationVariables = { id: string };

/**
 * @summary Sign one of your devices out
 */
export const useEndSession = <
  TError = ErrorType<
    EndSession401 | EndSession403 | EndSession404 | EndSession409 | EndSession422 | EndSession500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof endSession>>,
      TError,
      EndSessionMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof endSession>>,
  TError,
  EndSessionMutationVariables,
  TContext
> => {
  return useMutation(getEndSessionMutationOptions(options), queryClient);
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
export const getListSignupLinksUrl = () => {
  return `/api/v1/signup-links`;
};

/**
 * @summary List the Signup links nobody has used yet
 */
export const listSignupLinks = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityPendingLinks> => {
  return apiFetch<IdentityPendingLinks>(getListSignupLinksUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListSignupLinksQueryKey = () => {
  return [`/api/v1/signup-links`] as const;
};

export const getListSignupLinksSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listSignupLinks>>,
  TError = ErrorType<
    ListSignupLinks401 | ListSignupLinks403 | ListSignupLinks409 | ListSignupLinks500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSignupLinks>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListSignupLinksQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listSignupLinks>>> = ({ signal }) =>
    listSignupLinks({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSignupLinks>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListSignupLinksSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listSignupLinks>>
>;
export type ListSignupLinksSuspenseQueryError = ErrorType<
  ListSignupLinks401 | ListSignupLinks403 | ListSignupLinks409 | ListSignupLinks500
>;

export function useListSignupLinksSuspense<
  TData = Awaited<ReturnType<typeof listSignupLinks>>,
  TError = ErrorType<
    ListSignupLinks401 | ListSignupLinks403 | ListSignupLinks409 | ListSignupLinks500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSignupLinks>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListSignupLinksSuspense<
  TData = Awaited<ReturnType<typeof listSignupLinks>>,
  TError = ErrorType<
    ListSignupLinks401 | ListSignupLinks403 | ListSignupLinks409 | ListSignupLinks500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSignupLinks>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListSignupLinksSuspense<
  TData = Awaited<ReturnType<typeof listSignupLinks>>,
  TError = ErrorType<
    ListSignupLinks401 | ListSignupLinks403 | ListSignupLinks409 | ListSignupLinks500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSignupLinks>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Signup links nobody has used yet
 */

export function useListSignupLinksSuspense<
  TData = Awaited<ReturnType<typeof listSignupLinks>>,
  TError = ErrorType<
    ListSignupLinks401 | ListSignupLinks403 | ListSignupLinks409 | ListSignupLinks500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSignupLinks>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListSignupLinksSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCreateSignupLinkUrl = () => {
  return `/api/v1/signup-links`;
};

/**
 * @summary Issue a Signup link for an email, email it, and answer it to copy
 */
export const createSignupLink = async (
  newSignupLinkInputBody: NewSignupLinkInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityIssuedSignupLink> => {
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
  return apiFetch<IdentityIssuedSignupLink>(getCreateSignupLinkUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(newSignupLinkInputBody),
  });
};

export const getCreateSignupLinkMutationKey = () => ["createSignupLink"] as const;

export const getCreateSignupLinkMutationOptions = <
  TError = ErrorType<
    | CreateSignupLink401
    | CreateSignupLink403
    | CreateSignupLink409
    | CreateSignupLink422
    | CreateSignupLink500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof createSignupLink>>,
    TError,
    CreateSignupLinkMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof createSignupLink>>,
  TError,
  CreateSignupLinkMutationVariables,
  TContext
> => {
  const mutationKey = getCreateSignupLinkMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof createSignupLink>>,
    CreateSignupLinkMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return createSignupLink(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CreateSignupLinkMutationResult = NonNullable<
  Awaited<ReturnType<typeof createSignupLink>>
>;
export type CreateSignupLinkMutationBody = NewSignupLinkInputBody;
export type CreateSignupLinkMutationError = ErrorType<
  | CreateSignupLink401
  | CreateSignupLink403
  | CreateSignupLink409
  | CreateSignupLink422
  | CreateSignupLink500
>;
export type CreateSignupLinkMutationVariables = { data: NewSignupLinkInputBody };

/**
 * @summary Issue a Signup link for an email, email it, and answer it to copy
 */
export const useCreateSignupLink = <
  TError = ErrorType<
    | CreateSignupLink401
    | CreateSignupLink403
    | CreateSignupLink409
    | CreateSignupLink422
    | CreateSignupLink500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof createSignupLink>>,
      TError,
      CreateSignupLinkMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof createSignupLink>>,
  TError,
  CreateSignupLinkMutationVariables,
  TContext
> => {
  return useMutation(getCreateSignupLinkMutationOptions(options), queryClient);
};
export const getGetSignupLinkByTokenUrl = (token: string) => {
  return `/api/v1/signup-links/by-token/${token}`;
};

/**
 * @summary Get what a Signup link's page shows: who it is from, or why it no longer works
 */
export const getSignupLinkByToken = async (
  token: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityLinkView> => {
  return apiFetch<IdentityLinkView>(getGetSignupLinkByTokenUrl(token), {
    ...options,
    method: "GET",
  });
};

export const getGetSignupLinkByTokenQueryKey = (token: string) => {
  return [`/api/v1/signup-links/by-token/${token}`] as const;
};

export const getGetSignupLinkByTokenSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSignupLinkByToken>>,
  TError = ErrorType<GetSignupLinkByToken422 | GetSignupLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSignupLinkByTokenQueryKey(token);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSignupLinkByToken>>> = ({ signal }) =>
    getSignupLinkByToken(token, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLinkByToken>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSignupLinkByTokenSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getSignupLinkByToken>>
>;
export type GetSignupLinkByTokenSuspenseQueryError = ErrorType<
  GetSignupLinkByToken422 | GetSignupLinkByToken500
>;

export function useGetSignupLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getSignupLinkByToken>>,
  TError = ErrorType<GetSignupLinkByToken422 | GetSignupLinkByToken500>,
>(
  token: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSignupLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getSignupLinkByToken>>,
  TError = ErrorType<GetSignupLinkByToken422 | GetSignupLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSignupLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getSignupLinkByToken>>,
  TError = ErrorType<GetSignupLinkByToken422 | GetSignupLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get what a Signup link's page shows: who it is from, or why it no longer works
 */

export function useGetSignupLinkByTokenSuspense<
  TData = Awaited<ReturnType<typeof getSignupLinkByToken>>,
  TError = ErrorType<GetSignupLinkByToken422 | GetSignupLinkByToken500>,
>(
  token: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLinkByToken>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSignupLinkByTokenSuspenseQueryOptions(token, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getRedeemSignupLinkUrl = (token: string) => {
  return `/api/v1/signup-links/by-token/${token}/account`;
};

/**
 * @summary Make the Member's Account with a Signup link and sign in
 */
export const redeemSignupLink = async (
  token: string,
  joinInputBody: JoinInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentitySignedIn> => {
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
  return apiFetch<IdentitySignedIn>(getRedeemSignupLinkUrl(token), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(joinInputBody),
  });
};

export const getRedeemSignupLinkMutationKey = () => ["redeemSignupLink"] as const;

export const getRedeemSignupLinkMutationOptions = <
  TError = ErrorType<
    RedeemSignupLink409 | RedeemSignupLink410 | RedeemSignupLink422 | RedeemSignupLink500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof redeemSignupLink>>,
    TError,
    RedeemSignupLinkMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof redeemSignupLink>>,
  TError,
  RedeemSignupLinkMutationVariables,
  TContext
> => {
  const mutationKey = getRedeemSignupLinkMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof redeemSignupLink>>,
    RedeemSignupLinkMutationVariables
  > = (props) => {
    const { token, data } = props ?? {};

    return redeemSignupLink(token, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RedeemSignupLinkMutationResult = NonNullable<
  Awaited<ReturnType<typeof redeemSignupLink>>
>;
export type RedeemSignupLinkMutationBody = JoinInputBody;
export type RedeemSignupLinkMutationError = ErrorType<
  RedeemSignupLink409 | RedeemSignupLink410 | RedeemSignupLink422 | RedeemSignupLink500
>;
export type RedeemSignupLinkMutationVariables = { token: string; data: JoinInputBody };

/**
 * @summary Make the Member's Account with a Signup link and sign in
 */
export const useRedeemSignupLink = <
  TError = ErrorType<
    RedeemSignupLink409 | RedeemSignupLink410 | RedeemSignupLink422 | RedeemSignupLink500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof redeemSignupLink>>,
      TError,
      RedeemSignupLinkMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof redeemSignupLink>>,
  TError,
  RedeemSignupLinkMutationVariables,
  TContext
> => {
  return useMutation(getRedeemSignupLinkMutationOptions(options), queryClient);
};
export const getDeleteSignupLinkUrl = (id: string) => {
  return `/api/v1/signup-links/${id}`;
};

/**
 * @summary Cancel a pending Signup link
 */
export const deleteSignupLink = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getDeleteSignupLinkUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteSignupLinkMutationKey = () => ["deleteSignupLink"] as const;

export const getDeleteSignupLinkMutationOptions = <
  TError = ErrorType<
    | DeleteSignupLink401
    | DeleteSignupLink403
    | DeleteSignupLink404
    | DeleteSignupLink409
    | DeleteSignupLink422
    | DeleteSignupLink500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteSignupLink>>,
    TError,
    DeleteSignupLinkMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteSignupLink>>,
  TError,
  DeleteSignupLinkMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteSignupLinkMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteSignupLink>>,
    DeleteSignupLinkMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return deleteSignupLink(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteSignupLinkMutationResult = NonNullable<
  Awaited<ReturnType<typeof deleteSignupLink>>
>;

export type DeleteSignupLinkMutationError = ErrorType<
  | DeleteSignupLink401
  | DeleteSignupLink403
  | DeleteSignupLink404
  | DeleteSignupLink409
  | DeleteSignupLink422
  | DeleteSignupLink500
>;
export type DeleteSignupLinkMutationVariables = { id: string };

/**
 * @summary Cancel a pending Signup link
 */
export const useDeleteSignupLink = <
  TError = ErrorType<
    | DeleteSignupLink401
    | DeleteSignupLink403
    | DeleteSignupLink404
    | DeleteSignupLink409
    | DeleteSignupLink422
    | DeleteSignupLink500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteSignupLink>>,
      TError,
      DeleteSignupLinkMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteSignupLink>>,
  TError,
  DeleteSignupLinkMutationVariables,
  TContext
> => {
  return useMutation(getDeleteSignupLinkMutationOptions(options), queryClient);
};
export const getGetSignupLinkUrl = (id: string) => {
  return `/api/v1/signup-links/${id}`;
};

/**
 * @summary Get a pending Signup link with its URL, to copy again
 */
export const getSignupLink = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentitySignupLinkURL> => {
  return apiFetch<IdentitySignupLinkURL>(getGetSignupLinkUrl(id), {
    ...options,
    method: "GET",
  });
};

export const getGetSignupLinkQueryKey = (id: string) => {
  return [`/api/v1/signup-links/${id}`] as const;
};

export const getGetSignupLinkSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSignupLink>>,
  TError = ErrorType<
    | GetSignupLink401
    | GetSignupLink403
    | GetSignupLink404
    | GetSignupLink409
    | GetSignupLink422
    | GetSignupLink500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLink>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSignupLinkQueryKey(id);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSignupLink>>> = ({ signal }) =>
    getSignupLink(id, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLink>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSignupLinkSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getSignupLink>>
>;
export type GetSignupLinkSuspenseQueryError = ErrorType<
  | GetSignupLink401
  | GetSignupLink403
  | GetSignupLink404
  | GetSignupLink409
  | GetSignupLink422
  | GetSignupLink500
>;

export function useGetSignupLinkSuspense<
  TData = Awaited<ReturnType<typeof getSignupLink>>,
  TError = ErrorType<
    | GetSignupLink401
    | GetSignupLink403
    | GetSignupLink404
    | GetSignupLink409
    | GetSignupLink422
    | GetSignupLink500
  >,
>(
  id: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLink>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSignupLinkSuspense<
  TData = Awaited<ReturnType<typeof getSignupLink>>,
  TError = ErrorType<
    | GetSignupLink401
    | GetSignupLink403
    | GetSignupLink404
    | GetSignupLink409
    | GetSignupLink422
    | GetSignupLink500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLink>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSignupLinkSuspense<
  TData = Awaited<ReturnType<typeof getSignupLink>>,
  TError = ErrorType<
    | GetSignupLink401
    | GetSignupLink403
    | GetSignupLink404
    | GetSignupLink409
    | GetSignupLink422
    | GetSignupLink500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLink>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a pending Signup link with its URL, to copy again
 */

export function useGetSignupLinkSuspense<
  TData = Awaited<ReturnType<typeof getSignupLink>>,
  TError = ErrorType<
    | GetSignupLink401
    | GetSignupLink403
    | GetSignupLink404
    | GetSignupLink409
    | GetSignupLink422
    | GetSignupLink500
  >,
>(
  id: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSignupLink>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSignupLinkSuspenseQueryOptions(id, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getResendSignupLinkUrl = (id: string) => {
  return `/api/v1/signup-links/${id}/resend`;
};

/**
 * @summary Send a pending Signup link again as a new link; the old one stops working
 */
export const resendSignupLink = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<IdentityIssuedSignupLink> => {
  return apiFetch<IdentityIssuedSignupLink>(getResendSignupLinkUrl(id), {
    ...options,
    method: "POST",
  });
};

export const getResendSignupLinkMutationKey = () => ["resendSignupLink"] as const;

export const getResendSignupLinkMutationOptions = <
  TError = ErrorType<
    | ResendSignupLink401
    | ResendSignupLink403
    | ResendSignupLink404
    | ResendSignupLink409
    | ResendSignupLink422
    | ResendSignupLink500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof resendSignupLink>>,
    TError,
    ResendSignupLinkMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof resendSignupLink>>,
  TError,
  ResendSignupLinkMutationVariables,
  TContext
> => {
  const mutationKey = getResendSignupLinkMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof resendSignupLink>>,
    ResendSignupLinkMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return resendSignupLink(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ResendSignupLinkMutationResult = NonNullable<
  Awaited<ReturnType<typeof resendSignupLink>>
>;

export type ResendSignupLinkMutationError = ErrorType<
  | ResendSignupLink401
  | ResendSignupLink403
  | ResendSignupLink404
  | ResendSignupLink409
  | ResendSignupLink422
  | ResendSignupLink500
>;
export type ResendSignupLinkMutationVariables = { id: string };

/**
 * @summary Send a pending Signup link again as a new link; the old one stops working
 */
export const useResendSignupLink = <
  TError = ErrorType<
    | ResendSignupLink401
    | ResendSignupLink403
    | ResendSignupLink404
    | ResendSignupLink409
    | ResendSignupLink422
    | ResendSignupLink500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof resendSignupLink>>,
      TError,
      ResendSignupLinkMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof resendSignupLink>>,
  TError,
  ResendSignupLinkMutationVariables,
  TContext
> => {
  return useMutation(getResendSignupLinkMutationOptions(options), queryClient);
};
