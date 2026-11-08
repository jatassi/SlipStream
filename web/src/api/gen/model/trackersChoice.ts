export interface TrackersChoice {
  /** A Node, for its built-in client; set this or clientId */
  builtInOn?: string;
  /** An external client; set this or builtInOn */
  clientId?: string;
}
