import type { MetadataDetails } from "./metadataDetails.ts";
import type { MetadataProviderKey } from "./metadataProviderKey.ts";
import type { MetadataProviderState } from "./metadataProviderState.ts";

export interface MetadataProvider {
  details: MetadataDetails;
  /** The provider, such as tmdb */
  id: string;
  /** Whose key calls carry: none in this build, SlipStream's, or the Admin's */
  key: MetadataProviderKey;
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
