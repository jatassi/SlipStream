import type { IdentityPendingLink } from "./identityPendingLink.ts";
import type { MailOutcome } from "./mailOutcome.ts";

export interface IdentityIssuedSignupLink {
  /**
   * Send again only: the Media Servers the old link offered that the new one doesn't, because they are removed or no longer shown to Members
   * @nullable
   */
  dropped?: string[] | null;
  email: MailOutcome;
  link: IdentityPendingLink;
  /** The link itself, to copy */
  url: string;
}
