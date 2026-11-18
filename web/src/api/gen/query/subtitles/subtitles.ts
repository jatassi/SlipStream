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
  CountSubtitleTrack401,
  CountSubtitleTrack403,
  CountSubtitleTrack404,
  CountSubtitleTrack409,
  CountSubtitleTrack422,
  CountSubtitleTrack500,
  DeleteSubtitleProviderKey401,
  DeleteSubtitleProviderKey403,
  DeleteSubtitleProviderKey404,
  DeleteSubtitleProviderKey409,
  DeleteSubtitleProviderKey422,
  DeleteSubtitleProviderKey500,
  GetSubtitleAttempt401,
  GetSubtitleAttempt403,
  GetSubtitleAttempt404,
  GetSubtitleAttempt409,
  GetSubtitleAttempt410,
  GetSubtitleAttempt422,
  GetSubtitleAttempt500,
  GetSubtitleCandidates400,
  GetSubtitleCandidates401,
  GetSubtitleCandidates403,
  GetSubtitleCandidates404,
  GetSubtitleCandidates409,
  GetSubtitleCandidates410,
  GetSubtitleCandidates422,
  GetSubtitleCandidates500,
  GetSubtitleNodeSettings401,
  GetSubtitleNodeSettings403,
  GetSubtitleNodeSettings404,
  GetSubtitleNodeSettings409,
  GetSubtitleNodeSettings422,
  GetSubtitleNodeSettings500,
  GetSubtitleSettings401,
  GetSubtitleSettings403,
  GetSubtitleSettings409,
  GetSubtitleSettings500,
  GetWorkSubtitles401,
  GetWorkSubtitles403,
  GetWorkSubtitles404,
  GetWorkSubtitles409,
  GetWorkSubtitles422,
  GetWorkSubtitles500,
  IgnoreSubtitleTrack401,
  IgnoreSubtitleTrack403,
  IgnoreSubtitleTrack404,
  IgnoreSubtitleTrack409,
  IgnoreSubtitleTrack422,
  IgnoreSubtitleTrack500,
  KeepSubtitleUpload401,
  KeepSubtitleUpload403,
  KeepSubtitleUpload404,
  KeepSubtitleUpload409,
  KeepSubtitleUpload410,
  KeepSubtitleUpload422,
  KeepSubtitleUpload500,
  ListSubtitleProviders401,
  ListSubtitleProviders403,
  ListSubtitleProviders409,
  ListSubtitleProviders500,
  MarkSubtitleBad400,
  MarkSubtitleBad401,
  MarkSubtitleBad403,
  MarkSubtitleBad404,
  MarkSubtitleBad409,
  MarkSubtitleBad422,
  MarkSubtitleBad500,
  OrderSubtitleProviders401,
  OrderSubtitleProviders403,
  OrderSubtitleProviders409,
  OrderSubtitleProviders422,
  OrderSubtitleProviders500,
  PickSubtitleCandidate400,
  PickSubtitleCandidate401,
  PickSubtitleCandidate403,
  PickSubtitleCandidate404,
  PickSubtitleCandidate409,
  PickSubtitleCandidate410,
  PickSubtitleCandidate422,
  PickSubtitleCandidate500,
  SearchSubtitle400,
  SearchSubtitle401,
  SearchSubtitle403,
  SearchSubtitle404,
  SearchSubtitle409,
  SearchSubtitle422,
  SearchSubtitle500,
  SetSubtitleProviderKey401,
  SetSubtitleProviderKey403,
  SetSubtitleProviderKey404,
  SetSubtitleProviderKey409,
  SetSubtitleProviderKey422,
  SetSubtitleProviderKey500,
  SetSubtitleProviderKey502,
  StartSubtitleCandidates400,
  StartSubtitleCandidates401,
  StartSubtitleCandidates403,
  StartSubtitleCandidates404,
  StartSubtitleCandidates409,
  StartSubtitleCandidates422,
  StartSubtitleCandidates500,
  SubtitlesAttempt,
  SubtitlesAttempting,
  SubtitlesCandidates,
  SubtitlesNodeSettings,
  SubtitlesProvider,
  SubtitlesProviderChange,
  SubtitlesProviderKey,
  SubtitlesProviderList,
  SubtitlesProviderOrder,
  SubtitlesStarted,
  SubtitlesSubtitleSettings,
  SubtitlesUpload,
  SubtitlesValuesBody,
  SubtitlesWorkSubtitles,
  UpdateSubtitleNodeSettings401,
  UpdateSubtitleNodeSettings403,
  UpdateSubtitleNodeSettings404,
  UpdateSubtitleNodeSettings409,
  UpdateSubtitleNodeSettings422,
  UpdateSubtitleNodeSettings500,
  UpdateSubtitleProvider401,
  UpdateSubtitleProvider403,
  UpdateSubtitleProvider404,
  UpdateSubtitleProvider409,
  UpdateSubtitleProvider422,
  UpdateSubtitleProvider500,
  UpdateSubtitleSettings401,
  UpdateSubtitleSettings403,
  UpdateSubtitleSettings409,
  UpdateSubtitleSettings422,
  UpdateSubtitleSettings500,
  UploadSubtitle400,
  UploadSubtitle401,
  UploadSubtitle403,
  UploadSubtitle404,
  UploadSubtitle409,
  UploadSubtitle422,
  UploadSubtitle500,
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

export const getGetSubtitleAttemptUrl = (fileId: string, attemptId: string) => {
  return `/api/v1/files/${fileId}/subtitles/attempts/${attemptId}`;
};

/**
 * @summary Get a subtitle download's or upload's step, and its outcome once finished
 */
