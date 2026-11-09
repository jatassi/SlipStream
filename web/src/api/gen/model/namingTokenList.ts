import type { NamingFieldTokens } from "./namingFieldTokens.ts";

export interface NamingTokenList {
  /** Each template setting, in the settings' order */
  fields: NamingFieldTokens[];
}
