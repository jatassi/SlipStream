export interface IdentityRegeneratedAPIKey {
  /** @pattern ^ssk_[A-Za-z0-9_-]{43}$ */
  apiKey: string;
  createdAt: string;
}
