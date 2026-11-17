export type GetCalendarParams = {
  /**
   * The first day
   */
  from: string;
  /**
   * The day after the last day, at most 42 days after from
   */
  to: string;
  /**
   * Left out, every Content Type that is on
   * @pattern ^[a-z][a-z0-9]{1,15}$
   */
  contentType?: string;
  /**
   * A comma list of the dated kinds' keys GET /content-types lists; left out, every kind
   * @nullable
   */
  kinds?: string[] | null;
  /**
   * Also list unmonitored Items, each in a row of their own
   */
  unmonitored?: boolean;
};
