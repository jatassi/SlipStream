export type MovesMoveUnitPhase =
  | "queued"
  | "transfer"
  | "place"
  | "switch"
  | "cleanup"
  | "done"
  | "failed"
  | "canceled";
