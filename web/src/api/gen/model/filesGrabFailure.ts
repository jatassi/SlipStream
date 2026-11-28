import type { FilesGrabFailureKind } from "./filesGrabFailureKind.ts";

export interface FilesGrabFailure {
  grabId: string;
  kind: FilesGrabFailureKind;
}
