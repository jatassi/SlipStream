export type ConversionsConversionRowStage =
  | "queued"
  | "copying"
  | "ready"
  | "running"
  | "paused"
  | "checking"
  | "returning"
  | "landing"
  | "done"
  | "failed"
  | "canceled";
