export type ListPendingImportsParams = {
  /**
   * Only this root's pending imports and lead; every root's when left out
   */
  rootId?: string;
  /**
   * List the ignored pending imports instead of the ones that need the Admin
   */
  ignored?: boolean;
};
