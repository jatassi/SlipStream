import type { FilesNoRootNeedKind } from "./filesNoRootNeedKind.ts";

export interface FilesNoRootNeed {
  /** The Work has no root. Its downloads wait until it has one */
  kind: FilesNoRootNeedKind;
}
