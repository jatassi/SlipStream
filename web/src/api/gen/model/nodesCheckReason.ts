export type NodesCheckReason =
  | "role_taken"
  | "probe_fails"
  | "probe_weak"
  | "offline_unchecked"
  | "reach_failed"
  | "reached"
  | "nothing_to_test"
  | "node_is_controller"
  | "node_holds_work"
  | "nothing_held"
  | "keeps_role";
