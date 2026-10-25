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
  GetMailSettings401,
  GetMailSettings403,
  GetMailSettings409,
  GetMailSettings500,
  MailOutcome,
  MailSettings,
  MailSettingsPatch,
  SendTestMail401,
  SendTestMail403,
  SendTestMail409,
  SendTestMail422,
  SendTestMail500,
  UpdateMailSettings401,
  UpdateMailSettings403,
  UpdateMailSettings409,
  UpdateMailSettings422,
  UpdateMailSettings500,
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

export const getGetMailSettingsUrl = () => {
  return `/api/v1/mail/settings`;
};

/**
 * @summary Get the SMTP server every email goes through
 */
export const getMailSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<MailSettings> => {
  return apiFetch<MailSettings>(getGetMailSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetMailSettingsQueryKey = () => {
  return [`/api/v1/mail/settings`] as const;
};

export const getGetMailSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getMailSettings>>,
  TError = ErrorType<
    GetMailSettings401 | GetMailSettings403 | GetMailSettings409 | GetMailSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMailSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetMailSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getMailSettings>>> = ({ signal }) =>
    getMailSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMailSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetMailSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getMailSettings>>
>;
export type GetMailSettingsSuspenseQueryError = ErrorType<
  GetMailSettings401 | GetMailSettings403 | GetMailSettings409 | GetMailSettings500
>;

export function useGetMailSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMailSettings>>,
  TError = ErrorType<
    GetMailSettings401 | GetMailSettings403 | GetMailSettings409 | GetMailSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMailSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMailSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMailSettings>>,
  TError = ErrorType<
    GetMailSettings401 | GetMailSettings403 | GetMailSettings409 | GetMailSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMailSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetMailSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMailSettings>>,
  TError = ErrorType<
    GetMailSettings401 | GetMailSettings403 | GetMailSettings409 | GetMailSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMailSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the SMTP server every email goes through
 */

export function useGetMailSettingsSuspense<
  TData = Awaited<ReturnType<typeof getMailSettings>>,
  TError = ErrorType<
    GetMailSettings401 | GetMailSettings403 | GetMailSettings409 | GetMailSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getMailSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetMailSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateMailSettingsUrl = () => {
  return `/api/v1/mail/settings`;
};

/**
 * @summary Change the SMTP server every email goes through
 */
export const updateMailSettings = async (
  mailSettingsPatch: MailSettingsPatch,
  options?: Parameters<typeof apiFetch>[1],
): Promise<MailSettings> => {
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
  return apiFetch<MailSettings>(getUpdateMailSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(mailSettingsPatch),
  });
};

export const getUpdateMailSettingsMutationKey = () => ["updateMailSettings"] as const;

export const getUpdateMailSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateMailSettings401
    | UpdateMailSettings403
    | UpdateMailSettings409
    | UpdateMailSettings422
    | UpdateMailSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateMailSettings>>,
    TError,
    UpdateMailSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateMailSettings>>,
  TError,
  UpdateMailSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateMailSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateMailSettings>>,
    UpdateMailSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateMailSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateMailSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateMailSettings>>
>;
export type UpdateMailSettingsMutationBody = MailSettingsPatch;
export type UpdateMailSettingsMutationError = ErrorType<
  | UpdateMailSettings401
  | UpdateMailSettings403
  | UpdateMailSettings409
  | UpdateMailSettings422
  | UpdateMailSettings500
>;
export type UpdateMailSettingsMutationVariables = { data: MailSettingsPatch };

/**
 * @summary Change the SMTP server every email goes through
 */
export const useUpdateMailSettings = <
  TError = ErrorType<
    | UpdateMailSettings401
    | UpdateMailSettings403
    | UpdateMailSettings409
    | UpdateMailSettings422
    | UpdateMailSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateMailSettings>>,
      TError,
      UpdateMailSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateMailSettings>>,
  TError,
  UpdateMailSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateMailSettingsMutationOptions(options), queryClient);
};
export const getSendTestMailUrl = () => {
  return `/api/v1/mail/test`;
};

/**
 * @summary Send a test email to your own address and return the mail server's reply
 */
export const sendTestMail = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<MailOutcome> => {
  return apiFetch<MailOutcome>(getSendTestMailUrl(), {
    ...options,
    method: "POST",
  });
};

export const getSendTestMailMutationKey = () => ["sendTestMail"] as const;

export const getSendTestMailMutationOptions = <
  TError = ErrorType<
    SendTestMail401 | SendTestMail403 | SendTestMail409 | SendTestMail422 | SendTestMail500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<Awaited<ReturnType<typeof sendTestMail>>, TError, void, TContext>;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof sendTestMail>>, TError, void, TContext> => {
  const mutationKey = getSendTestMailMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof sendTestMail>>, void> = () => {
    return sendTestMail(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SendTestMailMutationResult = NonNullable<Awaited<ReturnType<typeof sendTestMail>>>;

export type SendTestMailMutationError = ErrorType<
  SendTestMail401 | SendTestMail403 | SendTestMail409 | SendTestMail422 | SendTestMail500
>;

/**
 * @summary Send a test email to your own address and return the mail server's reply
 */
export const useSendTestMail = <
  TError = ErrorType<
    SendTestMail401 | SendTestMail403 | SendTestMail409 | SendTestMail422 | SendTestMail500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<Awaited<ReturnType<typeof sendTestMail>>, TError, void, TContext>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof sendTestMail>>, TError, void, TContext> => {
  return useMutation(getSendTestMailMutationOptions(options), queryClient);
};
