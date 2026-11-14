/**
 * The first step of the ranking where the reference Release ranked above this one
 */
export type GrabsLostAtStep =
  | "want"
  | "fewer_releases"
  | "tier"
  | "revision"
  | "score"
  | "chosen_by_another"
  | "fewer_unwanted"
  | "priority"
  | "freeleech"
  | "seeders"
  | "size"
  | "title";
