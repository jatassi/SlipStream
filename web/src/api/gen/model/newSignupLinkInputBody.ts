import type { IdentityServerGrant } from "./identityServerGrant.ts";

export interface NewSignupLinkInputBody {
  /** @maxLength 254 */
  email: string;
  /**
   * The Media Servers to offer the new Member, each shown to Members
   * @maxItems 100
   * @nullable
   */
  mediaServers?: IdentityServerGrant[] | null;
}
