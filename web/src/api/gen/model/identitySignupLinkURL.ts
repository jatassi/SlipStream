import type { IdentityPendingLink } from "./identityPendingLink.ts";

export interface IdentitySignupLinkURL {
  link: IdentityPendingLink;
  url: string;
}
