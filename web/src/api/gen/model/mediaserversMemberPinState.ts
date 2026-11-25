import type { MediaserversMemberPinStateRefusal } from "./mediaserversMemberPinStateRefusal.ts";
import type { MediaserversMemberPinStateState } from "./mediaserversMemberPinStateState.ts";

export interface MediaserversMemberPinState {
  /** Set when state is refused */
  refusal?: MediaserversMemberPinStateRefusal;
  state: MediaserversMemberPinStateState;
}
