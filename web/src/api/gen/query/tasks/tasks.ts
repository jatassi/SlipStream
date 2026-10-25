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
  CancelTaskRun401,
  CancelTaskRun403,
  CancelTaskRun404,
  CancelTaskRun409,
  CancelTaskRun422,
  CancelTaskRun500,
  GetTask401,
  GetTask403,
  GetTask404,
  GetTask409,
  GetTask422,
  GetTask500,
  GetTaskSettings401,
  GetTaskSettings403,
  GetTaskSettings409,
  GetTaskSettings500,
  ListOpenTaskRuns401,
  ListOpenTaskRuns403,
  ListOpenTaskRuns409,
  ListOpenTaskRuns500,
  ListTasks401,
  ListTasks403,
  ListTasks409,
  ListTasks500,
  RunTask401,
  RunTask403,
  RunTask404,
  RunTask409,
  RunTask422,
  RunTask500,
  TasksDetail,
  TasksOpenRuns,
  TasksOverview,
  TasksRunStarted,
  TasksSchedule,
  TasksSchedulePatch,
  TasksTask,
  TasksTaskPatch,
  UpdateTask401,
  UpdateTask403,
  UpdateTask404,
  UpdateTask409,
  UpdateTask422,
  UpdateTask500,
  UpdateTaskSettings401,
  UpdateTaskSettings403,
  UpdateTaskSettings409,
  UpdateTaskSettings422,
  UpdateTaskSettings500,
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

export const getListTasksUrl = () => {
  return `/api/v1/tasks`;
};

/**
 * @summary List every shown task by group, with the status sentence and the schedule settings
 */
