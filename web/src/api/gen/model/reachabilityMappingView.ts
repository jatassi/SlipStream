export interface ReachabilityMappingView {
  /** @nullable */
  expires: string | null;
  external: number;
  internal: number;
  /** The router took the mapping only with no expiry */
  permanent: boolean;
}
