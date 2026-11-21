import type { SubtitlesAllowance } from "./subtitlesAllowance.ts";
import type { SubtitlesProviderKindsItem } from "./subtitlesProviderKindsItem.ts";
import type { SubtitlesStanding } from "./subtitlesStanding.ts";

export interface SubtitlesProvider {
  /**
   * The account the key belongs to, as the provider names it
   * @nullable
   */
  account: string | null;
  /** What the provider last reported of the key's allowance; null until it reports one */
  allowance: SubtitlesAllowance | null;
  hasKey: boolean;
  /** The provider, such as subdl */
  id: string;
  /**
   * Where an Admin gets a key; null for a provider that needs none
   * @nullable
   */
  keyPage: string | null;
  /** What it searches: films by their ids, episodes by their series and Aired numbers */
  kinds: SubtitlesProviderKindsItem[];
  name: string;
  /** The provider needs the Admin's key */
  needsKey: boolean;
  on: boolean;
  /**
   * The account's plan, as the provider names it
   * @nullable
   */
  plan: string | null;
  /**
   * Its place in the search order
   * @minimum 1
   */
  position: number;
  state: SubtitlesStanding;
}
