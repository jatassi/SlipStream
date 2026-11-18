import type { SubtitlesValuesBodyValues } from "./subtitlesValuesBodyValues.ts";

export interface SubtitlesValuesBody {
  /** Each setting's new value by key, all or none; null resets one to its default */
  values: SubtitlesValuesBodyValues;
}
