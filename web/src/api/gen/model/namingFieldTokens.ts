import type { NamingToken } from "./namingToken.ts";

export interface NamingFieldTokens {
  /** The template setting's key */
  key: string;
  /** Every token the template may name, the Content Type's own first */
  tokens: NamingToken[];
}
