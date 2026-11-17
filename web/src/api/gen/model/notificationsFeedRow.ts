export interface NotificationsFeedRow {
  at: string;
  event: string;
  /** The message's first line; empty for none */
  line: string;
  /**
   * Where the row leads; null while no Address is set
   * @nullable
   */
  link: string | null;
  seq: number;
  title: string;
  /**
   * Null for a request made with the API key
   * @nullable
   */
  unread: boolean | null;
}
