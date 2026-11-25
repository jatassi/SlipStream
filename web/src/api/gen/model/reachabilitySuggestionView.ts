import type { ReachabilitySuggestionViewKind } from "./reachabilitySuggestionViewKind.ts";

export interface ReachabilitySuggestionView {
  /** The UI and portal has no Address outsiders can use: set one first */
  addressPrivate: boolean;
  kind: ReachabilitySuggestionViewKind;
  /** The prefilled public address, empty when none is known */
  public: string;
  serverId?: string;
  subject: string;
}
