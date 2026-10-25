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
  ClaimWorker401,
  ClaimWorker403,
  ClaimWorker409,
  ClaimWorker422,
  ClaimWorker500,
  ClaimWorker502,
  GetWorkerClaim401,
  GetWorkerClaim403,
  GetWorkerClaim404,
  GetWorkerClaim409,
  GetWorkerClaim500,
  PairingJoinBody,
  PairingWorkerClaim,
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

export const getGetWorkerClaimUrl = () => {
  return `/api/v1/setup/worker`;
};

/**
 * @summary Get the Join that awaited its Controller's dial: still awaiting, joined or failed
 */
export const getWorkerClaim = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<PairingWorkerClaim> => {
  return apiFetch<PairingWorkerClaim>(getGetWorkerClaimUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetWorkerClaimQueryKey = () => {
  return [`/api/v1/setup/worker`] as const;
};

export const getGetWorkerClaimSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getWorkerClaim>>,
  TError = ErrorType<
    | GetWorkerClaim401
    | GetWorkerClaim403
    | GetWorkerClaim404
    | GetWorkerClaim409
    | GetWorkerClaim500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkerClaim>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetWorkerClaimQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getWorkerClaim>>> = ({ signal }) =>
    getWorkerClaim({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkerClaim>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetWorkerClaimSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getWorkerClaim>>
>;
export type GetWorkerClaimSuspenseQueryError = ErrorType<
  GetWorkerClaim401 | GetWorkerClaim403 | GetWorkerClaim404 | GetWorkerClaim409 | GetWorkerClaim500
>;

export function useGetWorkerClaimSuspense<
  TData = Awaited<ReturnType<typeof getWorkerClaim>>,
  TError = ErrorType<
    | GetWorkerClaim401
    | GetWorkerClaim403
    | GetWorkerClaim404
    | GetWorkerClaim409
    | GetWorkerClaim500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkerClaim>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkerClaimSuspense<
  TData = Awaited<ReturnType<typeof getWorkerClaim>>,
  TError = ErrorType<
    | GetWorkerClaim401
    | GetWorkerClaim403
    | GetWorkerClaim404
    | GetWorkerClaim409
    | GetWorkerClaim500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkerClaim>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkerClaimSuspense<
  TData = Awaited<ReturnType<typeof getWorkerClaim>>,
  TError = ErrorType<
    | GetWorkerClaim401
    | GetWorkerClaim403
    | GetWorkerClaim404
    | GetWorkerClaim409
    | GetWorkerClaim500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkerClaim>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Join that awaited its Controller's dial: still awaiting, joined or failed
 */

export function useGetWorkerClaimSuspense<
  TData = Awaited<ReturnType<typeof getWorkerClaim>>,
  TError = ErrorType<
    | GetWorkerClaim401
    | GetWorkerClaim403
    | GetWorkerClaim404
    | GetWorkerClaim409
    | GetWorkerClaim500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkerClaim>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetWorkerClaimSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getClaimWorkerUrl = () => {
  return `/api/v1/setup/worker`;
};

/**
 * @summary Join this Node to a Controller as a Worker with a Join Code. Returns 202 while it awaits the Controller's dial
 */
export const claimWorker = async (
  pairingJoinBody: PairingJoinBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<PairingWorkerClaim> => {
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
  return apiFetch<PairingWorkerClaim>(getClaimWorkerUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(pairingJoinBody),
  });
};

export const getClaimWorkerMutationKey = () => ["claimWorker"] as const;

export const getClaimWorkerMutationOptions = <
  TError = ErrorType<
    | ClaimWorker401
    | ClaimWorker403
    | ClaimWorker409
    | ClaimWorker422
    | ClaimWorker500
    | ClaimWorker502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof claimWorker>>,
    TError,
    ClaimWorkerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof claimWorker>>,
  TError,
  ClaimWorkerMutationVariables,
  TContext
> => {
  const mutationKey = getClaimWorkerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof claimWorker>>,
    ClaimWorkerMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return claimWorker(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ClaimWorkerMutationResult = NonNullable<Awaited<ReturnType<typeof claimWorker>>>;
export type ClaimWorkerMutationBody = PairingJoinBody;
export type ClaimWorkerMutationError = ErrorType<
  | ClaimWorker401
  | ClaimWorker403
  | ClaimWorker409
  | ClaimWorker422
  | ClaimWorker500
  | ClaimWorker502
>;
export type ClaimWorkerMutationVariables = { data: PairingJoinBody };

/**
 * @summary Join this Node to a Controller as a Worker with a Join Code. Returns 202 while it awaits the Controller's dial
 */
export const useClaimWorker = <
  TError = ErrorType<
    | ClaimWorker401
    | ClaimWorker403
    | ClaimWorker409
    | ClaimWorker422
    | ClaimWorker500
    | ClaimWorker502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof claimWorker>>,
      TError,
      ClaimWorkerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof claimWorker>>,
  TError,
  ClaimWorkerMutationVariables,
  TContext
> => {
  return useMutation(getClaimWorkerMutationOptions(options), queryClient);
};
