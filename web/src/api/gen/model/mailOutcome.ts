import type { MailOutcomeOutcome } from "./mailOutcomeOutcome.ts";

export interface MailOutcome {
  /** The mail server's reply, or why it didn't respond, when the email failed */
  detail?: string;
  outcome: MailOutcomeOutcome;
}