export const getSubtitleAttempt = async (
  fileId: string,
  attemptId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesAttempt> => {
  return apiFetch<SubtitlesAttempt>(getGetSubtitleAttemptUrl(fileId, attemptId), {
    ...options,
    method: "GET",
  });
};

export const getGetSubtitleAttemptQueryKey = (fileId: string, attemptId: string) => {
  return [`/api/v1/files/${fileId}/subtitles/attempts/${attemptId}`] as const;
};

export const getGetSubtitleAttemptSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSubtitleAttempt>>,
  TError = ErrorType<
    | GetSubtitleAttempt401
    | GetSubtitleAttempt403
    | GetSubtitleAttempt404
    | GetSubtitleAttempt409
    | GetSubtitleAttempt410
    | GetSubtitleAttempt422
    | GetSubtitleAttempt500
  >,
>(
  fileId: string,
  attemptId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleAttempt>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSubtitleAttemptQueryKey(fileId, attemptId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSubtitleAttempt>>> = ({ signal }) =>
    getSubtitleAttempt(fileId, attemptId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleAttempt>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSubtitleAttemptSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getSubtitleAttempt>>
>;
export type GetSubtitleAttemptSuspenseQueryError = ErrorType<
  | GetSubtitleAttempt401
  | GetSubtitleAttempt403
  | GetSubtitleAttempt404
  | GetSubtitleAttempt409
  | GetSubtitleAttempt410
  | GetSubtitleAttempt422
  | GetSubtitleAttempt500
>;

export function useGetSubtitleAttemptSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleAttempt>>,
  TError = ErrorType<
    | GetSubtitleAttempt401
    | GetSubtitleAttempt403
    | GetSubtitleAttempt404
    | GetSubtitleAttempt409
    | GetSubtitleAttempt410
    | GetSubtitleAttempt422
    | GetSubtitleAttempt500
  >,
>(
  fileId: string,
  attemptId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleAttempt>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSubtitleAttemptSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleAttempt>>,
  TError = ErrorType<
    | GetSubtitleAttempt401
    | GetSubtitleAttempt403
    | GetSubtitleAttempt404
    | GetSubtitleAttempt409
    | GetSubtitleAttempt410
    | GetSubtitleAttempt422
    | GetSubtitleAttempt500
  >,
>(
  fileId: string,
  attemptId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleAttempt>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSubtitleAttemptSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleAttempt>>,
  TError = ErrorType<
    | GetSubtitleAttempt401
    | GetSubtitleAttempt403
    | GetSubtitleAttempt404
    | GetSubtitleAttempt409
    | GetSubtitleAttempt410
    | GetSubtitleAttempt422
    | GetSubtitleAttempt500
  >,
>(
  fileId: string,
  attemptId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleAttempt>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a subtitle download's or upload's step, and its outcome once finished
 */

export function useGetSubtitleAttemptSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleAttempt>>,
  TError = ErrorType<
    | GetSubtitleAttempt401
    | GetSubtitleAttempt403
    | GetSubtitleAttempt404
    | GetSubtitleAttempt409
    | GetSubtitleAttempt410
    | GetSubtitleAttempt422
    | GetSubtitleAttempt500
  >,
>(
  fileId: string,
  attemptId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleAttempt>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSubtitleAttemptSuspenseQueryOptions(fileId, attemptId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getCountSubtitleTrackUrl = (fileId: string, stream: number) => {
  return `/api/v1/files/${fileId}/subtitles/tracks/${stream}/ignored`;
};

/**
 * @summary Count an embedded subtitle track toward the file's wants again
 */
export const countSubtitleTrack = async (
  fileId: string,
  stream: number,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getCountSubtitleTrackUrl(fileId, stream), {
    ...options,
    method: "DELETE",
  });
};

export const getCountSubtitleTrackMutationKey = () => ["countSubtitleTrack"] as const;

export const getCountSubtitleTrackMutationOptions = <
  TError = ErrorType<
    | CountSubtitleTrack401
    | CountSubtitleTrack403
    | CountSubtitleTrack404
    | CountSubtitleTrack409
    | CountSubtitleTrack422
    | CountSubtitleTrack500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof countSubtitleTrack>>,
    TError,
    CountSubtitleTrackMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof countSubtitleTrack>>,
  TError,
  CountSubtitleTrackMutationVariables,
  TContext
> => {
  const mutationKey = getCountSubtitleTrackMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof countSubtitleTrack>>,
    CountSubtitleTrackMutationVariables
  > = (props) => {
    const { fileId, stream } = props ?? {};

    return countSubtitleTrack(fileId, stream, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type CountSubtitleTrackMutationResult = NonNullable<
  Awaited<ReturnType<typeof countSubtitleTrack>>
>;

export type CountSubtitleTrackMutationError = ErrorType<
  | CountSubtitleTrack401
  | CountSubtitleTrack403
  | CountSubtitleTrack404
  | CountSubtitleTrack409
  | CountSubtitleTrack422
  | CountSubtitleTrack500
>;
export type CountSubtitleTrackMutationVariables = { fileId: string; stream: number };

/**
 * @summary Count an embedded subtitle track toward the file's wants again
 */
export const useCountSubtitleTrack = <
  TError = ErrorType<
    | CountSubtitleTrack401
    | CountSubtitleTrack403
    | CountSubtitleTrack404
    | CountSubtitleTrack409
    | CountSubtitleTrack422
    | CountSubtitleTrack500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof countSubtitleTrack>>,
      TError,
      CountSubtitleTrackMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof countSubtitleTrack>>,
  TError,
  CountSubtitleTrackMutationVariables,
  TContext
> => {
  return useMutation(getCountSubtitleTrackMutationOptions(options), queryClient);
};
export const getIgnoreSubtitleTrackUrl = (fileId: string, stream: number) => {
  return `/api/v1/files/${fileId}/subtitles/tracks/${stream}/ignored`;
};

/**
 * @summary Don't count an embedded subtitle track toward the file's wants
 */
export const ignoreSubtitleTrack = async (
  fileId: string,
  stream: number,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getIgnoreSubtitleTrackUrl(fileId, stream), {
    ...options,
    method: "PUT",
  });
};

export const getIgnoreSubtitleTrackMutationKey = () => ["ignoreSubtitleTrack"] as const;

export const getIgnoreSubtitleTrackMutationOptions = <
  TError = ErrorType<
    | IgnoreSubtitleTrack401
    | IgnoreSubtitleTrack403
    | IgnoreSubtitleTrack404
    | IgnoreSubtitleTrack409
    | IgnoreSubtitleTrack422
    | IgnoreSubtitleTrack500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof ignoreSubtitleTrack>>,
    TError,
    IgnoreSubtitleTrackMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof ignoreSubtitleTrack>>,
  TError,
  IgnoreSubtitleTrackMutationVariables,
  TContext
> => {
  const mutationKey = getIgnoreSubtitleTrackMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof ignoreSubtitleTrack>>,
    IgnoreSubtitleTrackMutationVariables
  > = (props) => {
    const { fileId, stream } = props ?? {};

    return ignoreSubtitleTrack(fileId, stream, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type IgnoreSubtitleTrackMutationResult = NonNullable<
  Awaited<ReturnType<typeof ignoreSubtitleTrack>>
>;

export type IgnoreSubtitleTrackMutationError = ErrorType<
  | IgnoreSubtitleTrack401
  | IgnoreSubtitleTrack403
  | IgnoreSubtitleTrack404
  | IgnoreSubtitleTrack409
  | IgnoreSubtitleTrack422
  | IgnoreSubtitleTrack500
>;
export type IgnoreSubtitleTrackMutationVariables = { fileId: string; stream: number };

/**
 * @summary Don't count an embedded subtitle track toward the file's wants
 */
export const useIgnoreSubtitleTrack = <
  TError = ErrorType<
    | IgnoreSubtitleTrack401
    | IgnoreSubtitleTrack403
    | IgnoreSubtitleTrack404
    | IgnoreSubtitleTrack409
    | IgnoreSubtitleTrack422
    | IgnoreSubtitleTrack500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof ignoreSubtitleTrack>>,
      TError,
      IgnoreSubtitleTrackMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof ignoreSubtitleTrack>>,
  TError,
  IgnoreSubtitleTrackMutationVariables,
  TContext
> => {
  return useMutation(getIgnoreSubtitleTrackMutationOptions(options), queryClient);
};
export const getKeepSubtitleUploadUrl = (fileId: string, uploadId: string) => {
  return `/api/v1/files/${fileId}/subtitles/uploads/${uploadId}/keep`;
};

/**
 * @summary Keep a refused upload without a timing check
 */
export const keepSubtitleUpload = async (
  fileId: string,
  uploadId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getKeepSubtitleUploadUrl(fileId, uploadId), {
    ...options,
    method: "POST",
  });
};

export const getKeepSubtitleUploadMutationKey = () => ["keepSubtitleUpload"] as const;

export const getKeepSubtitleUploadMutationOptions = <
  TError = ErrorType<
    | KeepSubtitleUpload401
    | KeepSubtitleUpload403
    | KeepSubtitleUpload404
    | KeepSubtitleUpload409
    | KeepSubtitleUpload410
    | KeepSubtitleUpload422
    | KeepSubtitleUpload500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof keepSubtitleUpload>>,
    TError,
    KeepSubtitleUploadMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof keepSubtitleUpload>>,
  TError,
  KeepSubtitleUploadMutationVariables,
  TContext
> => {
  const mutationKey = getKeepSubtitleUploadMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof keepSubtitleUpload>>,
    KeepSubtitleUploadMutationVariables
  > = (props) => {
    const { fileId, uploadId } = props ?? {};

    return keepSubtitleUpload(fileId, uploadId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type KeepSubtitleUploadMutationResult = NonNullable<
  Awaited<ReturnType<typeof keepSubtitleUpload>>
>;

export type KeepSubtitleUploadMutationError = ErrorType<
  | KeepSubtitleUpload401
  | KeepSubtitleUpload403
  | KeepSubtitleUpload404
  | KeepSubtitleUpload409
  | KeepSubtitleUpload410
  | KeepSubtitleUpload422
  | KeepSubtitleUpload500
>;
export type KeepSubtitleUploadMutationVariables = { fileId: string; uploadId: string };

/**
 * @summary Keep a refused upload without a timing check
 */
export const useKeepSubtitleUpload = <
  TError = ErrorType<
    | KeepSubtitleUpload401
    | KeepSubtitleUpload403
    | KeepSubtitleUpload404
    | KeepSubtitleUpload409
    | KeepSubtitleUpload410
    | KeepSubtitleUpload422
    | KeepSubtitleUpload500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof keepSubtitleUpload>>,
      TError,
      KeepSubtitleUploadMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof keepSubtitleUpload>>,
  TError,
  KeepSubtitleUploadMutationVariables,
  TContext
> => {
  return useMutation(getKeepSubtitleUploadMutationOptions(options), queryClient);
};
export const getStartSubtitleCandidatesUrl = (fileId: string, want: string) => {
  return `/api/v1/files/${fileId}/subtitles/${want}/candidates`;
};

/**
 * @summary Start a Manual subtitle search on one file's wanted subtitle
 */
export const startSubtitleCandidates = async (
  fileId: string,
  want: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesStarted> => {
  return apiFetch<SubtitlesStarted>(getStartSubtitleCandidatesUrl(fileId, want), {
    ...options,
    method: "POST",
  });
};

export const getStartSubtitleCandidatesMutationKey = () => ["startSubtitleCandidates"] as const;

export const getStartSubtitleCandidatesMutationOptions = <
  TError = ErrorType<
    | StartSubtitleCandidates400
    | StartSubtitleCandidates401
    | StartSubtitleCandidates403
    | StartSubtitleCandidates404
    | StartSubtitleCandidates409
    | StartSubtitleCandidates422
    | StartSubtitleCandidates500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof startSubtitleCandidates>>,
    TError,
    StartSubtitleCandidatesMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof startSubtitleCandidates>>,
  TError,
  StartSubtitleCandidatesMutationVariables,
  TContext
> => {
  const mutationKey = getStartSubtitleCandidatesMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof startSubtitleCandidates>>,
    StartSubtitleCandidatesMutationVariables
  > = (props) => {
    const { fileId, want } = props ?? {};

    return startSubtitleCandidates(fileId, want, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type StartSubtitleCandidatesMutationResult = NonNullable<
  Awaited<ReturnType<typeof startSubtitleCandidates>>
>;

export type StartSubtitleCandidatesMutationError = ErrorType<
  | StartSubtitleCandidates400
  | StartSubtitleCandidates401
  | StartSubtitleCandidates403
  | StartSubtitleCandidates404
  | StartSubtitleCandidates409
  | StartSubtitleCandidates422
  | StartSubtitleCandidates500
>;
export type StartSubtitleCandidatesMutationVariables = { fileId: string; want: string };

/**
 * @summary Start a Manual subtitle search on one file's wanted subtitle
 */
export const useStartSubtitleCandidates = <
  TError = ErrorType<
    | StartSubtitleCandidates400
    | StartSubtitleCandidates401
    | StartSubtitleCandidates403
    | StartSubtitleCandidates404
    | StartSubtitleCandidates409
    | StartSubtitleCandidates422
    | StartSubtitleCandidates500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof startSubtitleCandidates>>,
      TError,
      StartSubtitleCandidatesMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof startSubtitleCandidates>>,
  TError,
  StartSubtitleCandidatesMutationVariables,
  TContext
> => {
  return useMutation(getStartSubtitleCandidatesMutationOptions(options), queryClient);
};
export const getGetSubtitleCandidatesUrl = (fileId: string, want: string, searchId: string) => {
  return `/api/v1/files/${fileId}/subtitles/${want}/candidates/${searchId}`;
};

/**
 * @summary Get a Manual subtitle search's ranked candidates so far
 */
export const getSubtitleCandidates = async (
  fileId: string,
  want: string,
  searchId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesCandidates> => {
  return apiFetch<SubtitlesCandidates>(getGetSubtitleCandidatesUrl(fileId, want, searchId), {
    ...options,
    method: "GET",
  });
};

export const getGetSubtitleCandidatesQueryKey = (
  fileId: string,
  want: string,
  searchId: string,
) => {
  return [`/api/v1/files/${fileId}/subtitles/${want}/candidates/${searchId}`] as const;
};

export const getGetSubtitleCandidatesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSubtitleCandidates>>,
  TError = ErrorType<
    | GetSubtitleCandidates400
    | GetSubtitleCandidates401
    | GetSubtitleCandidates403
    | GetSubtitleCandidates404
    | GetSubtitleCandidates409
    | GetSubtitleCandidates410
    | GetSubtitleCandidates422
    | GetSubtitleCandidates500
  >,
>(
  fileId: string,
  want: string,
  searchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleCandidates>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey =
    queryOptions?.queryKey ?? getGetSubtitleCandidatesQueryKey(fileId, want, searchId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSubtitleCandidates>>> = ({ signal }) =>
    getSubtitleCandidates(fileId, want, searchId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof getSubtitleCandidates>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSubtitleCandidatesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getSubtitleCandidates>>
>;
export type GetSubtitleCandidatesSuspenseQueryError = ErrorType<
  | GetSubtitleCandidates400
  | GetSubtitleCandidates401
  | GetSubtitleCandidates403
  | GetSubtitleCandidates404
  | GetSubtitleCandidates409
  | GetSubtitleCandidates410
  | GetSubtitleCandidates422
  | GetSubtitleCandidates500
>;

export function useGetSubtitleCandidatesSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleCandidates>>,
  TError = ErrorType<
    | GetSubtitleCandidates400
    | GetSubtitleCandidates401
    | GetSubtitleCandidates403
    | GetSubtitleCandidates404
    | GetSubtitleCandidates409
    | GetSubtitleCandidates410
    | GetSubtitleCandidates422
    | GetSubtitleCandidates500
  >,
>(
  fileId: string,
  want: string,
  searchId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleCandidates>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSubtitleCandidatesSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleCandidates>>,
  TError = ErrorType<
    | GetSubtitleCandidates400
    | GetSubtitleCandidates401
    | GetSubtitleCandidates403
    | GetSubtitleCandidates404
    | GetSubtitleCandidates409
    | GetSubtitleCandidates410
    | GetSubtitleCandidates422
    | GetSubtitleCandidates500
  >,
>(
  fileId: string,
  want: string,
  searchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleCandidates>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSubtitleCandidatesSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleCandidates>>,
  TError = ErrorType<
    | GetSubtitleCandidates400
    | GetSubtitleCandidates401
    | GetSubtitleCandidates403
    | GetSubtitleCandidates404
    | GetSubtitleCandidates409
    | GetSubtitleCandidates410
    | GetSubtitleCandidates422
    | GetSubtitleCandidates500
  >,
>(
  fileId: string,
  want: string,
  searchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleCandidates>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Manual subtitle search's ranked candidates so far
 */

export function useGetSubtitleCandidatesSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleCandidates>>,
  TError = ErrorType<
    | GetSubtitleCandidates400
    | GetSubtitleCandidates401
    | GetSubtitleCandidates403
    | GetSubtitleCandidates404
    | GetSubtitleCandidates409
    | GetSubtitleCandidates410
    | GetSubtitleCandidates422
    | GetSubtitleCandidates500
  >,
>(
  fileId: string,
  want: string,
  searchId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleCandidates>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSubtitleCandidatesSuspenseQueryOptions(
    fileId,
    want,
    searchId,
    options,
  );

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getPickSubtitleCandidateUrl = (
  fileId: string,
  want: string,
  searchId: string,
  key: string,
) => {
  return `/api/v1/files/${fileId}/subtitles/${want}/candidates/${searchId}/${key}`;
};

/**
 * @summary Download a candidate and check its timing against the file
 */
export const pickSubtitleCandidate = async (
  fileId: string,
  want: string,
  searchId: string,
  key: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesAttempting> => {
  return apiFetch<SubtitlesAttempting>(getPickSubtitleCandidateUrl(fileId, want, searchId, key), {
    ...options,
    method: "POST",
  });
};

export const getPickSubtitleCandidateMutationKey = () => ["pickSubtitleCandidate"] as const;

export const getPickSubtitleCandidateMutationOptions = <
  TError = ErrorType<
    | PickSubtitleCandidate400
    | PickSubtitleCandidate401
    | PickSubtitleCandidate403
    | PickSubtitleCandidate404
    | PickSubtitleCandidate409
    | PickSubtitleCandidate410
    | PickSubtitleCandidate422
    | PickSubtitleCandidate500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof pickSubtitleCandidate>>,
    TError,
    PickSubtitleCandidateMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof pickSubtitleCandidate>>,
  TError,
  PickSubtitleCandidateMutationVariables,
  TContext
> => {
  const mutationKey = getPickSubtitleCandidateMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof pickSubtitleCandidate>>,
    PickSubtitleCandidateMutationVariables
  > = (props) => {
    const { fileId, want, searchId, key } = props ?? {};

    return pickSubtitleCandidate(fileId, want, searchId, key, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type PickSubtitleCandidateMutationResult = NonNullable<
  Awaited<ReturnType<typeof pickSubtitleCandidate>>
>;

export type PickSubtitleCandidateMutationError = ErrorType<
  | PickSubtitleCandidate400
  | PickSubtitleCandidate401
  | PickSubtitleCandidate403
  | PickSubtitleCandidate404
  | PickSubtitleCandidate409
  | PickSubtitleCandidate410
  | PickSubtitleCandidate422
  | PickSubtitleCandidate500
>;
export type PickSubtitleCandidateMutationVariables = {
  fileId: string;
  want: string;
  searchId: string;
  key: string;
};

/**
 * @summary Download a candidate and check its timing against the file
 */
export const usePickSubtitleCandidate = <
  TError = ErrorType<
    | PickSubtitleCandidate400
    | PickSubtitleCandidate401
    | PickSubtitleCandidate403
    | PickSubtitleCandidate404
    | PickSubtitleCandidate409
    | PickSubtitleCandidate410
    | PickSubtitleCandidate422
    | PickSubtitleCandidate500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof pickSubtitleCandidate>>,
      TError,
      PickSubtitleCandidateMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof pickSubtitleCandidate>>,
  TError,
  PickSubtitleCandidateMutationVariables,
  TContext
> => {
  return useMutation(getPickSubtitleCandidateMutationOptions(options), queryClient);
};
export const getMarkSubtitleBadUrl = (fileId: string, want: string) => {
  return `/api/v1/files/${fileId}/subtitles/${want}/mark-bad`;
};

/**
 * @summary Delete a file's subtitle, never use it for the file again and search for another
 */
export const markSubtitleBad = async (
  fileId: string,
  want: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getMarkSubtitleBadUrl(fileId, want), {
    ...options,
    method: "POST",
  });
};

export const getMarkSubtitleBadMutationKey = () => ["markSubtitleBad"] as const;

export const getMarkSubtitleBadMutationOptions = <
  TError = ErrorType<
    | MarkSubtitleBad400
    | MarkSubtitleBad401
    | MarkSubtitleBad403
    | MarkSubtitleBad404
    | MarkSubtitleBad409
    | MarkSubtitleBad422
    | MarkSubtitleBad500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof markSubtitleBad>>,
    TError,
    MarkSubtitleBadMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof markSubtitleBad>>,
  TError,
  MarkSubtitleBadMutationVariables,
  TContext
> => {
  const mutationKey = getMarkSubtitleBadMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof markSubtitleBad>>,
    MarkSubtitleBadMutationVariables
  > = (props) => {
    const { fileId, want } = props ?? {};

    return markSubtitleBad(fileId, want, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type MarkSubtitleBadMutationResult = NonNullable<
  Awaited<ReturnType<typeof markSubtitleBad>>
>;

export type MarkSubtitleBadMutationError = ErrorType<
  | MarkSubtitleBad400
  | MarkSubtitleBad401
  | MarkSubtitleBad403
  | MarkSubtitleBad404
  | MarkSubtitleBad409
  | MarkSubtitleBad422
  | MarkSubtitleBad500
>;
export type MarkSubtitleBadMutationVariables = { fileId: string; want: string };

/**
 * @summary Delete a file's subtitle, never use it for the file again and search for another
 */
export const useMarkSubtitleBad = <
  TError = ErrorType<
    | MarkSubtitleBad400
    | MarkSubtitleBad401
    | MarkSubtitleBad403
    | MarkSubtitleBad404
    | MarkSubtitleBad409
    | MarkSubtitleBad422
    | MarkSubtitleBad500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof markSubtitleBad>>,
      TError,
      MarkSubtitleBadMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof markSubtitleBad>>,
  TError,
  MarkSubtitleBadMutationVariables,
  TContext
> => {
  return useMutation(getMarkSubtitleBadMutationOptions(options), queryClient);
};
export const getSearchSubtitleUrl = (fileId: string, want: string) => {
  return `/api/v1/files/${fileId}/subtitles/${want}/search`;
};

/**
 * @summary Search for one file's wanted subtitle now, ahead of the schedule
 */
export const searchSubtitle = async (
  fileId: string,
  want: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<void> => {
  return apiFetch<void>(getSearchSubtitleUrl(fileId, want), {
    ...options,
    method: "POST",
  });
};

export const getSearchSubtitleMutationKey = () => ["searchSubtitle"] as const;

export const getSearchSubtitleMutationOptions = <
  TError = ErrorType<
    | SearchSubtitle400
    | SearchSubtitle401
    | SearchSubtitle403
    | SearchSubtitle404
    | SearchSubtitle409
    | SearchSubtitle422
    | SearchSubtitle500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof searchSubtitle>>,
    TError,
    SearchSubtitleMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof searchSubtitle>>,
  TError,
  SearchSubtitleMutationVariables,
  TContext
> => {
  const mutationKey = getSearchSubtitleMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof searchSubtitle>>,
    SearchSubtitleMutationVariables
  > = (props) => {
    const { fileId, want } = props ?? {};

    return searchSubtitle(fileId, want, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SearchSubtitleMutationResult = NonNullable<Awaited<ReturnType<typeof searchSubtitle>>>;

export type SearchSubtitleMutationError = ErrorType<
  | SearchSubtitle400
  | SearchSubtitle401
  | SearchSubtitle403
  | SearchSubtitle404
  | SearchSubtitle409
  | SearchSubtitle422
  | SearchSubtitle500
>;
export type SearchSubtitleMutationVariables = { fileId: string; want: string };

/**
 * @summary Search for one file's wanted subtitle now, ahead of the schedule
 */
export const useSearchSubtitle = <
  TError = ErrorType<
    | SearchSubtitle400
    | SearchSubtitle401
    | SearchSubtitle403
    | SearchSubtitle404
    | SearchSubtitle409
    | SearchSubtitle422
    | SearchSubtitle500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof searchSubtitle>>,
      TError,
      SearchSubtitleMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof searchSubtitle>>,
  TError,
  SearchSubtitleMutationVariables,
  TContext
> => {
  return useMutation(getSearchSubtitleMutationOptions(options), queryClient);
};
export const getUploadSubtitleUrl = (fileId: string, want: string) => {
  return `/api/v1/files/${fileId}/subtitles/${want}/upload`;
};

/**
 * @summary Upload a subtitle file and check its timing against the file
 */
export const uploadSubtitle = async (
  fileId: string,
  want: string,
  subtitlesUpload: SubtitlesUpload,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesAttempting> => {
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
  return apiFetch<SubtitlesAttempting>(getUploadSubtitleUrl(fileId, want), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(subtitlesUpload),
  });
};

export const getUploadSubtitleMutationKey = () => ["uploadSubtitle"] as const;

export const getUploadSubtitleMutationOptions = <
  TError = ErrorType<
    | UploadSubtitle400
    | UploadSubtitle401
    | UploadSubtitle403
    | UploadSubtitle404
    | UploadSubtitle409
    | UploadSubtitle422
    | UploadSubtitle500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof uploadSubtitle>>,
    TError,
    UploadSubtitleMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof uploadSubtitle>>,
  TError,
  UploadSubtitleMutationVariables,
  TContext
> => {
  const mutationKey = getUploadSubtitleMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof uploadSubtitle>>,
    UploadSubtitleMutationVariables
  > = (props) => {
    const { fileId, want, data } = props ?? {};

    return uploadSubtitle(fileId, want, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UploadSubtitleMutationResult = NonNullable<Awaited<ReturnType<typeof uploadSubtitle>>>;
export type UploadSubtitleMutationBody = SubtitlesUpload;
export type UploadSubtitleMutationError = ErrorType<
  | UploadSubtitle400
  | UploadSubtitle401
  | UploadSubtitle403
  | UploadSubtitle404
  | UploadSubtitle409
  | UploadSubtitle422
  | UploadSubtitle500
>;
export type UploadSubtitleMutationVariables = {
  fileId: string;
  want: string;
  data: SubtitlesUpload;
};

/**
 * @summary Upload a subtitle file and check its timing against the file
 */
export const useUploadSubtitle = <
  TError = ErrorType<
    | UploadSubtitle400
    | UploadSubtitle401
    | UploadSubtitle403
    | UploadSubtitle404
    | UploadSubtitle409
    | UploadSubtitle422
    | UploadSubtitle500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof uploadSubtitle>>,
      TError,
      UploadSubtitleMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof uploadSubtitle>>,
  TError,
  UploadSubtitleMutationVariables,
  TContext
> => {
  return useMutation(getUploadSubtitleMutationOptions(options), queryClient);
};
export const getGetSubtitleNodeSettingsUrl = (nodeId: string) => {
  return `/api/v1/subtitles/nodes/${nodeId}/settings`;
};

/**
 * @summary Get a Storage Node's Subtitles settings
 */
export const getSubtitleNodeSettings = async (
  nodeId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesNodeSettings> => {
  return apiFetch<SubtitlesNodeSettings>(getGetSubtitleNodeSettingsUrl(nodeId), {
    ...options,
    method: "GET",
  });
};

export const getGetSubtitleNodeSettingsQueryKey = (nodeId: string) => {
  return [`/api/v1/subtitles/nodes/${nodeId}/settings`] as const;
};

export const getGetSubtitleNodeSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSubtitleNodeSettings>>,
  TError = ErrorType<
    | GetSubtitleNodeSettings401
    | GetSubtitleNodeSettings403
    | GetSubtitleNodeSettings404
    | GetSubtitleNodeSettings409
    | GetSubtitleNodeSettings422
    | GetSubtitleNodeSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleNodeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSubtitleNodeSettingsQueryKey(nodeId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSubtitleNodeSettings>>> = ({
    signal,
  }) => getSubtitleNodeSettings(nodeId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof getSubtitleNodeSettings>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSubtitleNodeSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getSubtitleNodeSettings>>
>;
export type GetSubtitleNodeSettingsSuspenseQueryError = ErrorType<
  | GetSubtitleNodeSettings401
  | GetSubtitleNodeSettings403
  | GetSubtitleNodeSettings404
  | GetSubtitleNodeSettings409
  | GetSubtitleNodeSettings422
  | GetSubtitleNodeSettings500
>;

export function useGetSubtitleNodeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleNodeSettings>>,
  TError = ErrorType<
    | GetSubtitleNodeSettings401
    | GetSubtitleNodeSettings403
    | GetSubtitleNodeSettings404
    | GetSubtitleNodeSettings409
    | GetSubtitleNodeSettings422
    | GetSubtitleNodeSettings500
  >,
>(
  nodeId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleNodeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSubtitleNodeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleNodeSettings>>,
  TError = ErrorType<
    | GetSubtitleNodeSettings401
    | GetSubtitleNodeSettings403
    | GetSubtitleNodeSettings404
    | GetSubtitleNodeSettings409
    | GetSubtitleNodeSettings422
    | GetSubtitleNodeSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleNodeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSubtitleNodeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleNodeSettings>>,
  TError = ErrorType<
    | GetSubtitleNodeSettings401
    | GetSubtitleNodeSettings403
    | GetSubtitleNodeSettings404
    | GetSubtitleNodeSettings409
    | GetSubtitleNodeSettings422
    | GetSubtitleNodeSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleNodeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get a Storage Node's Subtitles settings
 */

export function useGetSubtitleNodeSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleNodeSettings>>,
  TError = ErrorType<
    | GetSubtitleNodeSettings401
    | GetSubtitleNodeSettings403
    | GetSubtitleNodeSettings404
    | GetSubtitleNodeSettings409
    | GetSubtitleNodeSettings422
    | GetSubtitleNodeSettings500
  >,
>(
  nodeId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleNodeSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSubtitleNodeSettingsSuspenseQueryOptions(nodeId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateSubtitleNodeSettingsUrl = (nodeId: string) => {
  return `/api/v1/subtitles/nodes/${nodeId}/settings`;
};

/**
 * @summary Change a Storage Node's Subtitles settings by key, all or none
 */
export const updateSubtitleNodeSettings = async (
  nodeId: string,
  subtitlesValuesBody: SubtitlesValuesBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesNodeSettings> => {
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
  return apiFetch<SubtitlesNodeSettings>(getUpdateSubtitleNodeSettingsUrl(nodeId), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(subtitlesValuesBody),
  });
};

export const getUpdateSubtitleNodeSettingsMutationKey = () =>
  ["updateSubtitleNodeSettings"] as const;

export const getUpdateSubtitleNodeSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateSubtitleNodeSettings401
    | UpdateSubtitleNodeSettings403
    | UpdateSubtitleNodeSettings404
    | UpdateSubtitleNodeSettings409
    | UpdateSubtitleNodeSettings422
    | UpdateSubtitleNodeSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateSubtitleNodeSettings>>,
    TError,
    UpdateSubtitleNodeSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateSubtitleNodeSettings>>,
  TError,
  UpdateSubtitleNodeSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateSubtitleNodeSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateSubtitleNodeSettings>>,
    UpdateSubtitleNodeSettingsMutationVariables
  > = (props) => {
    const { nodeId, data } = props ?? {};

    return updateSubtitleNodeSettings(nodeId, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateSubtitleNodeSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateSubtitleNodeSettings>>
>;
export type UpdateSubtitleNodeSettingsMutationBody = SubtitlesValuesBody;
export type UpdateSubtitleNodeSettingsMutationError = ErrorType<
  | UpdateSubtitleNodeSettings401
  | UpdateSubtitleNodeSettings403
  | UpdateSubtitleNodeSettings404
  | UpdateSubtitleNodeSettings409
  | UpdateSubtitleNodeSettings422
  | UpdateSubtitleNodeSettings500
>;
export type UpdateSubtitleNodeSettingsMutationVariables = {
  nodeId: string;
  data: SubtitlesValuesBody;
};

/**
 * @summary Change a Storage Node's Subtitles settings by key, all or none
 */
export const useUpdateSubtitleNodeSettings = <
  TError = ErrorType<
    | UpdateSubtitleNodeSettings401
    | UpdateSubtitleNodeSettings403
    | UpdateSubtitleNodeSettings404
    | UpdateSubtitleNodeSettings409
    | UpdateSubtitleNodeSettings422
    | UpdateSubtitleNodeSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateSubtitleNodeSettings>>,
      TError,
      UpdateSubtitleNodeSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateSubtitleNodeSettings>>,
  TError,
  UpdateSubtitleNodeSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateSubtitleNodeSettingsMutationOptions(options), queryClient);
};
export const getListSubtitleProvidersUrl = () => {
  return `/api/v1/subtitles/providers`;
};

/**
 * @summary List the Subtitle Providers in search order
 */
export const listSubtitleProviders = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesProviderList> => {
  return apiFetch<SubtitlesProviderList>(getListSubtitleProvidersUrl(), {
    ...options,
    method: "GET",
  });
};

export const getListSubtitleProvidersQueryKey = () => {
  return [`/api/v1/subtitles/providers`] as const;
};

export const getListSubtitleProvidersSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof listSubtitleProviders>>,
  TError = ErrorType<
    | ListSubtitleProviders401
    | ListSubtitleProviders403
    | ListSubtitleProviders409
    | ListSubtitleProviders500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSubtitleProviders>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getListSubtitleProvidersQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof listSubtitleProviders>>> = ({ signal }) =>
    listSubtitleProviders({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<
    Awaited<ReturnType<typeof listSubtitleProviders>>,
    TError,
    TData
  > & { queryKey: DataTag<QueryKey, TData, TError> } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type ListSubtitleProvidersSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof listSubtitleProviders>>
>;
export type ListSubtitleProvidersSuspenseQueryError = ErrorType<
  | ListSubtitleProviders401
  | ListSubtitleProviders403
  | ListSubtitleProviders409
  | ListSubtitleProviders500
>;

export function useListSubtitleProvidersSuspense<
  TData = Awaited<ReturnType<typeof listSubtitleProviders>>,
  TError = ErrorType<
    | ListSubtitleProviders401
    | ListSubtitleProviders403
    | ListSubtitleProviders409
    | ListSubtitleProviders500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSubtitleProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListSubtitleProvidersSuspense<
  TData = Awaited<ReturnType<typeof listSubtitleProviders>>,
  TError = ErrorType<
    | ListSubtitleProviders401
    | ListSubtitleProviders403
    | ListSubtitleProviders409
    | ListSubtitleProviders500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSubtitleProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useListSubtitleProvidersSuspense<
  TData = Awaited<ReturnType<typeof listSubtitleProviders>>,
  TError = ErrorType<
    | ListSubtitleProviders401
    | ListSubtitleProviders403
    | ListSubtitleProviders409
    | ListSubtitleProviders500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSubtitleProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary List the Subtitle Providers in search order
 */

export function useListSubtitleProvidersSuspense<
  TData = Awaited<ReturnType<typeof listSubtitleProviders>>,
  TError = ErrorType<
    | ListSubtitleProviders401
    | ListSubtitleProviders403
    | ListSubtitleProviders409
    | ListSubtitleProviders500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof listSubtitleProviders>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getListSubtitleProvidersSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getOrderSubtitleProvidersUrl = () => {
  return `/api/v1/subtitles/providers/order`;
};

/**
 * @summary Set the Subtitle Providers' search order
 */
export const orderSubtitleProviders = async (
  subtitlesProviderOrder: SubtitlesProviderOrder,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesProviderList> => {
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
  return apiFetch<SubtitlesProviderList>(getOrderSubtitleProvidersUrl(), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(subtitlesProviderOrder),
  });
};

export const getOrderSubtitleProvidersMutationKey = () => ["orderSubtitleProviders"] as const;

export const getOrderSubtitleProvidersMutationOptions = <
  TError = ErrorType<
    | OrderSubtitleProviders401
    | OrderSubtitleProviders403
    | OrderSubtitleProviders409
    | OrderSubtitleProviders422
    | OrderSubtitleProviders500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof orderSubtitleProviders>>,
    TError,
    OrderSubtitleProvidersMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof orderSubtitleProviders>>,
  TError,
  OrderSubtitleProvidersMutationVariables,
  TContext
> => {
  const mutationKey = getOrderSubtitleProvidersMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof orderSubtitleProviders>>,
    OrderSubtitleProvidersMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return orderSubtitleProviders(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type OrderSubtitleProvidersMutationResult = NonNullable<
  Awaited<ReturnType<typeof orderSubtitleProviders>>
>;
export type OrderSubtitleProvidersMutationBody = SubtitlesProviderOrder;
export type OrderSubtitleProvidersMutationError = ErrorType<
  | OrderSubtitleProviders401
  | OrderSubtitleProviders403
  | OrderSubtitleProviders409
  | OrderSubtitleProviders422
  | OrderSubtitleProviders500
>;
export type OrderSubtitleProvidersMutationVariables = { data: SubtitlesProviderOrder };

/**
 * @summary Set the Subtitle Providers' search order
 */
export const useOrderSubtitleProviders = <
  TError = ErrorType<
    | OrderSubtitleProviders401
    | OrderSubtitleProviders403
    | OrderSubtitleProviders409
    | OrderSubtitleProviders422
    | OrderSubtitleProviders500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof orderSubtitleProviders>>,
      TError,
      OrderSubtitleProvidersMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof orderSubtitleProviders>>,
  TError,
  OrderSubtitleProvidersMutationVariables,
  TContext
> => {
  return useMutation(getOrderSubtitleProvidersMutationOptions(options), queryClient);
};
export const getUpdateSubtitleProviderUrl = (id: string) => {
  return `/api/v1/subtitles/providers/${id}`;
};

/**
 * @summary Turn a Subtitle Provider on or off
 */
export const updateSubtitleProvider = async (
  id: string,
  subtitlesProviderChange: SubtitlesProviderChange,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesProvider> => {
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
  return apiFetch<SubtitlesProvider>(getUpdateSubtitleProviderUrl(id), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(subtitlesProviderChange),
  });
};

export const getUpdateSubtitleProviderMutationKey = () => ["updateSubtitleProvider"] as const;

export const getUpdateSubtitleProviderMutationOptions = <
  TError = ErrorType<
    | UpdateSubtitleProvider401
    | UpdateSubtitleProvider403
    | UpdateSubtitleProvider404
    | UpdateSubtitleProvider409
    | UpdateSubtitleProvider422
    | UpdateSubtitleProvider500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateSubtitleProvider>>,
    TError,
    UpdateSubtitleProviderMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateSubtitleProvider>>,
  TError,
  UpdateSubtitleProviderMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateSubtitleProviderMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateSubtitleProvider>>,
    UpdateSubtitleProviderMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return updateSubtitleProvider(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateSubtitleProviderMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateSubtitleProvider>>
>;
export type UpdateSubtitleProviderMutationBody = SubtitlesProviderChange;
export type UpdateSubtitleProviderMutationError = ErrorType<
  | UpdateSubtitleProvider401
  | UpdateSubtitleProvider403
  | UpdateSubtitleProvider404
  | UpdateSubtitleProvider409
  | UpdateSubtitleProvider422
  | UpdateSubtitleProvider500
>;
export type UpdateSubtitleProviderMutationVariables = { id: string; data: SubtitlesProviderChange };

/**
 * @summary Turn a Subtitle Provider on or off
 */
export const useUpdateSubtitleProvider = <
  TError = ErrorType<
    | UpdateSubtitleProvider401
    | UpdateSubtitleProvider403
    | UpdateSubtitleProvider404
    | UpdateSubtitleProvider409
    | UpdateSubtitleProvider422
    | UpdateSubtitleProvider500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateSubtitleProvider>>,
      TError,
      UpdateSubtitleProviderMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateSubtitleProvider>>,
  TError,
  UpdateSubtitleProviderMutationVariables,
  TContext
> => {
  return useMutation(getUpdateSubtitleProviderMutationOptions(options), queryClient);
};
export const getDeleteSubtitleProviderKeyUrl = (id: string) => {
  return `/api/v1/subtitles/providers/${id}/key`;
};

/**
 * @summary Forget a Subtitle Provider's key
 */
export const deleteSubtitleProviderKey = async (
  id: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesProvider> => {
  return apiFetch<SubtitlesProvider>(getDeleteSubtitleProviderKeyUrl(id), {
    ...options,
    method: "DELETE",
  });
};

export const getDeleteSubtitleProviderKeyMutationKey = () => ["deleteSubtitleProviderKey"] as const;

export const getDeleteSubtitleProviderKeyMutationOptions = <
  TError = ErrorType<
    | DeleteSubtitleProviderKey401
    | DeleteSubtitleProviderKey403
    | DeleteSubtitleProviderKey404
    | DeleteSubtitleProviderKey409
    | DeleteSubtitleProviderKey422
    | DeleteSubtitleProviderKey500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof deleteSubtitleProviderKey>>,
    TError,
    DeleteSubtitleProviderKeyMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof deleteSubtitleProviderKey>>,
  TError,
  DeleteSubtitleProviderKeyMutationVariables,
  TContext
> => {
  const mutationKey = getDeleteSubtitleProviderKeyMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof deleteSubtitleProviderKey>>,
    DeleteSubtitleProviderKeyMutationVariables
  > = (props) => {
    const { id } = props ?? {};

    return deleteSubtitleProviderKey(id, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type DeleteSubtitleProviderKeyMutationResult = NonNullable<
  Awaited<ReturnType<typeof deleteSubtitleProviderKey>>
>;

export type DeleteSubtitleProviderKeyMutationError = ErrorType<
  | DeleteSubtitleProviderKey401
  | DeleteSubtitleProviderKey403
  | DeleteSubtitleProviderKey404
  | DeleteSubtitleProviderKey409
  | DeleteSubtitleProviderKey422
  | DeleteSubtitleProviderKey500
>;
export type DeleteSubtitleProviderKeyMutationVariables = { id: string };

/**
 * @summary Forget a Subtitle Provider's key
 */
export const useDeleteSubtitleProviderKey = <
  TError = ErrorType<
    | DeleteSubtitleProviderKey401
    | DeleteSubtitleProviderKey403
    | DeleteSubtitleProviderKey404
    | DeleteSubtitleProviderKey409
    | DeleteSubtitleProviderKey422
    | DeleteSubtitleProviderKey500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof deleteSubtitleProviderKey>>,
      TError,
      DeleteSubtitleProviderKeyMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof deleteSubtitleProviderKey>>,
  TError,
  DeleteSubtitleProviderKeyMutationVariables,
  TContext
> => {
  return useMutation(getDeleteSubtitleProviderKeyMutationOptions(options), queryClient);
};
export const getSetSubtitleProviderKeyUrl = (id: string) => {
  return `/api/v1/subtitles/providers/${id}/key`;
};

/**
 * @summary Check a Subtitle Provider key on a Subtitles holder, then save it
 */
export const setSubtitleProviderKey = async (
  id: string,
  subtitlesProviderKey: SubtitlesProviderKey,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesProvider> => {
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
  return apiFetch<SubtitlesProvider>(getSetSubtitleProviderKeyUrl(id), {
    ...options,
    method: "PUT",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(subtitlesProviderKey),
  });
};

export const getSetSubtitleProviderKeyMutationKey = () => ["setSubtitleProviderKey"] as const;

export const getSetSubtitleProviderKeyMutationOptions = <
  TError = ErrorType<
    | SetSubtitleProviderKey401
    | SetSubtitleProviderKey403
    | SetSubtitleProviderKey404
    | SetSubtitleProviderKey409
    | SetSubtitleProviderKey422
    | SetSubtitleProviderKey500
    | SetSubtitleProviderKey502
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setSubtitleProviderKey>>,
    TError,
    SetSubtitleProviderKeyMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof setSubtitleProviderKey>>,
  TError,
  SetSubtitleProviderKeyMutationVariables,
  TContext
> => {
  const mutationKey = getSetSubtitleProviderKeyMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof setSubtitleProviderKey>>,
    SetSubtitleProviderKeyMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return setSubtitleProviderKey(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type SetSubtitleProviderKeyMutationResult = NonNullable<
  Awaited<ReturnType<typeof setSubtitleProviderKey>>
>;
export type SetSubtitleProviderKeyMutationBody = SubtitlesProviderKey;
export type SetSubtitleProviderKeyMutationError = ErrorType<
  | SetSubtitleProviderKey401
  | SetSubtitleProviderKey403
  | SetSubtitleProviderKey404
  | SetSubtitleProviderKey409
  | SetSubtitleProviderKey422
  | SetSubtitleProviderKey500
  | SetSubtitleProviderKey502
>;
export type SetSubtitleProviderKeyMutationVariables = { id: string; data: SubtitlesProviderKey };

/**
 * @summary Check a Subtitle Provider key on a Subtitles holder, then save it
 */
export const useSetSubtitleProviderKey = <
  TError = ErrorType<
    | SetSubtitleProviderKey401
    | SetSubtitleProviderKey403
    | SetSubtitleProviderKey404
    | SetSubtitleProviderKey409
    | SetSubtitleProviderKey422
    | SetSubtitleProviderKey500
    | SetSubtitleProviderKey502
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof setSubtitleProviderKey>>,
      TError,
      SetSubtitleProviderKeyMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof setSubtitleProviderKey>>,
  TError,
  SetSubtitleProviderKeyMutationVariables,
  TContext
> => {
  return useMutation(getSetSubtitleProviderKeyMutationOptions(options), queryClient);
};
export const getGetSubtitleSettingsUrl = () => {
  return `/api/v1/subtitles/settings`;
};

/**
 * @summary Get the Subtitles settings, the section's lead and the Profiles with Wanted Languages
 */
export const getSubtitleSettings = async (
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesSubtitleSettings> => {
  return apiFetch<SubtitlesSubtitleSettings>(getGetSubtitleSettingsUrl(), {
    ...options,
    method: "GET",
  });
};

export const getGetSubtitleSettingsQueryKey = () => {
  return [`/api/v1/subtitles/settings`] as const;
};

export const getGetSubtitleSettingsSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getSubtitleSettings>>,
  TError = ErrorType<
    | GetSubtitleSettings401
    | GetSubtitleSettings403
    | GetSubtitleSettings409
    | GetSubtitleSettings500
  >,
>(options?: {
  query?: Partial<
    UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleSettings>>, TError, TData>
  >;
  request?: SecondParameter<typeof apiFetch>;
}) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetSubtitleSettingsQueryKey();

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getSubtitleSettings>>> = ({ signal }) =>
    getSubtitleSettings({ signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleSettings>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetSubtitleSettingsSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getSubtitleSettings>>
>;
export type GetSubtitleSettingsSuspenseQueryError = ErrorType<
  GetSubtitleSettings401 | GetSubtitleSettings403 | GetSubtitleSettings409 | GetSubtitleSettings500
>;

export function useGetSubtitleSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleSettings>>,
  TError = ErrorType<
    | GetSubtitleSettings401
    | GetSubtitleSettings403
    | GetSubtitleSettings409
    | GetSubtitleSettings500
  >,
>(
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSubtitleSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleSettings>>,
  TError = ErrorType<
    | GetSubtitleSettings401
    | GetSubtitleSettings403
    | GetSubtitleSettings409
    | GetSubtitleSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetSubtitleSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleSettings>>,
  TError = ErrorType<
    | GetSubtitleSettings401
    | GetSubtitleSettings403
    | GetSubtitleSettings409
    | GetSubtitleSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get the Subtitles settings, the section's lead and the Profiles with Wanted Languages
 */

export function useGetSubtitleSettingsSuspense<
  TData = Awaited<ReturnType<typeof getSubtitleSettings>>,
  TError = ErrorType<
    | GetSubtitleSettings401
    | GetSubtitleSettings403
    | GetSubtitleSettings409
    | GetSubtitleSettings500
  >,
>(
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getSubtitleSettings>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetSubtitleSettingsSuspenseQueryOptions(options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}

export const getUpdateSubtitleSettingsUrl = () => {
  return `/api/v1/subtitles/settings`;
};

/**
 * @summary Change Subtitles settings by key, all or none
 */
export const updateSubtitleSettings = async (
  subtitlesValuesBody: SubtitlesValuesBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesSubtitleSettings> => {
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
  return apiFetch<SubtitlesSubtitleSettings>(getUpdateSubtitleSettingsUrl(), {
    ...options,
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(subtitlesValuesBody),
  });
};

export const getUpdateSubtitleSettingsMutationKey = () => ["updateSubtitleSettings"] as const;

export const getUpdateSubtitleSettingsMutationOptions = <
  TError = ErrorType<
    | UpdateSubtitleSettings401
    | UpdateSubtitleSettings403
    | UpdateSubtitleSettings409
    | UpdateSubtitleSettings422
    | UpdateSubtitleSettings500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof updateSubtitleSettings>>,
    TError,
    UpdateSubtitleSettingsMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof updateSubtitleSettings>>,
  TError,
  UpdateSubtitleSettingsMutationVariables,
  TContext
> => {
  const mutationKey = getUpdateSubtitleSettingsMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof updateSubtitleSettings>>,
    UpdateSubtitleSettingsMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return updateSubtitleSettings(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UpdateSubtitleSettingsMutationResult = NonNullable<
  Awaited<ReturnType<typeof updateSubtitleSettings>>
>;
export type UpdateSubtitleSettingsMutationBody = SubtitlesValuesBody;
export type UpdateSubtitleSettingsMutationError = ErrorType<
  | UpdateSubtitleSettings401
  | UpdateSubtitleSettings403
  | UpdateSubtitleSettings409
  | UpdateSubtitleSettings422
  | UpdateSubtitleSettings500
>;
export type UpdateSubtitleSettingsMutationVariables = { data: SubtitlesValuesBody };

/**
 * @summary Change Subtitles settings by key, all or none
 */
export const useUpdateSubtitleSettings = <
  TError = ErrorType<
    | UpdateSubtitleSettings401
    | UpdateSubtitleSettings403
    | UpdateSubtitleSettings409
    | UpdateSubtitleSettings422
    | UpdateSubtitleSettings500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof updateSubtitleSettings>>,
      TError,
      UpdateSubtitleSettingsMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof updateSubtitleSettings>>,
  TError,
  UpdateSubtitleSettingsMutationVariables,
  TContext
> => {
  return useMutation(getUpdateSubtitleSettingsMutationOptions(options), queryClient);
};
export const getGetWorkSubtitlesUrl = (workId: string) => {
  return `/api/v1/works/${workId}/subtitles`;
};

/**
 * @summary Get each library file of a Work with what each of its wanted subtitles reads as
 */
export const getWorkSubtitles = async (
  workId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<SubtitlesWorkSubtitles> => {
  return apiFetch<SubtitlesWorkSubtitles>(getGetWorkSubtitlesUrl(workId), {
    ...options,
    method: "GET",
  });
};

export const getGetWorkSubtitlesQueryKey = (workId: string) => {
  return [`/api/v1/works/${workId}/subtitles`] as const;
};

export const getGetWorkSubtitlesSuspenseQueryOptions = <
  TData = Awaited<ReturnType<typeof getWorkSubtitles>>,
  TError = ErrorType<
    | GetWorkSubtitles401
    | GetWorkSubtitles403
    | GetWorkSubtitles404
    | GetWorkSubtitles409
    | GetWorkSubtitles422
    | GetWorkSubtitles500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSubtitles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
) => {
  const { query: queryOptions, request: requestOptions } = options ?? {};

  const queryKey = queryOptions?.queryKey ?? getGetWorkSubtitlesQueryKey(workId);

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getWorkSubtitles>>> = ({ signal }) =>
    getWorkSubtitles(workId, { signal, ...requestOptions });

  return queryOptionsBuilder({
    queryKey,
    ...queryOptions,
    queryFn: queryOptions?.queryFn ?? queryFn,
  }) as UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSubtitles>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>;
  } & {
    throwOnError?: ((this: never, error: TError) => boolean) & { readonly __inferenceOnly: never };
  };
};

export type GetWorkSubtitlesSuspenseQueryResult = NonNullable<
  Awaited<ReturnType<typeof getWorkSubtitles>>
>;
export type GetWorkSubtitlesSuspenseQueryError = ErrorType<
  | GetWorkSubtitles401
  | GetWorkSubtitles403
  | GetWorkSubtitles404
  | GetWorkSubtitles409
  | GetWorkSubtitles422
  | GetWorkSubtitles500
>;

export function useGetWorkSubtitlesSuspense<
  TData = Awaited<ReturnType<typeof getWorkSubtitles>>,
  TError = ErrorType<
    | GetWorkSubtitles401
    | GetWorkSubtitles403
    | GetWorkSubtitles404
    | GetWorkSubtitles409
    | GetWorkSubtitles422
    | GetWorkSubtitles500
  >,
>(
  workId: string,
  options: {
    query: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSubtitles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkSubtitlesSuspense<
  TData = Awaited<ReturnType<typeof getWorkSubtitles>>,
  TError = ErrorType<
    | GetWorkSubtitles401
    | GetWorkSubtitles403
    | GetWorkSubtitles404
    | GetWorkSubtitles409
    | GetWorkSubtitles422
    | GetWorkSubtitles500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSubtitles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
export function useGetWorkSubtitlesSuspense<
  TData = Awaited<ReturnType<typeof getWorkSubtitles>>,
  TError = ErrorType<
    | GetWorkSubtitles401
    | GetWorkSubtitles403
    | GetWorkSubtitles404
    | GetWorkSubtitles409
    | GetWorkSubtitles422
    | GetWorkSubtitles500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSubtitles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };
/**
 * @summary Get each library file of a Work with what each of its wanted subtitles reads as
 */

export function useGetWorkSubtitlesSuspense<
  TData = Awaited<ReturnType<typeof getWorkSubtitles>>,
  TError = ErrorType<
    | GetWorkSubtitles401
    | GetWorkSubtitles403
    | GetWorkSubtitles404
    | GetWorkSubtitles409
    | GetWorkSubtitles422
    | GetWorkSubtitles500
  >,
>(
  workId: string,
  options?: {
    query?: Partial<
      UseSuspenseQueryOptions<Awaited<ReturnType<typeof getWorkSubtitles>>, TError, TData>
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseSuspenseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryOptions = getGetWorkSubtitlesSuspenseQueryOptions(workId, options);

  const query = useSuspenseQuery(queryOptions, queryClient) as UseSuspenseQueryResult<
    TData,
    TError
  > & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}
