export interface IdentityPendingLink {
  email: string;
  expiresAt: string;
  id: string;
  issuedAt: string;
  /** The Admin who issued it; absent for the API key */
  issuedBy?: string;
}
