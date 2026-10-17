import type { IdentityPinStateState } from "./identityPinStateState.ts";

export interface IdentityPinState {
  state: IdentityPinStateState;
  /** When the wait ends, while it waits */
  until?: string;
}
