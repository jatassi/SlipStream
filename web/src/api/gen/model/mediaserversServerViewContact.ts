/**
 * As the reaching Node last reported it; unknown until it reports and while it is offline
 */
export type MediaserversServerViewContact =
  | "unknown"
  | "responding"
  | "not_responding"
  | "sign_in_refused"
  | "other_server";
