import type { TransfersTransfer } from "../api/gen/model";
import { useListTransfersSuspense } from "../api/gen/query/transfers/transfers";

/** The Transfer a job asked for, such as a transferring Grab's, while it has one. */
export const useTransferOf = (job: string): TransfersTransfer | undefined =>
  useListTransfersSuspense(undefined, {
    query: { select: ({ transfers }) => transfers.find((transfer) => transfer.job === job) },
  }).data;
