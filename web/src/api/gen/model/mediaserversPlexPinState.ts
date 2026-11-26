import type { MediaserversPlexPinStateState } from "./mediaserversPlexPinStateState.ts";

export interface MediaserversPlexPinState {
  /** other_account is a sign-in as a Plex account other than the one SlipStream is signed in to, which replacePlexAccount confirms */
  state: MediaserversPlexPinStateState;
}
