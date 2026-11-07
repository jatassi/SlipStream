export type ListLibraryParams = {
  /**
   * Only this Content Type's Works
   * @pattern ^[a-z][a-z0-9]{1,15}$
   */
  contentType?: string;
  /**
   * Only these Works, such as the ones a works.changed named; not with contentType
   * @maxItems 200
   * @nullable
   */
  workIds?: string[] | null;
};
