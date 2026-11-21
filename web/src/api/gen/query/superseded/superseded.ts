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
  GetSuperseded401,
  GetSuperseded403,
  GetSuperseded409,
  GetSuperseded500,
  ResumeSupersededController401,
  ResumeSupersededController403,
  ResumeSupersededController409,
  ResumeSupersededController500,
  RetireController401,
  RetireController403,
  RetireController409,
  RetireController500,
  StopSupersededController401,
  StopSupersededController403,
  StopSupersededController409,
  StopSupersededController500,
  SupersededView,
  WipeInputBody,
  WipeSupersededController401,
  WipeSupersededController403,
  WipeSupersededController409,
  WipeSupersededController422,
  WipeSupersededController500,
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

export const getRetireControllerUrl = () => {
  return `/api/v1/backups/retire`;
};

/**
 * @summary Retire this Controller after its library was restored on another machine
 */
export const retireController = async (options?: Parameters<typeof apiFetch>[1]): Promise<void> => {
  return apiFetch<void>(getRetireControllerUrl(), {
    ...options,
    method: "POST",
  });
};

export const getRetireControllerMutationKey = () => ["retireController"] as const;

export const getRetireControllerMutationOptions = <
  TError = ErrorType<
    RetireController401 | RetireController403 | RetireController409 | RetireController500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof retireController>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof retireController>>, TError, void, TContext> => {
  const mutationKey = getRetireControllerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof retireController>>, void> = () => {
    return retireController(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RetireControllerMutationResult = NonNullable<
  Awaited<ReturnType<typeof retireController>>
>;

export type RetireControllerMutationError = ErrorType<
  RetireController401 | RetireController403 | RetireController409 | RetireController500
>;

/**
 * @summary Retire this Controller after its library was restored on another machine
 */
export const useRetireController = <
  TError = ErrorType<
    RetireController401 | RetireController403 | RetireController409 | RetireController500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof retireController>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof retireController>>, TError, void, TContext> => {
  return useMutation(getRetireControllerMutationOptions(options), queryClient);
};
export const getGetSupersededUrl = () => {
  return `/api/v1/superseded`;
};

/**
 * @summary Read whether a restore on another machine replaced this Controller
 */
export const getSuperseded = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<SupersededView> => {
  return apiFetch<SupersededView>(getGetSupersededUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetSupersededQueryKey = () => {
  return [`/api/v1/superseded`] as const;
};

export const getGetSupersededSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSuperseded>>,
  TError = ErrorType<GetSuperseded401 | GetSuperseded403 | GetSuperseded409 | GetSuperseded500>,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSuperseded>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSupersededQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSuperseded>>> = ({ signal }) =>
    getSuperseded({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSuperseded>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSupersededSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getSuperseded>>
>;
export type GetSupersededSuspenseQueryError = ErrorType<
  GetSuperseded401 | GetSuperseded403 | GetSuperseded409 | GetSuperseded500
>;

export function useGetSupersededSuspense<
  TData = Awaited<ReturnType<typeof getSuperseded>>,
  TError = ErrorType<GetSuperseded401 | GetSuperseded403 | GetSuperseded409 | GetSuperseded500>,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSuperseded>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSupersededSuspense<
  TData = Awaited<ReturnType<typeof getSuperseded>>,
  TError = ErrorType<GetSuperseded401 | GetSuperseded403 | GetSuperseded409 | GetSuperseded500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSuperseded>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSupersededSuspense<
  TData = Awaited<ReturnType<typeof getSuperseded>>,
  TError = ErrorType<GetSuperseded401 | GetSuperseded403 | GetSuperseded409 | GetSuperseded500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSuperseded>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Read whether a restore on another machine replaced this Controller
 */

export function useGetSupersededSuspense<
  TData = Awaited<ReturnType<typeof getSuperseded>>,
  TError = ErrorType<GetSuperseded401 | GetSuperseded403 | GetSuperseded409 | GetSuperseded500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSuperseded>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSupersededSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getResumeSupersededControllerUrl = () => {
  return `/api/v1/superseded/resume`;
};

/**
 * @summary Resume this Controller, so an earlier proof of a restore can't stop it again
 */
export const resumeSupersededController = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getResumeSupersededControllerUrl(), {
    ...options,
    method: "POST",
  });
};

export const getResumeSupersededControllerMutationKey = () =>
  ["resumeSupersededController"] as const;

export const getResumeSupersededControllerMutationOptions = <
  TError = ErrorType<
    | ResumeSupersededController401
    | ResumeSupersededController403
    | ResumeSupersededController409
    | ResumeSupersededController500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof resumeSupersededController>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof resumeSupersededController>>,
  TError,
  void,
  TContext
> => {
  const mutationKey = getResumeSupersededControllerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof resumeSupersededController>>,
    void
  > = () => {
    return resumeSupersededController(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ResumeSupersededControllerMutationResult = NonNullable<
  Awaited<ReturnType<typeof resumeSupersededController>>
>;

export type ResumeSupersededControllerMutationError = ErrorType<
  | ResumeSupersededController401
  | ResumeSupersededController403
  | ResumeSupersededController409
  | ResumeSupersededController500
>;

/**
 * @summary Resume this Controller, so an earlier proof of a restore can't stop it again
 */
export const useResumeSupersededController = <
  TError = ErrorType<
    | ResumeSupersededController401
    | ResumeSupersededController403
    | ResumeSupersededController409
    | ResumeSupersededController500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof resumeSupersededController>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof resumeSupersededController>>,
  TError,
  void,
  TContext
> => {
  return useMutation(getResumeSupersededControllerMutationOptions(options), queryClient);
};
export const getStopSupersededControllerUrl = () => {
  return `/api/v1/superseded/stop`;
};

/**
 * @summary Stop this Controller after a Worker said a restore replaced it
 */
export const stopSupersededController = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getStopSupersededControllerUrl(), {
    ...options,
    method: "POST",
  });
};

export const getStopSupersededControllerMutationKey = () => ["stopSupersededController"] as const;

export const getStopSupersededControllerMutationOptions = <
  TError = ErrorType<
    | StopSupersededController401
    | StopSupersededController403
    | StopSupersededController409
    | StopSupersededController500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof stopSupersededController>>,
    TError,
    void,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof stopSupersededController>>,
  TError,
  void,
  TContext
> => {
  const mutationKey = getStopSupersededControllerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof stopSupersededController>>,
    void
  > = () => {
    return stopSupersededController(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type StopSupersededControllerMutationResult = NonNullable<
  Awaited<ReturnType<typeof stopSupersededController>>
>;

export type StopSupersededControllerMutationError = ErrorType<
  | StopSupersededController401
  | StopSupersededController403
  | StopSupersededController409
  | StopSupersededController500
>;

/**
 * @summary Stop this Controller after a Worker said a restore replaced it
 */
export const useStopSupersededController = <
  TError = ErrorType<
    | StopSupersededController401
    | StopSupersededController403
    | StopSupersededController409
    | StopSupersededController500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof stopSupersededController>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof stopSupersededController>>,
  TError,
  void,
  TContext
> => {
  return useMutation(getStopSupersededControllerMutationOptions(options), queryClient);
};
export const getWipeSupersededControllerUrl = () => {
  return `/api/v1/superseded/wipe`;
};

/**
 * @summary Back up and wipe this Controller, then join it to the new Controller as a Worker
 */
export const wipeSupersededController = async (
  wipeInputBody: WipeInputBody,
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
  return apiFetch<void>(getWipeSupersededControllerUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(wipeInputBody),
  });
};

export const getWipeSupersededControllerMutationKey = () => ["wipeSupersededController"] as const;

export const getWipeSupersededControllerMutationOptions = <
  TError = ErrorType<
    | WipeSupersededController401
    | WipeSupersededController403
    | WipeSupersededController409
    | WipeSupersededController422
    | WipeSupersededController500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof wipeSupersededController>>,
    TError,
    WipeSupersededControllerMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof wipeSupersededController>>,
  TError,
  WipeSupersededControllerMutationVariables,
  TContext
> => {
  const mutationKey = getWipeSupersededControllerMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof wipeSupersededController>>,
    WipeSupersededControllerMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return wipeSupersededController(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type WipeSupersededControllerMutationResult = NonNullable<
  Awaited<ReturnType<typeof wipeSupersededController>>
>;
export type WipeSupersededControllerMutationBody = WipeInputBody;
export type WipeSupersededControllerMutationError = ErrorType<
  | WipeSupersededController401
  | WipeSupersededController403
  | WipeSupersededController409
  | WipeSupersededController422
  | WipeSupersededController500
>;
export type WipeSupersededControllerMutationVariables = { data: WipeInputBody };

/**
 * @summary Back up and wipe this Controller, then join it to the new Controller as a Worker
 */
export const useWipeSupersededController = <
  TError = ErrorType<
    | WipeSupersededController401
    | WipeSupersededController403
    | WipeSupersededController409
    | WipeSupersededController422
    | WipeSupersededController500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof wipeSupersededController>>,
      TError,
      WipeSupersededControllerMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof wipeSupersededController>>,
  TError,
  WipeSupersededControllerMutationVariables,
  TContext
> => {
  return useMutation(getWipeSupersededControllerMutationOptions(options), queryClient);
};
