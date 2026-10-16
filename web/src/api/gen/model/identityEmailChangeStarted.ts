import type { MailOutcome } from "./mailOutcome.ts";

export interface IdentityEmailChangeStarted {
  /** What became of the confirm link's email to the new address */
  email: MailOutcome;
}
