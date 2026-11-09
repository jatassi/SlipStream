/**
 * The first step of the ranking where the reference Release ranked above this one
 */
export type GrabsLostAtStep =
  | "want"
  | "tier"
  | "revision"
  | "score"
  | "priority"
  | "freeleech"
  | "seeders"
  | "size"
  | "title";
