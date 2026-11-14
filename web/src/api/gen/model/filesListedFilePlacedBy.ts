/**
 * How its import put it into the library; null for a file no import recorded
 * @nullable
 */
export type FilesListedFilePlacedBy =
  | "link"
  | "clone"
  | "copy"
  | "transfer"
  | "move"
  | "in_place"
  | null;