export const listTasks = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TasksOverview> => {
  return apiFetch<TasksOverview>(getListTasksUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListTasksQueryKey = () => {
  return [`/api/v1/tasks`] as const;
};

export const getListTasksSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listTasks>>,
  TError = ErrorType<ListTasks401 | ListTasks403 | ListTasks409 | ListTasks500>,
>(options?: {
  query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTasks>>, TError, TData>>;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListTasksQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listTasks>>> = ({ signal }) =>
    listTasks({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTasks>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListTasksSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof listTasks>>>;
export type ListTasksSuspenseQueryError = ErrorType<
  ListTasks401 | ListTasks403 | ListTasks409 | ListTasks500
>;

export function useListTasksSuspense<
  TData = Awaited<ReturnType<typeof listTasks>>,
  TError = ErrorType<ListTasks401 | ListTasks403 | ListTasks409 | ListTasks500>,
>(
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTasks>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTasksSuspense<
  TData = Awaited<ReturnType<typeof listTasks>>,
  TError = ErrorType<ListTasks401 | ListTasks403 | ListTasks409 | ListTasks500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTasks>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListTasksSuspense<
  TData = Awaited<ReturnType<typeof listTasks>>,
  TError = ErrorType<ListTasks401 | ListTasks403 | ListTasks409 | ListTasks500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTasks>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List every shown task by group, with the status sentence and the schedule settings
 */

export function useListTasksSuspense<
  TData = Awaited<ReturnType<typeof listTasks>>,
  TError = ErrorType<ListTasks401 | ListTasks403 | ListTasks409 | ListTasks500>,
>(
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof listTasks>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListTasksSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getListOpenTaskRunsUrl = () => {
  return `/api/v1/tasks/runs/open`;
};

/**
 * @summary List every open run, with how long a scheduled one waits before the header shows it
 */
export const listOpenTaskRuns = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TasksOpenRuns> => {
  return apiFetch<TasksOpenRuns>(getListOpenTaskRunsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListOpenTaskRunsQueryKey = () => {
  return [`/api/v1/tasks/runs/open`] as const;
};

export const getListOpenTaskRunsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listOpenTaskRuns>>,
  TError = ErrorType<
    ListOpenTaskRuns401 | ListOpenTaskRuns403 | ListOpenTaskRuns409 | ListOpenTaskRuns500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listOpenTaskRuns>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListOpenTaskRunsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listOpenTaskRuns>>> = ({ signal }) =>
    listOpenTaskRuns({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof listOpenTaskRuns>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListOpenTaskRunsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listOpenTaskRuns>>
>;
export type ListOpenTaskRunsSuspenseQueryError = ErrorType<
  ListOpenTaskRuns401 | ListOpenTaskRuns403 | ListOpenTaskRuns409 | ListOpenTaskRuns500
>;

export function useListOpenTaskRunsSuspense<
  TData = Awaited<ReturnType<typeof listOpenTaskRuns>>,
  TError = ErrorType<
    ListOpenTaskRuns401 | ListOpenTaskRuns403 | ListOpenTaskRuns409 | ListOpenTaskRuns500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listOpenTaskRuns>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListOpenTaskRunsSuspense<
  TData = Awaited<ReturnType<typeof listOpenTaskRuns>>,
  TError = ErrorType<
    ListOpenTaskRuns401 | ListOpenTaskRuns403 | ListOpenTaskRuns409 | ListOpenTaskRuns500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listOpenTaskRuns>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListOpenTaskRunsSuspense<
  TData = Awaited<ReturnType<typeof listOpenTaskRuns>>,
  TError = ErrorType<
    ListOpenTaskRuns401 | ListOpenTaskRuns403 | ListOpenTaskRuns409 | ListOpenTaskRuns500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listOpenTaskRuns>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List every open run, with how long a scheduled one waits before the header shows it
 */

export function useListOpenTaskRunsSuspense<
  TData = Awaited<ReturnType<typeof listOpenTaskRuns>>,
  TError = ErrorType<
    ListOpenTaskRuns401 | ListOpenTaskRuns403 | ListOpenTaskRuns409 | ListOpenTaskRuns500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listOpenTaskRuns>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListOpenTaskRunsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCancelTaskRunUrl = (runId: number) => {
  return `/api/v1/tasks/runs/${runId}`;
};

/**
 * @summary Cancel an open run
 */
export const cancelTaskRun = async (
  runId: number,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getCancelTaskRunUrl(runId), {
    ...options,
    method: "DELETE",
  });
};

export const getCancelTaskRunMutationKey = () => ["cancelTaskRun"] as const;

export const getCancelTaskRunMutationOptions = <
  TError = ErrorType<
    | CancelTaskRun401
    | CancelTaskRun403
    | CancelTaskRun404
    | CancelTaskRun409
    | CancelTaskRun422
    | CancelTaskRun500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof cancelTaskRun>>,
    TError,
    CancelTaskRunMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof cancelTaskRun>>,
  TError,
  CancelTaskRunMutationVariables,
  TContext
> => {
  const mutationKey = getCancelTaskRunMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof cancelTaskRun>>,
    CancelTaskRunMutationVariables
  > = (props) => {
    const { runId } = props ?? {};

    return cancelTaskRun(runId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CancelTaskRunMutationResult = NonNullable<Awaited<ReturnType<typeof cancelTaskRun>>>;

export type CancelTaskRunMutationError = ErrorType<
  | CancelTaskRun401
  | CancelTaskRun403
  | CancelTaskRun404
  | CancelTaskRun409
  | CancelTaskRun422
  | CancelTaskRun500
>;
export type CancelTaskRunMutationVariables = { runId: number };

/**
 * @summary Cancel an open run
 */
export const useCancelTaskRun = <
  TError = ErrorType<
    | CancelTaskRun401
    | CancelTaskRun403
    | CancelTaskRun404
    | CancelTaskRun409
    | CancelTaskRun422
    | CancelTaskRun500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof cancelTaskRun>>,
      TError,
      CancelTaskRunMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof cancelTaskRun>>,
  TError,
  CancelTaskRunMutationVariables,
  TContext
> => {
  return useMutation(getCancelTaskRunMutationOptions(options), queryClient);
};
export const getGetTaskSettingsUrl = () => {
  return `/api/v1/tasks/settings`;
};

/**
 * @summary Get the schedule settings
 */
export const getTaskSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<TasksSchedule> => {
  return apiFetch<TasksSchedule>(getGetTaskSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetTaskSettingsQueryKey = () => {
  return [`/api/v1/tasks/settings`] as const;
};

export const getGetTaskSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getTaskSettings>>,
  TError = ErrorType<
    GetTaskSettings401 | GetTaskSettings403 | GetTaskSettings409 | GetTaskSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTaskSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetTaskSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getTaskSettings>>> = ({ signal }) =>
    getTaskSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTaskSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetTaskSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getTaskSettings>>
>;
export type GetTaskSettingsSuspenseQueryError = ErrorType<
  GetTaskSettings401 | GetTaskSettings403 | GetTaskSettings409 | GetTaskSettings500
>;

export function useGetTaskSettingsSuspense<
  TData = Awaited<ReturnType<typeof getTaskSettings>>,
  TError = ErrorType<
    GetTaskSettings401 | GetTaskSettings403 | GetTaskSettings409 | GetTaskSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTaskSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTaskSettingsSuspense<
  TData = Awaited<ReturnType<typeof getTaskSettings>>,
  TError = ErrorType<
    GetTaskSettings401 | GetTaskSettings403 | GetTaskSettings409 | GetTaskSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTaskSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTaskSettingsSuspense<
  TData = Awaited<ReturnType<typeof getTaskSettings>>,
  TError = ErrorType<
    GetTaskSettings401 | GetTaskSettings403 | GetTaskSettings409 | GetTaskSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTaskSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the schedule settings
 */

export function useGetTaskSettingsSuspense<
  TData = Awaited<ReturnType<typeof getTaskSettings>>,
  TError = ErrorType<
    GetTaskSettings401 | GetTaskSettings403 | GetTaskSettings409 | GetTaskSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTaskSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetTaskSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateTaskSettingsUrl = () => {
  return `/api/v1/tasks/settings`;
};

/**
 * @summary Change the schedule settings
 */
export const updateTaskSettings = async (
  tasksSchedulePatch: TasksSchedulePatch,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TasksSchedule> => {
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
  return apiFetch<TasksSchedule>(getUpdateTaskSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(tasksSchedulePatch),
  });
};

export const getUpdateTaskSettingsMutationKey = () => ["updateTaskSettings"] as const;

export const getUpdateTaskSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateTaskSettings401
    | UpdateTaskSettings403
    | UpdateTaskSettings409
    | UpdateTaskSettings422
    | UpdateTaskSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateTaskSettings>>,
    TError,
    UpdateTaskSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateTaskSettings>>,
  TError,
  UpdateTaskSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateTaskSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateTaskSettings>>,
    UpdateTaskSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateTaskSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateTaskSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateTaskSettings>>
>;
export type UpdateTaskSettingsMutationBody = TasksSchedulePatch;
export type UpdateTaskSettingsMutationError = ErrorType<
  | UpdateTaskSettings401
  | UpdateTaskSettings403
  | UpdateTaskSettings409
  | UpdateTaskSettings422
  | UpdateTaskSettings500
>;
export type UpdateTaskSettingsMutationVariables = { data: TasksSchedulePatch };

/**
 * @summary Change the schedule settings
 */
export const useUpdateTaskSettings = <
  TError = ErrorType<
    | UpdateTaskSettings401
    | UpdateTaskSettings403
    | UpdateTaskSettings409
    | UpdateTaskSettings422
    | UpdateTaskSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateTaskSettings>>,
      TError,
      UpdateTaskSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateTaskSettings>>,
  TError,
  UpdateTaskSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateTaskSettingsMutationOptions(options), queryClient);
};
export const getGetTaskUrl = (taskId: string) => {
  return `/api/v1/tasks/${taskId}`;
};

/**
 * @summary Get one shown task with its units and its runs
 */
export const getTask = async (
  taskId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TasksDetail> => {
  return apiFetch<TasksDetail>(getGetTaskUrl(taskId), {
    ...options,
    method: "GET",
  });
};

export const getGetTaskQueryKey = (taskId: string) => {
  return [`/api/v1/tasks/${taskId}`] as const;
};

export const getGetTaskSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getTask>>,
  TError = ErrorType<GetTask401 | GetTask403 | GetTask404 | GetTask409 | GetTask422 | GetTask500>,
>(
  taskId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTask>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetTaskQueryKey(taskId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getTask>>> = ({ signal }) =>
    getTask(taskId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTask>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetTaskSuspenseQueryResult = NonNullable<Awaited<ReturnType<typeof getTask>>>;
export type GetTaskSuspenseQueryError = ErrorType<
  GetTask401 | GetTask403 | GetTask404 | GetTask409 | GetTask422 | GetTask500
>;

export function useGetTaskSuspense<
  TData = Awaited<ReturnType<typeof getTask>>,
  TError = ErrorType<GetTask401 | GetTask403 | GetTask404 | GetTask409 | GetTask422 | GetTask500>,
>(
  taskId: string,
  options: {
    query: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTask>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTaskSuspense<
  TData = Awaited<ReturnType<typeof getTask>>,
  TError = ErrorType<GetTask401 | GetTask403 | GetTask404 | GetTask409 | GetTask422 | GetTask500>,
>(
  taskId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTask>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetTaskSuspense<
  TData = Awaited<ReturnType<typeof getTask>>,
  TError = ErrorType<GetTask401 | GetTask403 | GetTask404 | GetTask409 | GetTask422 | GetTask500>,
>(
  taskId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTask>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get one shown task with its units and its runs
 */

export function useGetTaskSuspense<
  TData = Awaited<ReturnType<typeof getTask>>,
  TError = ErrorType<GetTask401 | GetTask403 | GetTask404 | GetTask409 | GetTask422 | GetTask500>,
>(
  taskId: string,
  options?: {
    query?: Partial<UseSuspenseQueryOptions<Awaited<ReturnType<typeof getTask>>, TError, TData>>;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetTaskSuspenseQueryOptions(taskId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateTaskUrl = (taskId: string) => {
  return `/api/v1/tasks/${taskId}`;
};

/**
 * @summary Change a task's cadence, overdue rule or tuning
 */
export const updateTask = async (
  taskId: string,
  tasksTaskPatch: TasksTaskPatch,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TasksTask> => {
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
  return apiFetch<TasksTask>(getUpdateTaskUrl(taskId), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(tasksTaskPatch),
  });
};

export const getUpdateTaskMutationKey = () => ["updateTask"] as const;

export const getUpdateTaskMutationOptions = <
  TError = ErrorType<
    UpdateTask401 | UpdateTask403 | UpdateTask404 | UpdateTask409 | UpdateTask422 | UpdateTask500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateTask>>,
    TError,
    UpdateTaskMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateTask>>,
  TError,
  UpdateTaskMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateTaskMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateTask>>,
    UpdateTaskMutationVariables
  > = (props) => {
    const { taskId, data } = props ?? {};

    return updateTask(taskId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateTaskMutationResult = NonNullable<Awaited<ReturnType<typeof updateTask>>>;
export type UpdateTaskMutationBody = TasksTaskPatch;
export type UpdateTaskMutationError = ErrorType<
  UpdateTask401 | UpdateTask403 | UpdateTask404 | UpdateTask409 | UpdateTask422 | UpdateTask500
>;
export type UpdateTaskMutationVariables = { taskId: string; data: TasksTaskPatch };

/**
 * @summary Change a task's cadence, overdue rule or tuning
 */
export const useUpdateTask = <
  TError = ErrorType<
    UpdateTask401 | UpdateTask403 | UpdateTask404 | UpdateTask409 | UpdateTask422 | UpdateTask500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateTask>>,
      TError,
      UpdateTaskMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateTask>>,
  TError,
  UpdateTaskMutationVariables,
  TContext
> => {
  return useMutation(getUpdateTaskMutationOptions(options), queryClient);
};
export const getRunTaskUrl = (taskId: string) => {
  return `/api/v1/tasks/${taskId}/runs`;
};

/**
 * @summary Run a task now over every unit. Its schedule stays as it was
 */
export const runTask = async (
  taskId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<TasksRunStarted> => {
  return apiFetch<TasksRunStarted>(getRunTaskUrl(taskId), {
    ...options,
    method: "POST",
  });
};

export const getRunTaskMutationKey = () => ["runTask"] as const;

export const getRunTaskMutationOptions = <
  TError = ErrorType<RunTask401 | RunTask403 | RunTask404 | RunTask409 | RunTask422 | RunTask500>,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof runTask>>,
    TError,
    RunTaskMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof runTask>>,
  TError,
  RunTaskMutationVariables,
  TContext
> => {
  const mutationKey = getRunTaskMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof runTask>>,
    RunTaskMutationVariables
  > = (props) => {
    const { taskId } = props ?? {};

    return runTask(taskId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type RunTaskMutationResult = NonNullable<Awaited<ReturnType<typeof runTask>>>;

export type RunTaskMutationError = ErrorType<
  RunTask401 | RunTask403 | RunTask404 | RunTask409 | RunTask422 | RunTask500
>;
export type RunTaskMutationVariables = { taskId: string };

/**
 * @summary Run a task now over every unit. Its schedule stays as it was
 */
export const useRunTask = <
  TError = ErrorType<RunTask401 | RunTask403 | RunTask404 | RunTask409 | RunTask422 | RunTask500>,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof runTask>>,
      TError,
      RunTaskMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof runTask>>,
  TError,
  RunTaskMutationVariables,
  TContext
> => {
  return useMutation(getRunTaskMutationOptions(options), queryClient);
};
