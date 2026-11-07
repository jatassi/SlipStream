import type { TransfersTransfer } from "../api/gen/model";
import { useListTransfersSuspense } from "../api/gen/query/transfers/transfers";
import { paceLine } from "./words";

/** The Transfer a job asked for, such as a transferring Grab's, while it has one. */
export const useTransferOf = (job: string): TransfersTransfer | undefined =>
  useListTransfersSuspense(undefined, {
    query: { select: ({ transfers }) => transfers.find((transfer) => transfer.job === job) },
  }).data;

/** A job's Transfer as a progress bar reads it: its bytes, and its pace while it copies. */
export const useTransferProgressOf = (
  job: string,
): { done: number; total: number; pace: string } | undefined => {
  const transfer = useTransferOf(job);
  return transfer === undefined
    ? undefined
    : {
        done: transfer.verified,
        total: transfer.total,
        pace: transfer.state === "copying" ? paceLine(transfer) : "",
      };
};
