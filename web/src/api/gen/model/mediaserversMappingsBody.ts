import type { MediaserversMappingInput } from "./mediaserversMappingInput.ts";

export interface MediaserversMappingsBody {
  /** @maxItems 1000 */
  mappings: MediaserversMappingInput[];
}
