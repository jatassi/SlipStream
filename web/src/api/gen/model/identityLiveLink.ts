import type { IdentityLiveLinkState } from "./identityLiveLinkState.ts";

export interface IdentityLiveLink {
  /** Who issued the link; absent when the API key or the host command did */
  adminName?: string;
  /** Where the link went: the Account's email, or the new one it confirms */
  email: string;
  expiresAt: string;
  serverName: string;
  state: IdentityLiveLinkState;
}
