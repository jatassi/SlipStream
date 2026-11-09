import type { FormatsDialectSchema } from "./formatsDialectSchema.ts";

export interface FormatsSchema {
  /** SlipStream's own, then Radarr's and Sonarr's */
  dialects: FormatsDialectSchema[];
}
