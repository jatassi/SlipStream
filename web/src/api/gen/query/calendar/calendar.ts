import { queryOptions as queryOptionsBuilder, useSuspenseQuery } from "@tanstack/react-query";
import type {
  DataTag,
  QueryClient,
  QueryFunction,
  QueryKey,
  UseSuspenseQueryOptions,
  UseSuspenseQueryResult,
} from "@tanstack/react-query";

import type {
  CalendarAnswer,
  GetCalendar401,
  GetCalendar403,
  GetCalendar409,
  GetCalendar422,
  GetCalendar500,
  GetCalendarParams,
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

export const getGetCalendarUrl = (params: GetCalendarParams) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? "null" : String(value));
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0
    ? `/api/v1/calendar?${stringifiedParams}`
    : `/api/v1/calendar`;
};

/**
 * @summary Read release dates in a span of days
 */
export const getCalendar = async (
  params: GetCalendarParams,
  options?: Parameters<typeof apiFetch>[1],
): Promise<CalendarAnswer> => {
  return apiFetch<CalendarAnswer>(getGetCalendarUrl(params), {
    ...options,
    method: "GET",
  });
};

export const getGetCalendarQueryKey = (params?: GetCalendarParams) => {
  return [`/api/v1/calendar`, ...(params ? [params] : [])] as const;
};

export const getGetCalendarSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getCalendar>>,
  TError = ErrorType<
    GetCalendar401 | GetCalendar403 | GetCalendar409 | GetCalendar422 | GetCalendar500
  >,
>(
  params: GetCalendarParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCalendar>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetCalendarQueryKey(params);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getCalendar>>> = ({ signal }) =>
    getCalendar(params, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCalendar>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetCalendarSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getCalendar>>>;
export type GetCalendarSuspenseQueryError = ErrorType<
  GetCalendar401 | GetCalendar403 | GetCalendar409 | GetCalendar422 | GetCalendar500
>;

export function useGetCalendarSuspense<
  TData = Awaited<ReturnType<typeof getCalendar>>,
  TError = ErrorType<
    GetCalendar401 | GetCalendar403 | GetCalendar409 | GetCalendar422 | GetCalendar500
  >,
>(
  params: GetCalendarParams,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCalendar>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetCalendarSuspense<
  TData = Awaited<ReturnType<typeof getCalendar>>,
  TError = ErrorType<
    GetCalendar401 | GetCalendar403 | GetCalendar409 | GetCalendar422 | GetCalendar500
  >,
>(
  params: GetCalendarParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCalendar>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetCalendarSuspense<
  TData = Awaited<ReturnType<typeof getCalendar>>,
  TError = ErrorType<
    GetCalendar401 | GetCalendar403 | GetCalendar409 | GetCalendar422 | GetCalendar500
  >,
>(
  params: GetCalendarParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCalendar>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Read release dates in a span of days
 */

export function useGetCalendarSuspense<
  TData = Awaited<ReturnType<typeof getCalendar>>,
  TError = ErrorType<
    GetCalendar401 | GetCalendar403 | GetCalendar409 | GetCalendar422 | GetCalendar500
  >,
>(
  params: GetCalendarParams,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getCalendar>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetCalendarSuspenseQueryOptions(params, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
