import type { TransfersDestination } from "./transfersDestination.ts";
import type { TransfersDetails } from "./transfersDetails.ts";
import type { TransfersEnd } from "./transfersEnd.ts";
import type { TransfersFile } from "./transfersFile.ts";
import type { TransfersTransferPurpose } from "./transfersTransferPurpose.ts";
import type { TransfersTransferRoute } from "./transfersTransferRoute.ts";
import type { TransfersTransferState } from "./transfersTransferState.ts";
import type { TransfersWait } from "./transfersWait.ts";

export interface TransfersTransfer {
  destination: TransfersDestination;
  details: TransfersDetails;
  /**
   * Seconds until its files are verified; 0 while unknown
   * @minimum 0
   */
  estimateSeconds: number;
  /** In the order they copy */
  files: TransfersFile[];
  id: string;
  /** The id of the job that asked for it, such as its Grab's */
  job: string;
  /** The job's name, such as a Work's title and year */
  jobName: string;
  /** The kind of job that asked for it */
  purpose: TransfersTransferPurpose;
  /**
   * Bytes a second it copies at, else the last speed between its two Nodes; 0 while unknown
   * @minimum 0
   */
  rate: number;
  /** Why it isn't copying, as a sentence; absent while it copies */
  reason?: string;
  /** Which Node opens its lanes, or node_link when they run over the Node link; absent unless it's first in line */
  route?: TransfersTransferRoute;
  /** The Node it copies from */
  source: TransfersEnd;
  /** copying also covers files that are all verified and wait for their job to place them */
  state: TransfersTransferState;
  /**
   * Bytes
   * @minimum 0
   */
  total: number;
  /**
   * Bytes verified at the destination
   * @minimum 0
   */
  verified: number;
  /** What it waits for; absent unless its state is waiting */
  wait?: TransfersWait;
}
