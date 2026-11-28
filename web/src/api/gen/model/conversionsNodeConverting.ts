import type { ConversionsHours } from "./conversionsHours.ts";
import type { ConversionsProof } from "./conversionsProof.ts";
import type { ConversionsSentence } from "./conversionsSentence.ts";
import type { ConversionsVolume } from "./conversionsVolume.ts";

export interface ConversionsNodeConverting {
  /** @minimum 1 */
  atOnce: number;
  /** @minimum 1 */
  atOnceDefault: number;
  hours: ConversionsHours;
  /** Whether a Target's Run on names this Node while it can't run that Target's Conversions */
  override: boolean;
  /** Use the processor when no GPU can */
  processor: boolean;
  proofs: ConversionsProof[];
  /** @minimum 0 */
  running: number;
  /** The scratch folder's volume; null while unknown */
  scratch: ConversionsVolume | null;
  sentence: ConversionsSentence;
}
