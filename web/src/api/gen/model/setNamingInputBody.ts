import type { SetNamingInputBodyValues } from "./setNamingInputBodyValues.ts";

export interface SetNamingInputBody {
  /** Each setting's new value by key, all or none; null resets one to its default */
  values: SetNamingInputBodyValues;
}
