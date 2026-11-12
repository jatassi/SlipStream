import type { ImportsMatchActionAction } from "./importsMatchActionAction.ts";

export interface ImportsMatchAction {
  action: ImportsMatchActionAction;
  /**
   * A record's key at the Content Type's source, as searchCatalog lists it; it becomes the suggestion
   * @minLength 1
   */
  key: string;
}
