import type { MailOutcome } from "./mailOutcome.ts";

export interface IdentityIssuedResetLink {
  email: MailOutcome;
  /** The link itself, shown only in this response */
  url: string;
}
