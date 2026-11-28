import type { TargetsConversionSpecAudio } from "./targetsConversionSpecAudio.ts";
import type { TargetsConversionSpecCodec } from "./targetsConversionSpecCodec.ts";
import type { TargetsConversionSpecContainer } from "./targetsConversionSpecContainer.ts";
import type { TargetsConversionSpecCurve } from "./targetsConversionSpecCurve.ts";
import type { TargetsConversionSpecDepth } from "./targetsConversionSpecDepth.ts";
import type { TargetsConversionSpecMaxHeight } from "./targetsConversionSpecMaxHeight.ts";
import type { TargetsConversionSpecRange } from "./targetsConversionSpecRange.ts";

export interface TargetsConversionSpec {
  audio: TargetsConversionSpecAudio;
  codec: TargetsConversionSpecCodec;
  container: TargetsConversionSpecContainer;
  /** The tone mapping curve from HDR to SDR */
  curve: TargetsConversionSpecCurve;
  depth: TargetsConversionSpecDepth;
  dropCommentary: boolean;
  /**
   * The video bitrate at maxHeight
   * @minimum 100
   * @maximum 200000
   */
  kbps: number;
  keepDolbyVision: boolean;
  /** The largest output height. A wider picture fits the 16:9 box of that height */
  maxHeight: TargetsConversionSpecMaxHeight;
  /** source keeps the source's range, with HDR10+ written as HDR10 */
  range: TargetsConversionSpecRange;
  /**
   * The one Node its Conversions run on; null for any Node that can
   * @nullable
   */
  runOnNodeId: string | null;
}
