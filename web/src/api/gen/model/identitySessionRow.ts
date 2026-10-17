export interface IdentitySessionRow {
  /** The client address it signed in from */
  address: string;
  createdAt: string;
  /** The session making this request */
  current: boolean;
  /** Such as Safari on iPhone, from the browser at sign-in */
  device: string;
  id: string;
  lastSeenAt: string;
}
