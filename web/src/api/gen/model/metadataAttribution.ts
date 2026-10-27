import type { MetadataAttributionLogo } from "./metadataAttributionLogo.ts";

export interface MetadataAttribution {
  /** The logo the web app bundles to show beside it */
  logo: MetadataAttributionLogo;
  /** The sentence to show, word for word */
  text: string;
}
