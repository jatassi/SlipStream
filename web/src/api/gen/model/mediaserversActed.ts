import type { MailOutcome } from "./mailOutcome.ts";
import type { MediaserversAccessView } from "./mediaserversAccessView.ts";

export interface MediaserversActed {
  /** What became of the email telling the Member they can watch, when the operation sent one */
  mail?: MailOutcome;
  /** The row after the operation; absent when it deleted the row */
  row?: MediaserversAccessView;
}
