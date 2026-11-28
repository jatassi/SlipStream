import type { ConversionsConversionGroupGroupKind } from "./conversionsConversionGroupGroupKind.ts";
import type { ConversionsConversionGroupKind } from "./conversionsConversionGroupKind.ts";

export interface ConversionsConversionGroup {
  /**
   * Their forecast output size
   * @minimum 0
   */
  bytes: number;
  /** @minimum 1 */
  count: number;
  /**
   * null with no forecast
   * @nullable
   */
  finish: string | null;
  groupKind: ConversionsConversionGroupGroupKind;
  id: string;
  kind: ConversionsConversionGroupKind;
  /** Such as Backfill of 720p mobile, or The Office season 2 */
  label: string;
  /** The Nodes they are forecast to run on */
  nodes: string[];
  /**
   * Their forecast run time
   * @minimum 0
   */
  seconds: number;
  /**
   * null with no forecast
   * @nullable
   */
  start: string | null;
}
