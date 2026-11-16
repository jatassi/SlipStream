import type { MetadataAttribution } from "./metadataAttribution.ts";
import type { MetadataDetails } from "./metadataDetails.ts";
import type { MetadataProviderKey } from "./metadataProviderKey.ts";
import type { MetadataProviderState } from "./metadataProviderState.ts";

export interface MetadataProvider {
  /** What the provider's terms require SlipStream to show; null when they require nothing */
  attribution: MetadataAttribution | null;
  details: MetadataDetails;
  /** The provider, such as tmdb */
  id: string;
  /** Which key SlipStream sends: none (this build has none), the bundled key or the Admin's own key */
  key: MetadataProviderKey;
  /** Where an Admin gets a key, opened in the browser */
  keyPage: string;
  /**
   * When the Admin set the key in use; null unless it is theirs
   * @nullable
   */
  keySetAt: string | null;
  label: string;
  /** The Admin's key in use carries a PIN */
  pinSet: boolean;
  /** The provider's state from its latest call or test; unused until the first one */
  state: MetadataProviderState;
  /** How the provider stands and what to do, in plain words: the sentence a page leads with, the rest under Details */
  summary: string;
  /** Set key offers a subscriber PIN beside the key */
  takesPin: boolean;
  /**
   * When a provider that is rate limiting SlipStream, or whose daily request limit is used up, takes calls again
   * @nullable
   */
  until: string | null;
}
