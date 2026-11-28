export type FilesState =
  | "failed"
  | "missing"
  | "downloading"
  | "converting"
  | "upgradable"
  | "unreleased"
  | "met";
