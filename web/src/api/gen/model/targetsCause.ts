import type { TargetsCauseClass } from "./targetsCauseClass.ts";
import type { TargetsCauseWhy } from "./targetsCauseWhy.ts";
import type { TargetsNamed } from "./targetsNamed.ts";

export interface TargetsCause {
  /** The source class it covers */
  class: TargetsCauseClass;
  /**
   * The Conversions waiting on it
   * @minimum 1
   */
  count: number;
  /** The Node its sentence names; null when none */
  node: TargetsNamed | null;
  /** When the Admin was notified */
  notifiedAt: string;
  /** The settings shape it covers, such as h264.8.sdr.bt2390 */
  shape: string;
  why: TargetsCauseWhy;
}
