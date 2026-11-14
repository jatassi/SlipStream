import type { MediaserversKind } from "./mediaserversKind.ts";
import type { MediaserversKindViewSignIn } from "./mediaserversKindViewSignIn.ts";

export interface MediaserversKindView {
  /** The kind takes a browser address, which Open in uses */
  browser: boolean;
  /** The oldest version SlipStream supports, such as 12.2 */
  floor: string;
  kind: MediaserversKind;
  /** The product's name, such as Jellyfin */
  label: string;
  /** plex_pin signs in with a Plex account; api_key_or_admin takes an API key or an administrator's name and password */
  signIn: MediaserversKindViewSignIn;
}
