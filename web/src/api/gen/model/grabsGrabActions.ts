export interface GrabsGrabActions {
  /** A cancel can also put its Release on the Blocklist: its info hash is known */
  blocklistable: boolean;
  /** cancelGrab can cancel it */
  cancel: boolean;
  /** sendGrabTo can send it to another client */
  sendTo: boolean;
}
