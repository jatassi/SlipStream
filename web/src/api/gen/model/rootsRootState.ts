/**
 * Only ready takes imports. checking: its Node hasn't reported it yet; offline and written_off come from its Node's standing
 */
export type RootsRootState =
  | "ready"
  | "missing"
  | "unmounted"
  | "foreign"
  | "unwritable"
  | "unresponsive"
  | "checking"
  | "offline"
  | "written_off";
