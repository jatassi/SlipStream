import type { ImportsReasonCode } from "./importsReasonCode.ts";
import type { ImportsRootLabel } from "./importsRootLabel.ts";

export interface ImportsReason {
  /** Why the file joins no Work */
  code: ImportsReasonCode;
  /** For several: the candidates that match */
  count?: number;
  /** For unreadable: the first line ffprobe printed */
  line?: string;
  /** For no_such_item: the Item the file name names, such as S03E14 */
  names?: string;
  /** For elsewhere: the root the Work lives in */
  root?: ImportsRootLabel;
  /** For several, no_match, no_year, elsewhere and unlisted: the name's title or the Work's label */
  title?: string;
  /** The year beside title, when it has one */
  year?: number;
}
