import type { WorksWebLinkKind } from "./worksWebLinkKind.ts";

export interface WorksWebLink {
  /** The link's key, such as imdb */
  key: string;
  kind: WorksWebLinkKind;
  /** The site's name as a person reads it */
  label: string;
  url: string;
}
