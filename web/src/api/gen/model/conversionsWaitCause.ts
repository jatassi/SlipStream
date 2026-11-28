/**
 * For no_node: why the Node closest to running it can't
 */
export type ConversionsWaitCause =
  | "no_role"
  | "lapsed"
  | "runs_elsewhere"
  | "pinned_elsewhere"
  | "failed_here"
  | "untested"
  | "test_failed"
  | "retesting"
  | "proof_lost"
  | "processor_off"
  | "none_online"
  | "no_holder";
