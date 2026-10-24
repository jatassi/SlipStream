import type { MetadataProvider } from "./metadataProvider.ts";

export interface MetadataProviderList {
  /** In display order */
  providers: MetadataProvider[];
}
