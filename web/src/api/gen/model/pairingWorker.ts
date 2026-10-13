import type { PairingController } from "./pairingController.ts";

export interface PairingWorker {
  controller: PairingController;
  nodeId: string;
}
