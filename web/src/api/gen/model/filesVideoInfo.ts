import type { FilesDolbyVision } from "./filesDolbyVision.ts";
import type { FilesHdrFormat } from "./filesHdrFormat.ts";

export interface FilesVideoInfo {
  /**
   * 0 when the stream states none
   * @minimum 0
   */
  bitDepth: number;
  /** ffprobe's codec name, such as hevc */
  codec: string;
  /** Null when it carries no Dolby Vision */
  dolbyVision: FilesDolbyVision | null;
  /**
   * Frames per second to three decimals; 0 when the stream states none
   * @minimum 0
   */
  frameRate: number;
  /** Every HDR format it carries, or SDR alone */
  hdr: FilesHdrFormat[];
  /** @minimum 0 */
  height: number;
  /** Such as Main 10; empty when ffprobe names none */
  profile: string;
  /** @minimum 0 */
  width: number;
}
