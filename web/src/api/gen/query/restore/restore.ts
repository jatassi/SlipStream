import { useMutation } from "@tanstack/react-query";
import type {
  MutationFunction,
  QueryClient,
  UseMutationOptions,
  UseMutationResult,
} from "@tanstack/react-query";

import type {
  ConfirmRestore401,
  ConfirmRestore403,
  ConfirmRestore404,
  ConfirmRestore409,
  ConfirmRestore422,
  ConfirmRestore500,
  ConfirmRestoreInputBody,
  OpenRestore401,
  OpenRestore403,
  OpenRestore404,
  OpenRestore409,
  OpenRestore422,
  OpenRestore500,
  OpenRestoreInputBody,
  RestoreSummary,
  RestoreUploaded,
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
