import type { FilesRenameScopeKind } from "./filesRenameScopeKind.ts";

export interface FilesRenameScope {
  /** The Work's id for work, the Content Type's id for type; left out for library */
  id?: string;
  kind: FilesRenameScopeKind;
}
