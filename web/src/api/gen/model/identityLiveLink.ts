import type { IdentityLiveLinkState } from "./identityLiveLinkState.ts";
import type { IdentityOfferedServer } from "./identityOfferedServer.ts";

export interface IdentityLiveLink {
  /** Who issued the link; absent when the API key or the host command did */
  adminName?: string;
  /** Where the link went: the Account's email, or the new one it confirms */
  email: string;
  expiresAt: string;
  /** Signup links only: the Media Servers the link offers that are still shown to Members. Absent on Reset and email links */
  mediaServers?: IdentityOfferedServer[];
  serverName: string;
  state: IdentityLiveLinkState;
}
