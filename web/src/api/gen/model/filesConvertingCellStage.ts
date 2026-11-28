export type FilesConvertingCellStage =
  | "queued"
  | "copying"
  | "ready"
  | "running"
  | "paused"
  | "checking"
  | "returning"
  | "landing";
