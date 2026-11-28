import type { ConversionsSentenceApi } from "./conversionsSentenceApi.ts";

export interface ConversionsSentence {
  /**
   * The hardware it converts with; null with none
   * @nullable
   */
  api: ConversionsSentenceApi;
  /** Whether it can convert nothing */
  none: boolean;
  /** Whether it converts with its processor */
  processor: boolean;
}
