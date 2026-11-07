import type { TransfersDestinationKind } from "./transfersDestinationKind.ts";

export interface TransfersDestination {
  /** root is a library root */
  kind: TransfersDestinationKind;
  /** The Node that holds the root now */
  nodeId: string;
  nodeName: string;
  rootId: string;
}
