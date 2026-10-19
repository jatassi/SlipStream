/**
 * As listRoots names it
 */
export type NodesStorageRootState =
  | "ready"
  | "missing"
  | "unmounted"
  | "foreign"
  | "unwritable"
  | "unresponsive"
  | "checking"
  | "offline"
  | "written_off";
