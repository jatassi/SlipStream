import type { FilesConversionFailureKind } from "./filesConversionFailureKind.ts";

export interface FilesConversionFailure {
  /** The fault, such as encoder_failed */
  code: string;
  conversionId: string;
  kind: FilesConversionFailureKind;
}
