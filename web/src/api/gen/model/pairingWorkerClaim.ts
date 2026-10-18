import type { PairingController } from "./pairingController.ts";
import type { PairingProblem } from "./pairingProblem.ts";
import type { PairingWorkerClaimState } from "./pairingWorkerClaimState.ts";

export interface PairingWorkerClaim {
  /** Set once joined */
  controller?: PairingController;
  /** The SLN1 text to paste into the Controller's Add a Node dialog; absent once joined without a wait */
  nodeAddress?: string;
  /** This Node's id; set once joined */
  nodeId?: string;
  /** Why a failed Join failed */
  problem?: PairingProblem;
  state: PairingWorkerClaimState;
  /** When this Node stops waiting for the dial */
  until?: string;
}
