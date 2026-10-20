export interface WorksAvailability {
  /** The rule that chose the date, in words, such as Digital release in United States; empty when the Content Type names none */
  basis: string;
  /**
   * YYYY, YYYY-MM or YYYY-MM-DD
   * @pattern ^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$
   */
  date: string;
}
