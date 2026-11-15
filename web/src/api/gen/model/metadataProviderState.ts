/**
 * The provider's state from its latest call or test; unused until the first one
 */
export type MetadataProviderState =
  | "unused"
  | "working"
  | "key_refused"
  | "rate_limited"
  | "out_of_allowance"
  | "unreachable";
