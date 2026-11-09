import type { NamingTokenGroup } from "./namingTokenGroup.ts";
import type { NamingTokenKind } from "./namingTokenKind.ts";
import type { NamingTokenModifier } from "./namingTokenModifier.ts";
import type { NamingTokenScope } from "./namingTokenScope.ts";

export interface NamingToken {
  /** What the token prints, as a sentence */
  about: string;
  /** Other names that match the same token */
  aliases: string[];
  /** A value the token prints */
  example: string;
  /** The token builder's heading */
  group: NamingTokenGroup;
  kind: NamingTokenKind;
  /**
   * What it takes after a colon; null when it takes nothing
   * @nullable
   */
  modifier: NamingTokenModifier;
  name: string;
  /** The tier the value comes from */
  scope: NamingTokenScope;
  /** The words of the tags that may wrap it, as in {imdb-{ImdbId}} */
  tags: string[];
}
