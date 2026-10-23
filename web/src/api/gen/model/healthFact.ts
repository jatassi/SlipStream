export interface HealthFact {
  at?: string;
  bytes?: number;
  label: string;
  /** Exactly one of text, bytes and at is present */
  text?: string;
}
