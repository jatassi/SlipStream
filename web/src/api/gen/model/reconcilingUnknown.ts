export interface ReconcilingUnknown {
  /** The address it connected from */
  address: string;
  arch: string;
  /** When it last tried to connect */
  at: string;
  /** Its node key's fingerprint */
  fingerprint: string;
  os: string;
}
