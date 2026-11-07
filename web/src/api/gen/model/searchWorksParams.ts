export type SearchWorksParams = {
  /**
   * Words each match's titles hold as word prefixes, case and accents ignored
   * @minLength 1
   * @maxLength 200
   */
  q: string;
  /**
   * How many rows to return; 0 returns ids only
   * @minimum 0
   * @maximum 20000
   */
  limit?: number;
  /**
   * Only this Content Type's Works
   * @pattern ^[a-z][a-z0-9]{1,15}$
   */
  contentType?: string;
};
