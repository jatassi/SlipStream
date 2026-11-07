import type { TransfersSpeedRoute } from "./transfersSpeedRoute.ts";

export interface TransfersSpeed {
  destinationNodeId: string;
  /**
   * Bytes a second, the speed of the last Transfer from the source to the destination
   * @minimum 0
   */
  rate: number;
  rateAt: string;
  /** The last route that moved bytes; absent while none has */
  route?: TransfersSpeedRoute;
  routeAt?: string;
  sourceNodeId: string;
}
