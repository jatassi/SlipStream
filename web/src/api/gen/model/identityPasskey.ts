export interface IdentityPasskey {
  createdAt: string;
  /** The host name the passkey is registered for */
  host: string;
  id: string;
  lastUsedAt?: string;
  name: string;
}
