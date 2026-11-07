/**
 * The monitoring preset a new Work's Items follow; left out, the type's add default. Adding a Work already in the library changes nothing
 */
export type WorksNewWorkPreset =
  | "all"
  | "future"
  | "missing"
  | "existing"
  | "first"
  | "latest"
  | "none";
