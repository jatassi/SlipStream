/**
 * How it went into the library; null until it is placing
 * @nullable
 */
export type ImportsImportFilePlacedBy =
  | "link"
  | "clone"
  | "copy"
  | "transfer"
  | "move"
  | "in_place"
  | null;
