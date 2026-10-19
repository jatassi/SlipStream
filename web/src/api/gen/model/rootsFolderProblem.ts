import type { RootsFolderProblemKind } from "./rootsFolderProblemKind.ts";

export interface RootsFolderProblem {
  /** For denied, the account SlipStream runs as on the Node */
  account?: string;
  kind: RootsFolderProblemKind;
  /** The error as the Node's OS put it */
  osText: string;
}
