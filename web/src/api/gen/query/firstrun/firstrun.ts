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
  CheckSetupCode401,
  CheckSetupCode403,
  CheckSetupCode409,
  CheckSetupCode500,
  FinishFirstRun401,
  FinishFirstRun403,
  FinishFirstRun409,
  FinishFirstRun500,
  FirstRunStepInputBody,
  FirstrunFirstRun,
  GetFirstRun401,
  GetFirstRun403,
  GetFirstRun409,
  GetFirstRun500,
  SetFirstRunStep401,
  SetFirstRunStep403,
  SetFirstRunStep404,
  SetFirstRunStep409,
  SetFirstRunStep422,
  SetFirstRunStep500,
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

export const getGetFirstRunUrl = () => {
  return `/api/v1/first-run`;
};

/**
 * @summary Get first setup's steps
 */
export const getFirstRun = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<FirstrunFirstRun> => {
  return apiFetch<FirstrunFirstRun>(getGetFirstRunUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetFirstRunQueryKey = () => {
  return [`/api/v1/first-run`] as const;
};

export const getGetFirstRunSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getFirstRun>>,
  TError = ErrorType<GetFirstRun401 | GetFirstRun403 | GetFirstRun409 | GetFirstRun500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getFirstRun>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetFirstRunQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getFirstRun>>> = ({ signal }) =>
    getFirstRun({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getFirstRun>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetFirstRunSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getFirstRun>>>;
export type GetFirstRunSuspenseQueryError = ErrorType<
  GetFirstRun401 | GetFirstRun403 | GetFirstRun409 | GetFirstRun500
>;

export function useGetFirstRunSuspense<
  TData = Awaited<ReturnType<typeof getFirstRun>>,
  TError = ErrorType<GetFirstRun401 | GetFirstRun403 | GetFirstRun409 | GetFirstRun500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getFirstRun>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetFirstRunSuspense<
  TData = Awaited<ReturnType<typeof getFirstRun>>,
  TError = ErrorType<GetFirstRun401 | GetFirstRun403 | GetFirstRun409 | GetFirstRun500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getFirstRun>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetFirstRunSuspense<
  TData = Awaited<ReturnType<typeof getFirstRun>>,
  TError = ErrorType<GetFirstRun401 | GetFirstRun403 | GetFirstRun409 | GetFirstRun500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getFirstRun>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get first setup's steps
 */

export function useGetFirstRunSuspense<
  TData = Awaited<ReturnType<typeof getFirstRun>>,
  TError = ErrorType<GetFirstRun401 | GetFirstRun403 | GetFirstRun409 | GetFirstRun500>,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getFirstRun>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetFirstRunSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getFinishFirstRunUrl = () => {
  return `/api/v1/first-run/finish`;
};

/**
 * @summary Finish first setup. It never reopens
 */
export const finishFirstRun = async (options?: Parameters<typeof apiFetch>[1]): Promise<void> => {
  return apiFetch<void>(getFinishFirstRunUrl(), {
    ...options,
    method: "POST",
  });
};

export const getFinishFirstRunMutationKey = () => ["finishFirstRun"] as const;

export const getFinishFirstRunMutationOptions = <
  TError = ErrorType<FinishFirstRun401 | FinishFirstRun403 | FinishFirstRun409 | FinishFirstRun500>,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<Awaited<ReturnType<typeof finishFirstRun>>, TError, void, TContext>;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof finishFirstRun>>, TError, void, TContext> => {
  const mutationKey = getFinishFirstRunMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof finishFirstRun>>, void> = () => {
    return finishFirstRun(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type FinishFirstRunMutationResult = NonNullable<Awaited<ReturnType<typeof finishFirstRun>>>;

export type FinishFirstRunMutationError = ErrorType<
  FinishFirstRun401 | FinishFirstRun403 | FinishFirstRun409 | FinishFirstRun500
>;

/**
 * @summary Finish first setup. It never reopens
 */
export const useFinishFirstRun = <
  TError = ErrorType<FinishFirstRun401 | FinishFirstRun403 | FinishFirstRun409 | FinishFirstRun500>,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof finishFirstRun>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof finishFirstRun>>, TError, void, TContext> => {
  return useMutation(getFinishFirstRunMutationOptions(options), queryClient);
};
export const getSetFirstRunStepUrl = (id: string) => {
  return `/api/v1/first-run/steps/${id}`;
};

/**
 * @summary Mark a first setup step done or skipped
 */
export const setFirstRunStep = async (
  id: string,
  firstRunStepInputBody: FirstRunStepInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<FirstrunFirstRun> => {
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
  return apiFetch<FirstrunFirstRun>(getSetFirstRunStepUrl(id), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(firstRunStepInputBody),
  });
};

export const getSetFirstRunStepMutationKey = () => ["setFirstRunStep"] as const;

export const getSetFirstRunStepMutationOptions = <
  TError = ErrorType<
    | SetFirstRunStep401
    | SetFirstRunStep403
    | SetFirstRunStep404
    | SetFirstRunStep409
    | SetFirstRunStep422
    | SetFirstRunStep500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setFirstRunStep>>,
    TError,
    SetFirstRunStepMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setFirstRunStep>>,
  TError,
  SetFirstRunStepMutationVariables,
  TContext
> => {
  const mutationKey = getSetFirstRunStepMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setFirstRunStep>>,
    SetFirstRunStepMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return setFirstRunStep(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetFirstRunStepMutationResult = NonNullable<
  Awaited<ReturnType<typeof setFirstRunStep>>
>;
export type SetFirstRunStepMutationBody = FirstRunStepInputBody;
export type SetFirstRunStepMutationError = ErrorType<
  | SetFirstRunStep401
  | SetFirstRunStep403
  | SetFirstRunStep404
  | SetFirstRunStep409
  | SetFirstRunStep422
  | SetFirstRunStep500
>;
export type SetFirstRunStepMutationVariables = { id: string; data: FirstRunStepInputBody };

/**
 * @summary Mark a first setup step done or skipped
 */
export const useSetFirstRunStep = <
  TError = ErrorType<
    | SetFirstRunStep401
    | SetFirstRunStep403
    | SetFirstRunStep404
    | SetFirstRunStep409
    | SetFirstRunStep422
    | SetFirstRunStep500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setFirstRunStep>>,
      TError,
      SetFirstRunStepMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setFirstRunStep>>,
  TError,
  SetFirstRunStepMutationVariables,
  TContext
> => {
  return useMutation(getSetFirstRunStepMutationOptions(options), queryClient);
};
export const getCheckSetupCodeUrl = () => {
  return `/api/v1/setup/code`;
};

/**
 * @summary Check the Setup code without changing anything
 */
export const checkSetupCode = async (options?: Parameters<typeof apiFetch>[1]): Promise<void> => {
  return apiFetch<void>(getCheckSetupCodeUrl(), {
    ...options,
    method: "POST",
  });
};

export const getCheckSetupCodeMutationKey = () => ["checkSetupCode"] as const;

export const getCheckSetupCodeMutationOptions = <
  TError = ErrorType<CheckSetupCode401 | CheckSetupCode403 | CheckSetupCode409 | CheckSetupCode500>,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<Awaited<ReturnType<typeof checkSetupCode>>, TError, void, TContext>;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<Awaited<ReturnType<typeof checkSetupCode>>, TError, void, TContext> => {
  const mutationKey = getCheckSetupCodeMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<Awaited<ReturnType<typeof checkSetupCode>>, void> = () => {
    return checkSetupCode(requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CheckSetupCodeMutationResult = NonNullable<Awaited<ReturnType<typeof checkSetupCode>>>;

export type CheckSetupCodeMutationError = ErrorType<
  CheckSetupCode401 | CheckSetupCode403 | CheckSetupCode409 | CheckSetupCode500
>;

/**
 * @summary Check the Setup code without changing anything
 */
export const useCheckSetupCode = <
  TError = ErrorType<CheckSetupCode401 | CheckSetupCode403 | CheckSetupCode409 | CheckSetupCode500>,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof checkSetupCode>>,
      TError,
      void,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<Awaited<ReturnType<typeof checkSetupCode>>, TError, void, TContext> => {
  return useMutation(getCheckSetupCodeMutationOptions(options), queryClient);
};
