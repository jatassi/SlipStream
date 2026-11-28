import type { ConversionsShapeCodec } from "./conversionsShapeCodec.ts";
import type { ConversionsShapeCurve } from "./conversionsShapeCurve.ts";
import type { ConversionsShapeDepth } from "./conversionsShapeDepth.ts";
import type { ConversionsShapeRange } from "./conversionsShapeRange.ts";

export interface ConversionsShape {
  codec: ConversionsShapeCodec;
  curve: ConversionsShapeCurve;
  depth: ConversionsShapeDepth;
  keepDolbyVision: boolean;
  range: ConversionsShapeRange;
}
