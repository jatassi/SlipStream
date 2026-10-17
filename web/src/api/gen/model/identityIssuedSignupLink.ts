import type { IdentityPendingLink } from "./identityPendingLink.ts";
import type { MailOutcome } from "./mailOutcome.ts";

export interface IdentityIssuedSignupLink {
  email: MailOutcome;
  link: IdentityPendingLink;
  /** The link itself, to copy */
  url: string;
}
