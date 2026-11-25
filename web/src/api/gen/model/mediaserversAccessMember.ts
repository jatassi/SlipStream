export interface MediaserversAccessMember {
  /** Absent once the Account is deleted and the row waits for its removal */
  accountId?: string;
  /** Whether the Account is turned off */
  disabled: boolean;
  /** Absent once the Account is deleted */
  email?: string;
  name: string;
}
