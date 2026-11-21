import type { SupersededReplacement } from "./supersededReplacement.ts";
import type { SupersededViewState } from "./supersededViewState.ts";

export interface SupersededView {
  at?: string;
  replacement?: SupersededReplacement;
  state: SupersededViewState;
  worker?: string;
  workerId?: string;
}
