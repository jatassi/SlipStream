import type { MetadataAttribution } from "./metadataAttribution.ts";
import type { MetadataDetails } from "./metadataDetails.ts";
import type { MetadataProviderKey } from "./metadataProviderKey.ts";
import type { MetadataProviderState } from "./metadataProviderState.ts";

export interface MetadataProvider {
  /** The credit the provider's terms require beside it; null when they require none */
  attribution: MetadataAttribution | null;
  details: MetadataDetails;
  /** The provider, such as tmdb */
  id: string;
  /** Whose key calls carry: none in this build, SlipStream's, or the Admin's */
  key: MetadataProviderKey;
  /** The provider's page where an Admin gets a key, for the browser */
  keyPage: string;
  /**
   * When the Admin set their key; null unless key is yours
   * @nullable
   */
  keySetAt: string | null;
  label: string;
  /** From the latest outcome of a call or test, unused until one */
  state: MetadataProviderState;
  /** How the provider stands and what to do, in plain words: the sentence a page leads with, the rest under Details */
  summary: string;
  /**
   * When a rate limited provider, or one out of its daily allowance, takes calls again
   * @nullable
   */
  until: string | null;
}
