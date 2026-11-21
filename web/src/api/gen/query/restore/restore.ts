import { useMutation } from "@tanstack/react-query";
import type {
  MutationFunction,
  QueryClient,
  UseMutationOptions,
  UseMutationResult,
} from "@tanstack/react-query";

import type {
  ConfirmBackupRestore401,
  ConfirmBackupRestore403,
  ConfirmBackupRestore404,
  ConfirmBackupRestore409,
  ConfirmBackupRestore422,
  ConfirmBackupRestore500,
  ConfirmRestore401,
  ConfirmRestore403,
  ConfirmRestore404,
  ConfirmRestore409,
  ConfirmRestore422,
  ConfirmRestore500,
  ConfirmRestoreInputBody,
  OpenBackupRestore401,
  OpenBackupRestore403,
  OpenBackupRestore404,
  OpenBackupRestore409,
  OpenBackupRestore422,
  OpenBackupRestore500,
  OpenRestore401,
  OpenRestore403,
  OpenRestore404,
  OpenRestore409,
  OpenRestore422,
  OpenRestore500,
  OpenRestoreInputBody,
  RestoreStaged,
  RestoreSummary,
  RestoreUploaded,
  StageBackupRestore401,
  StageBackupRestore403,
  StageBackupRestore404,
  StageBackupRestore409,
  StageBackupRestore422,
  StageBackupRestore500,
  UploadBackupRestore401,
  UploadBackupRestore403,
  UploadBackupRestore409,
  UploadBackupRestore415,
  UploadBackupRestore422,
  UploadBackupRestore500,
  UploadRestore401,
  UploadRestore403,
  UploadRestore409,
  UploadRestore415,
  UploadRestore422,
  UploadRestore500,
} from "../../model";

import { apiFetch } from "../../../fetch.ts";
import type { ErrorType } from "../../../fetch.ts";

type SecondParameter<T extends (...args: never) => unknown> = Parameters<T>[1];

export const getUploadBackupRestoreUrl = () => {
  return `/api/v1/backups/restore`;
};

/**
 * @summary Upload a Backup to restore on this Controller
 */
export const uploadBackupRestore = async (
  uploadBackupRestoreBody: Blob,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RestoreUploaded> => {
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
  return apiFetch<RestoreUploaded>(getUploadBackupRestoreUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/octet-stream", ...getHeaders(options?.headers) },
    body: uploadBackupRestoreBody,
  });
};

export const getUploadBackupRestoreMutationKey = () => ["uploadBackupRestore"] as const;

export const getUploadBackupRestoreMutationOptions = <
  TError = ErrorType<
    | UploadBackupRestore401
    | UploadBackupRestore403
    | UploadBackupRestore409
    | UploadBackupRestore415
    | UploadBackupRestore422
    | UploadBackupRestore500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof uploadBackupRestore>>,
    TError,
    UploadBackupRestoreMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof uploadBackupRestore>>,
  TError,
  UploadBackupRestoreMutationVariables,
  TContext
