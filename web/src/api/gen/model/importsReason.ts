import type { ImportsReasonCode } from "./importsReasonCode.ts";
import type { ImportsRootLabel } from "./importsRootLabel.ts";

export interface ImportsReason {
  /** Why the file is left out, waits, isn't better or failed */
  code: ImportsReasonCode;
  /** For several: the candidates that match */
  count?: number;
  /** For unreadable: the first line ffprobe printed */
  line?: string;
  /** For no_such_item: the Item the file name names, such as S03E14 */
  names?: string;
  /** For elsewhere: the root the Work lives in */
  root?: ImportsRootLabel;
  /**
   * The one sentence the row leads with; null for a root Import's reason, which its other fields word
   * @nullable
   */
  sentence: string | null;
  /** For several, no_match, no_year, elsewhere and unlisted: the name's title or the Work's label */
  title?: string;
  /** The year beside title, when it has one */
  year?: number;
}
