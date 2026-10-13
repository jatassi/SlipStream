import { useMutation } from "@tanstack/react-query";
import type {
  MutationFunction,
  QueryClient,
  UseMutationOptions,
  UseMutationResult,
} from "@tanstack/react-query";

import type {
  ClaimWorker401,
  ClaimWorker403,
  ClaimWorker409,
  ClaimWorker422,
  ClaimWorker500,
  ClaimWorker502,
  PairingJoinBody,
  PairingWorker,
} from "../../model";

import { apiFetch } from "../../../fetch.ts";
import type { ErrorType } from "../../../fetch.ts";

type SecondParameter<T extends (...args: never) => unknown> = Parameters<T>[1];

export const getClaimWorkerUrl = () => {
  return `/api/v1/setup/worker`;
};

/**
 * @summary Join this Node to a Controller as a Worker with a Join Code
 */
export const claimWorker = async (
  pairingJoinBody: PairingJoinBody,
  options?: Parameters<typeof apiFetch>[1],
): Promise<PairingWorker> => {
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
  return apiFetch<PairingWorker>(getClaimWorkerUrl(), {
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
 * @summary Join this Node to a Controller as a Worker with a Join Code
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
