export type TransfersWaitKind =
  | "node_away"
  | "node_other_build"
  | "connecting"
  | "no_route"
  | "root_not_ready"
  | "storage_draining"
  | "source_checking"
  | "source_not_ready";
