import type { ConversionsNamed } from "./conversionsNamed.ts";
import type { ConversionsWaitCause } from "./conversionsWaitCause.ts";
import type { ConversionsWaitCode } from "./conversionsWaitCode.ts";

export interface ConversionsWait {
  /** For no_node: why the Node closest to running it can't */
  cause?: ConversionsWaitCause;
  /** Why it hasn't started */
  code: ConversionsWaitCode;
  /**
   * For room: the bytes free on the Node
   * @minimum 0
   */
  free?: number;
  /**
   * For room: the bytes it needs
   * @minimum 0
   */
  need?: number;
  /** The Node its words name, when one */
  node?: ConversionsNamed;
  /**
   * For in_line: its place in line, from 1
   * @minimum 1
   */
  place?: number;
  /**
   * For transcode: the Media Servers transcoding on its Node
   * @nullable
   */
  servers?: string[] | null;
  /** For download: the Target it is made from */
  sourceTarget?: string;
  /** For hours and download: when the wait ends */
  until?: string;
}