> => {
  const mutationKey = getUploadBackupRestoreMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof uploadBackupRestore>>,
    UploadBackupRestoreMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return uploadBackupRestore(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UploadBackupRestoreMutationResult = NonNullable<
  Awaited<ReturnType<typeof uploadBackupRestore>>
>;
export type UploadBackupRestoreMutationBody = Blob;
export type UploadBackupRestoreMutationError = ErrorType<
  | UploadBackupRestore401
  | UploadBackupRestore403
  | UploadBackupRestore409
  | UploadBackupRestore415
  | UploadBackupRestore422
  | UploadBackupRestore500
>;
export type UploadBackupRestoreMutationVariables = { data: Blob };

/**
 * @summary Upload a Backup to restore on this Controller
 */
export const useUploadBackupRestore = <
  TError = ErrorType<
    | UploadBackupRestore401
    | UploadBackupRestore403
    | UploadBackupRestore409
    | UploadBackupRestore415
    | UploadBackupRestore422
    | UploadBackupRestore500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof uploadBackupRestore>>,
      TError,
      UploadBackupRestoreMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof uploadBackupRestore>>,
  TError,
  UploadBackupRestoreMutationVariables,
  TContext
> => {
  return useMutation(getUploadBackupRestoreMutationOptions(options), queryClient);
};
export const getConfirmBackupRestoreUrl = (id: string) => {
  return `/api/v1/backups/restore/${id}/confirm`;
};

/**
 * @summary Restore the staged Backup. SlipStream backs up the library first, then restarts to finish it
 */
export const confirmBackupRestore = async (
  id: string,
  confirmRestoreInputBody: ConfirmRestoreInputBody,
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
  return apiFetch<void>(getConfirmBackupRestoreUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(confirmRestoreInputBody),
  });
};

export const getConfirmBackupRestoreMutationKey = () => ["confirmBackupRestore"] as const;

export const getConfirmBackupRestoreMutationOptions = <
  TError = ErrorType<
    | ConfirmBackupRestore401
    | ConfirmBackupRestore403
    | ConfirmBackupRestore404
    | ConfirmBackupRestore409
    | ConfirmBackupRestore422
    | ConfirmBackupRestore500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof confirmBackupRestore>>,
    TError,
    ConfirmBackupRestoreMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof confirmBackupRestore>>,
  TError,
  ConfirmBackupRestoreMutationVariables,
  TContext
> => {
  const mutationKey = getConfirmBackupRestoreMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof confirmBackupRestore>>,
    ConfirmBackupRestoreMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return confirmBackupRestore(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ConfirmBackupRestoreMutationResult = NonNullable<
  Awaited<ReturnType<typeof confirmBackupRestore>>
>;
export type ConfirmBackupRestoreMutationBody = ConfirmRestoreInputBody;
export type ConfirmBackupRestoreMutationError = ErrorType<
  | ConfirmBackupRestore401
  | ConfirmBackupRestore403
  | ConfirmBackupRestore404
  | ConfirmBackupRestore409
  | ConfirmBackupRestore422
  | ConfirmBackupRestore500
>;
export type ConfirmBackupRestoreMutationVariables = { id: string; data: ConfirmRestoreInputBody };

/**
 * @summary Restore the staged Backup. SlipStream backs up the library first, then restarts to finish it
 */
export const useConfirmBackupRestore = <
  TError = ErrorType<
    | ConfirmBackupRestore401
    | ConfirmBackupRestore403
    | ConfirmBackupRestore404
    | ConfirmBackupRestore409
    | ConfirmBackupRestore422
    | ConfirmBackupRestore500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof confirmBackupRestore>>,
      TError,
      ConfirmBackupRestoreMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof confirmBackupRestore>>,
  TError,
  ConfirmBackupRestoreMutationVariables,
  TContext
> => {
  return useMutation(getConfirmBackupRestoreMutationOptions(options), queryClient);
};
export const getOpenBackupRestoreUrl = (id: string) => {
  return `/api/v1/backups/restore/${id}/open`;
};

/**
 * @summary Open the uploaded Backup with its passphrase and summarize it
 */
export const openBackupRestore = async (
  id: string,
  openRestoreInputBody: OpenRestoreInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RestoreSummary> => {
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
  return apiFetch<RestoreSummary>(getOpenBackupRestoreUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(openRestoreInputBody),
  });
};

export const getOpenBackupRestoreMutationKey = () => ["openBackupRestore"] as const;

export const getOpenBackupRestoreMutationOptions = <
  TError = ErrorType<
    | OpenBackupRestore401
    | OpenBackupRestore403
    | OpenBackupRestore404
    | OpenBackupRestore409
    | OpenBackupRestore422
    | OpenBackupRestore500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof openBackupRestore>>,
    TError,
    OpenBackupRestoreMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof openBackupRestore>>,
  TError,
  OpenBackupRestoreMutationVariables,
  TContext
> => {
  const mutationKey = getOpenBackupRestoreMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof openBackupRestore>>,
    OpenBackupRestoreMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return openBackupRestore(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type OpenBackupRestoreMutationResult = NonNullable<
  Awaited<ReturnType<typeof openBackupRestore>>
>;
export type OpenBackupRestoreMutationBody = OpenRestoreInputBody;
export type OpenBackupRestoreMutationError = ErrorType<
  | OpenBackupRestore401
  | OpenBackupRestore403
  | OpenBackupRestore404
  | OpenBackupRestore409
  | OpenBackupRestore422
  | OpenBackupRestore500
>;
export type OpenBackupRestoreMutationVariables = { id: string; data: OpenRestoreInputBody };

/**
 * @summary Open the uploaded Backup with its passphrase and summarize it
 */
export const useOpenBackupRestore = <
  TError = ErrorType<
    | OpenBackupRestore401
    | OpenBackupRestore403
    | OpenBackupRestore404
    | OpenBackupRestore409
    | OpenBackupRestore422
    | OpenBackupRestore500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof openBackupRestore>>,
      TError,
      OpenBackupRestoreMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof openBackupRestore>>,
  TError,
  OpenBackupRestoreMutationVariables,
  TContext
> => {
  return useMutation(getOpenBackupRestoreMutationOptions(options), queryClient);
};
export const getStageBackupRestoreUrl = (backupId: string) => {
  return `/api/v1/backups/${backupId}/restore`;
};

/**
 * @summary Prepare the restore of a listed Backup and summarize it
 */
export const stageBackupRestore = async (
  backupId: string,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RestoreStaged> => {
  return apiFetch<RestoreStaged>(getStageBackupRestoreUrl(backupId), {
    ...options,
    method: "POST",
  });
};

export const getStageBackupRestoreMutationKey = () => ["stageBackupRestore"] as const;

export const getStageBackupRestoreMutationOptions = <
  TError = ErrorType<
    | StageBackupRestore401
    | StageBackupRestore403
    | StageBackupRestore404
    | StageBackupRestore409
    | StageBackupRestore422
    | StageBackupRestore500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof stageBackupRestore>>,
    TError,
    StageBackupRestoreMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof stageBackupRestore>>,
  TError,
  StageBackupRestoreMutationVariables,
  TContext
> => {
  const mutationKey = getStageBackupRestoreMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof stageBackupRestore>>,
    StageBackupRestoreMutationVariables
  > = (props) => {
    const { backupId } = props ?? {};

    return stageBackupRestore(backupId, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type StageBackupRestoreMutationResult = NonNullable<
  Awaited<ReturnType<typeof stageBackupRestore>>
>;

export type StageBackupRestoreMutationError = ErrorType<
  | StageBackupRestore401
  | StageBackupRestore403
  | StageBackupRestore404
  | StageBackupRestore409
  | StageBackupRestore422
  | StageBackupRestore500
>;
export type StageBackupRestoreMutationVariables = { backupId: string };

/**
 * @summary Prepare the restore of a listed Backup and summarize it
 */
export const useStageBackupRestore = <
  TError = ErrorType<
    | StageBackupRestore401
    | StageBackupRestore403
    | StageBackupRestore404
    | StageBackupRestore409
    | StageBackupRestore422
    | StageBackupRestore500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof stageBackupRestore>>,
      TError,
      StageBackupRestoreMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof stageBackupRestore>>,
  TError,
  StageBackupRestoreMutationVariables,
  TContext
> => {
  return useMutation(getStageBackupRestoreMutationOptions(options), queryClient);
};
export const getUploadRestoreUrl = () => {
  return `/api/v1/setup/restore`;
};

/**
 * @summary Upload a Backup to restore on this Node
 */
export const uploadRestore = async (
  uploadRestoreBody: Blob,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RestoreUploaded> => {
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
  return apiFetch<RestoreUploaded>(getUploadRestoreUrl(), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/octet-stream", ...getHeaders(options?.headers) },
    body: uploadRestoreBody,
  });
};

export const getUploadRestoreMutationKey = () => ["uploadRestore"] as const;

export const getUploadRestoreMutationOptions = <
  TError = ErrorType<
    | UploadRestore401
    | UploadRestore403
    | UploadRestore409
    | UploadRestore415
    | UploadRestore422
    | UploadRestore500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof uploadRestore>>,
    TError,
    UploadRestoreMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof uploadRestore>>,
  TError,
  UploadRestoreMutationVariables,
  TContext
> => {
  const mutationKey = getUploadRestoreMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof uploadRestore>>,
    UploadRestoreMutationVariables
  > = (props) => {
    const { data } = props ?? {};

    return uploadRestore(data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type UploadRestoreMutationResult = NonNullable<Awaited<ReturnType<typeof uploadRestore>>>;
export type UploadRestoreMutationBody = Blob;
export type UploadRestoreMutationError = ErrorType<
  | UploadRestore401
  | UploadRestore403
  | UploadRestore409
  | UploadRestore415
  | UploadRestore422
  | UploadRestore500
>;
export type UploadRestoreMutationVariables = { data: Blob };

/**
 * @summary Upload a Backup to restore on this Node
 */
export const useUploadRestore = <
  TError = ErrorType<
    | UploadRestore401
    | UploadRestore403
    | UploadRestore409
    | UploadRestore415
    | UploadRestore422
    | UploadRestore500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof uploadRestore>>,
      TError,
      UploadRestoreMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof uploadRestore>>,
  TError,
  UploadRestoreMutationVariables,
  TContext
> => {
  return useMutation(getUploadRestoreMutationOptions(options), queryClient);
};
export const getConfirmRestoreUrl = (id: string) => {
  return `/api/v1/setup/restore/${id}/confirm`;
};

/**
 * @summary Restore the opened Backup. The Node restarts to finish it, then opens sign-in
 */
export const confirmRestore = async (
  id: string,
  confirmRestoreInputBody: ConfirmRestoreInputBody,
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
  return apiFetch<void>(getConfirmRestoreUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(confirmRestoreInputBody),
  });
};

export const getConfirmRestoreMutationKey = () => ["confirmRestore"] as const;

export const getConfirmRestoreMutationOptions = <
  TError = ErrorType<
    | ConfirmRestore401
    | ConfirmRestore403
    | ConfirmRestore404
    | ConfirmRestore409
    | ConfirmRestore422
    | ConfirmRestore500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof confirmRestore>>,
    TError,
    ConfirmRestoreMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof confirmRestore>>,
  TError,
  ConfirmRestoreMutationVariables,
  TContext
> => {
  const mutationKey = getConfirmRestoreMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof confirmRestore>>,
    ConfirmRestoreMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return confirmRestore(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type ConfirmRestoreMutationResult = NonNullable<Awaited<ReturnType<typeof confirmRestore>>>;
export type ConfirmRestoreMutationBody = ConfirmRestoreInputBody;
export type ConfirmRestoreMutationError = ErrorType<
  | ConfirmRestore401
  | ConfirmRestore403
  | ConfirmRestore404
  | ConfirmRestore409
  | ConfirmRestore422
  | ConfirmRestore500
>;
export type ConfirmRestoreMutationVariables = { id: string; data: ConfirmRestoreInputBody };

/**
 * @summary Restore the opened Backup. The Node restarts to finish it, then opens sign-in
 */
export const useConfirmRestore = <
  TError = ErrorType<
    | ConfirmRestore401
    | ConfirmRestore403
    | ConfirmRestore404
    | ConfirmRestore409
    | ConfirmRestore422
    | ConfirmRestore500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof confirmRestore>>,
      TError,
      ConfirmRestoreMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof confirmRestore>>,
  TError,
  ConfirmRestoreMutationVariables,
  TContext
> => {
  return useMutation(getConfirmRestoreMutationOptions(options), queryClient);
};
export const getOpenRestoreUrl = (id: string) => {
  return `/api/v1/setup/restore/${id}/open`;
};

/**
 * @summary Open the uploaded Backup with its passphrase and summarize it
 */
export const openRestore = async (
  id: string,
  openRestoreInputBody: OpenRestoreInputBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<RestoreSummary> => {
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
  return apiFetch<RestoreSummary>(getOpenRestoreUrl(id), {
    ...options,
    method: "POST",
    headers: { "Content-Type": "application/json", ...getHeaders(options?.headers) },
    body: JSON.stringify(openRestoreInputBody),
  });
};

export const getOpenRestoreMutationKey = () => ["openRestore"] as const;

export const getOpenRestoreMutationOptions = <
  TError = ErrorType<
    | OpenRestore401
    | OpenRestore403
    | OpenRestore404
    | OpenRestore409
    | OpenRestore422
    | OpenRestore500
  >,
  TContext = unknown,
>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof openRestore>>,
    TError,
    OpenRestoreMutationVariables,
    TContext
  >;
  request?: SecondParameter<typeof apiFetch>;
}): UseMutationOptions<
  Awaited<ReturnType<typeof openRestore>>,
  TError,
  OpenRestoreMutationVariables,
  TContext
> => {
  const mutationKey = getOpenRestoreMutationKey();
  const { mutation: mutationOptions, request: requestOptions } = options
    ? options.mutation && "mutationKey" in options.mutation && options.mutation.mutationKey
      ? options
      : { ...options, mutation: { ...options.mutation, mutationKey } }
    : { mutation: { mutationKey }, request: undefined };

  const mutationFn: MutationFunction<
    Awaited<ReturnType<typeof openRestore>>,
    OpenRestoreMutationVariables
  > = (props) => {
    const { id, data } = props ?? {};

    return openRestore(id, data, requestOptions);
  };

  return { mutationFn, ...mutationOptions };
};

export type OpenRestoreMutationResult = NonNullable<Awaited<ReturnType<typeof openRestore>>>;
export type OpenRestoreMutationBody = OpenRestoreInputBody;
export type OpenRestoreMutationError = ErrorType<
  | OpenRestore401
  | OpenRestore403
  | OpenRestore404
  | OpenRestore409
  | OpenRestore422
  | OpenRestore500
>;
export type OpenRestoreMutationVariables = { id: string; data: OpenRestoreInputBody };

/**
 * @summary Open the uploaded Backup with its passphrase and summarize it
 */
export const useOpenRestore = <
  TError = ErrorType<
    | OpenRestore401
    | OpenRestore403
    | OpenRestore404
    | OpenRestore409
    | OpenRestore422
    | OpenRestore500
  >,
  TContext = unknown,
>(
  options?: {
    mutation?: UseMutationOptions<
      Awaited<ReturnType<typeof openRestore>>,
      TError,
      OpenRestoreMutationVariables,
      TContext
    >;
    request?: SecondParameter<typeof apiFetch>;
  },
  queryClient?: QueryClient,
): UseMutationResult<
  Awaited<ReturnType<typeof openRestore>>,
  TError,
  OpenRestoreMutationVariables,
  TContext
> => {
  return useMutation(getOpenRestoreMutationOptions(options), queryClient);
};
