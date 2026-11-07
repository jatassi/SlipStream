import type { IndexersCatalogEntry } from "./indexersCatalogEntry.ts";
import type { IndexersDefinitionField } from "./indexersDefinitionField.ts";

export interface IndexersDefinitionForm {
  /** Signing in asks for an image captcha */
  captcha: boolean;
  /** Its Indexers sign in with a browser cookie and its User-Agent, pasted as the cookie member, in place of a cookie field */
  cookie: boolean;
  entry: IndexersCatalogEntry;
  /** In the definition's order, the captcha last */
  fields: IndexersDefinitionField[];
  /** The site's addresses, the first being the default */
  links: string[];
  /** Its Indexers sign in and keep a session on their Node */
  session: boolean;
  /** The site sits behind a browser challenge that a Challenge solver can clear */
  solver: boolean;
}
