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
  AddBackupDestination401,
  AddBackupDestination403,
  AddBackupDestination404,
  AddBackupDestination409,
  AddBackupDestination422,
  AddBackupDestination500,
  AddBackupDestination503,
  BackupsAdded,
  BackupsDestinationList,
  BackupsNewDestination,
  ListBackupDestinations401,
  ListBackupDestinations403,
  ListBackupDestinations409,
  ListBackupDestinations500,
  RemoveBackupDestination401,
  RemoveBackupDestination403,
  RemoveBackupDestination404,
  RemoveBackupDestination409,
  RemoveBackupDestination422,
  RemoveBackupDestination500,
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

export const getListBackupDestinationsUrl = () => {
  return `/api/v1/backup-destinations`;
};

/**
 * @summary List the Backup destinations and their copies
 */
export const listBackupDestinations = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<BackupsDestinationList> => {
  return apiFetch<BackupsDestinationList>(getListBackupDestinationsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListBackupDestinationsQueryKey = () => {
  return [`/api/v1/backup-destinations`] as const;
};

export const getListBackupDestinationsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listBackupDestinations>>,
  TError = ErrorType<
    | ListBackupDestinations401
    | ListBackupDestinations403
    | ListBackupDestinations409
    | ListBackupDestinations500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackupDestinations>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListBackupDestinationsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listBackupDestinations>>> = ({ signal }) =>
    listBackupDestinations({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listBackupDestinations>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListBackupDestinationsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listBackupDestinations>>
>;
export type ListBackupDestinationsSuspenseQueryError = ErrorType<
  | ListBackupDestinations401
  | ListBackupDestinations403
  | ListBackupDestinations409
  | ListBackupDestinations500
>;

export function useListBackupDestinationsSuspense<
  TData = Awaited<ReturnType<typeof listBackupDestinations>>,
  TError = ErrorType<
    | ListBackupDestinations401
    | ListBackupDestinations403
    | ListBackupDestinations409
    | ListBackupDestinations500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackupDestinations>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListBackupDestinationsSuspense<
  TData = Awaited<ReturnType<typeof listBackupDestinations>>,
  TError = ErrorType<
    | ListBackupDestinations401
    | ListBackupDestinations403
    | ListBackupDestinations409
    | ListBackupDestinations500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackupDestinations>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListBackupDestinationsSuspense<
  TData = Awaited<ReturnType<typeof listBackupDestinations>>,
  TError = ErrorType<
    | ListBackupDestinations401
    | ListBackupDestinations403
    | ListBackupDestinations409
    | ListBackupDestinations500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackupDestinations>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Backup destinations and their copies
 */

export function useListBackupDestinationsSuspense<
  TData = Awaited<ReturnType<typeof listBackupDestinations>>,
  TError = ErrorType<
    | ListBackupDestinations401
    | ListBackupDestinations403
    | ListBackupDestinations409
    | ListBackupDestinations500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listBackupDestinations>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListBackupDestinationsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getAddBackupDestinationUrl = () => {
  return `/api/v1/backup-destinations`;
};

/**
 * @summary Make a folder on a Node a Backup destination, and copy every Backup there
 */
export const addBackupDestination = async (
  backupsNewDestination: BackupsNewDestination,
  options?: Parameters<typeof apiFetch>[1],
): Promise<BackupsAdded> => {
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
  return apiFetch<BackupsAdded>(getAddBackupDestinationUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(backupsNewDestination),
  });
};

export const getAddBackupDestinationMutationKey = () => ["addBackupDestination"] as const;

export const getAddBackupDestinationMutationOptions = <
  TError = ErrorType<
    | AddBackupDestination401
    | AddBackupDestination403
    | AddBackupDestination404
    | AddBackupDestination409
    | AddBackupDestination422
    | AddBackupDestination500
    | AddBackupDestination503
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof addBackupDestination>>,
    TError,
    AddBackupDestinationMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof addBackupDestination>>,
  TError,
  AddBackupDestinationMutationVariables,
  TContext
> => {
  const mutationKey = getAddBackupDestinationMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof addBackupDestination>>,
    AddBackupDestinationMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return addBackupDestination(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type AddBackupDestinationMutationResult = NonNullable<
  Awaited<ReturnType<typeof addBackupDestination>>
>;
export type AddBackupDestinationMutationBody = BackupsNewDestination;
export type AddBackupDestinationMutationError = ErrorType<
  | AddBackupDestination401
  | AddBackupDestination403
  | AddBackupDestination404
  | AddBackupDestination409
  | AddBackupDestination422
  | AddBackupDestination500
  | AddBackupDestination503
>;
export type AddBackupDestinationMutationVariables = { data: BackupsNewDestination };

/**
 * @summary Make a folder on a Node a Backup destination, and copy every Backup there
 */
export const useAddBackupDestination = <
  TError = ErrorType<
    | AddBackupDestination401
    | AddBackupDestination403
    | AddBackupDestination404
    | AddBackupDestination409
    | AddBackupDestination422
    | AddBackupDestination500
    | AddBackupDestination503
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof addBackupDestination>>,
      TError,
      AddBackupDestinationMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof addBackupDestination>>,
  TError,
  AddBackupDestinationMutationVariables,
  TContext
> => {
  return useMutation(getAddBackupDestinationMutationOptions(options), queryClient);
};
export const getRemoveBackupDestinationUrl = (destinationId: string) => {
  return `/api/v1/backup-destinations/${destinationId}`;
};

/**
 * @summary Stop copying Backups to a destination, leaving its copies in its folder
 */
export const removeBackupDestination = async (
  destinationId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getRemoveBackupDestinationUrl(destinationId), {
    ...options,
    method: "DELETE",
  });
};

export const getRemoveBackupDestinationMutationKey = () => ["removeBackupDestination"] as const;

export const getRemoveBackupDestinationMutationOptions = <
  TError = ErrorType<
    | RemoveBackupDestination401
    | RemoveBackupDestination403
    | RemoveBackupDestination404
    | RemoveBackupDestination409
    | RemoveBackupDestination422
    | RemoveBackupDestination500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof removeBackupDestination>>,
    TError,
    RemoveBackupDestinationMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof removeBackupDestination>>,
  TError,
  RemoveBackupDestinationMutationVariables,
  TContext
> => {
  const mutationKey = getRemoveBackupDestinationMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof removeBackupDestination>>,
    RemoveBackupDestinationMutationVariables
  > = (props) => {
    const { destinationId } = props ?? {};

    return removeBackupDestination(destinationId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RemoveBackupDestinationMutationResult = NonNullable<
  Awaited<ReturnType<typeof removeBackupDestination>>
>;

export type RemoveBackupDestinationMutationError = ErrorType<
  | RemoveBackupDestination401
  | RemoveBackupDestination403
  | RemoveBackupDestination404
  | RemoveBackupDestination409
  | RemoveBackupDestination422
  | RemoveBackupDestination500
>;
export type RemoveBackupDestinationMutationVariables = { destinationId: string };

/**
 * @summary Stop copying Backups to a destination, leaving its copies in its folder
 */
export const useRemoveBackupDestination = <
  TError = ErrorType<
    | RemoveBackupDestination401
    | RemoveBackupDestination403
    | RemoveBackupDestination404
    | RemoveBackupDestination409
    | RemoveBackupDestination422
    | RemoveBackupDestination500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof removeBackupDestination>>,
      TError,
      RemoveBackupDestinationMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof removeBackupDestination>>,
  TError,
  RemoveBackupDestinationMutationVariables,
  TContext
> => {
  return useMutation(getRemoveBackupDestinationMutationOptions(options), queryClient);
};
