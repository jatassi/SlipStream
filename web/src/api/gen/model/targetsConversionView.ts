import type { TargetsCause } from "./targetsCause.ts";
import type { TargetsConversionViewAudio } from "./targetsConversionViewAudio.ts";
import type { TargetsConversionViewCodec } from "./targetsConversionViewCodec.ts";
import type { TargetsConversionViewContainer } from "./targetsConversionViewContainer.ts";
import type { TargetsConversionViewCurve } from "./targetsConversionViewCurve.ts";
import type { TargetsConversionViewDepth } from "./targetsConversionViewDepth.ts";
import type { TargetsConversionViewMaxHeight } from "./targetsConversionViewMaxHeight.ts";
import type { TargetsConversionViewRange } from "./targetsConversionViewRange.ts";

export interface TargetsConversionView {
  audio: TargetsConversionViewAudio;
  /** Why no Node can convert some of its Conversions */
  causes: TargetsCause[];
  codec: TargetsConversionViewCodec;
  container: TargetsConversionViewContainer;
  /** The tone mapping curve from HDR to SDR */
  curve: TargetsConversionViewCurve;
  /**
   * The bitrate a new setting starts at
   * @minimum 100
   */
  defaultKbps: number;
  depth: TargetsConversionViewDepth;
  dropCommentary: boolean;
  /**
   * The video bitrate at maxHeight
   * @minimum 100
   * @maximum 200000
   */
  kbps: number;
  keepDolbyVision: boolean;
  /** The largest output height. A wider picture fits the 16:9 box of that height */
  maxHeight: TargetsConversionViewMaxHeight;
  /**
   * Made files the Target's older settings made
   * @minimum 0
   */
  older: number;
  /** source keeps the source's range, with HDR10+ written as HDR10 */
  range: TargetsConversionViewRange;
  /**
   * null for any Node that can
   * @nullable
   */
  runOnNodeId: string | null;
  /**
   * Rises with each change of the settings, not of Run on
   * @minimum 1
   */
  version: number;
}
