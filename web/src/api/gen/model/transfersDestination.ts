import type { TransfersDestinationKind } from "./transfersDestinationKind.ts";

export interface TransfersDestination {
  /**
   * The Backup destination, for backup
   * @pattern ^[0-9a-f]{32}$
   */
  backupDestinationId?: string;
  /** The folder on its Node, for backup and backups_folder */
  folder?: string;
  /** root is a library root, backup a Backup destination, backups_folder the Controller's Backups folder, download_folder a Download Client's folder, and scratch a Node's scratch folder for Conversions */
  kind: TransfersDestinationKind;
  /** The Node that holds it now */
  nodeId: string;
  nodeName: string;
  /** The root, for root */
  rootId?: string;
}
