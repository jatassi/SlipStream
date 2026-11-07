export interface WorksListing {
  /** A column of the Library's table */
  column: boolean;
  /** A filter: picked values for text and texts, a range for number, date and duration */
  filter: boolean;
  /** A sort, with sections by its value */
  sort: boolean;
}
