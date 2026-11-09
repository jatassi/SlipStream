export interface FilesConfirmedWork {
  /**
   * The digest the Work's plan showed
   * @pattern ^[0-9a-f]{16}$
   */
  digest: string;
  /**
   * Files to leave as they are; a file that trades or chains paths with another leaves both
   * @nullable
   */
  leave?: string[] | null;
  work: string;
}
