import type { ConversionsFailureClass } from "./conversionsFailureClass.ts";

export interface ConversionsFailure {
  /** run: it failed on every Node it may try; source: its source file can't be converted as its settings ask */
  class: ConversionsFailureClass;
  /** The fault, such as encoder_failed */
  code: string;
  /** ffmpeg's last lines, for Details */
  detail: string;
  /** The Nodes its counted attempts failed on */
  nodes: string[];
}
