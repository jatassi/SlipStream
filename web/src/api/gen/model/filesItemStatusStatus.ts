/**
 * The first state its cells hold, in the glossary's order; null for an Item with no cell
 * @nullable
 */
export type FilesItemStatusStatus =
  | "unreleased"
  | "missing"
  | "downloading"
  | "failed"
  | "upgradable"
  | "met"
  | null;
