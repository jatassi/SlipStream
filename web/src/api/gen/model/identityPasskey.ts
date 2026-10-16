export interface IdentityPasskey {
  createdAt: string;
  /** The address's host the passkey signs in at */
  host: string;
  id: string;
  lastUsedAt?: string;
  name: string;
}
