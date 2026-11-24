import type { MovesBatchRowCounts } from "./movesBatchRowCounts.ts";
import type { MovesBatchRowOrigin } from "./movesBatchRowOrigin.ts";

export interface MovesBatchRow {
  /** A unit hasn't switched yet */
  canCancel: boolean;
  /** The batch finished, and a unit it moved hasn't moved again since */
  canMoveBack: boolean;
  /** A unit failed */
  canRetry: boolean;
  canceledAt?: string;
  /** Units per phase */
  counts: MovesBatchRowCounts;
  createdAt: string;
  /** When its last unit ended; absent while one runs */
  finishedAt?: string;
  id: string;
  origin: MovesBatchRowOrigin;
  /** @minimum 0 */
  total: number;
  /** The batch in a sentence, such as 12 of 43 movies switched */
  words: string;
}
