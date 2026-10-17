import type { IdentityDeadLinkReason } from "./identityDeadLinkReason.ts";
import type { IdentityDeadLinkState } from "./identityDeadLinkState.ts";

export interface IdentityDeadLink {
  /** Who to ask: absent for unknown, which names nobody, and for a link the API key or the host command issued */
  adminName?: string;
  /** Every reason but unknown */
  expiresAt?: string;
  /** Every reason but unknown */
  issuedAt?: string;
  reason: IdentityDeadLinkReason;
  state: IdentityDeadLinkState;
}
