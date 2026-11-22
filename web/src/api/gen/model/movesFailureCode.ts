export type MovesFailureCode =
  | "source_gone"
  | "source_failed"
  | "damaged"
  | "source_written_off"
  | "destination_gone"
  | "destination_check_failed"
  | "switch_refused"
  | "destination_unreachable";
