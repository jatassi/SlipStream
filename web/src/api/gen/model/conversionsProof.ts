import type { ConversionsProofApi } from "./conversionsProofApi.ts";
import type { ConversionsProofClass } from "./conversionsProofClass.ts";
import type { ConversionsProofPath } from "./conversionsProofPath.ts";
import type { ConversionsShape } from "./conversionsShape.ts";

export interface ConversionsProof {
  /**
   * null for the processor
   * @nullable
   */
  api: ConversionsProofApi;
  class: ConversionsProofClass;
  /** Such as Intel UHD Graphics 770 */
  device: string;
  /** ffmpeg's last lines, for Details; empty unless it failed */
  error: string;
  /**
   * null unless it failed
   * @nullable
   */
  failedAt: string | null;
  /**
   * When a driver change lost it; null otherwise
   * @nullable
   */
  lostAt: string | null;
  path: ConversionsProofPath;
  /** Whether a pause stops its progress */
  pauses: boolean;
  /**
   * null unless it passed
   * @nullable
   */
  provenAt: string | null;
  shape: ConversionsShape;
  /**
   * Times real time at 1080p
   * @minimum 0
   */
  speed: number;
}
