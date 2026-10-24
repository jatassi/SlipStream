/**
 * From the latest outcome of a call or test, unused until one
 */
export type MetadataProviderState =
  | "unused"
  | "working"
  | "key_refused"
  | "rate_limited"
  | "out_of_allowance"
  | "unreachable";
