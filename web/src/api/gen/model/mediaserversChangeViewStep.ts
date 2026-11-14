export type MediaserversChangeViewStep =
  | "none"
  | "snapshotted"
  | "scanned"
  | "source_scanned"
  | "confirmed"
  | "idle"
  | "reported"
  | "settled"
  | "old_removed"
  | "new_added"
  | "new_found"
  | "restored";
