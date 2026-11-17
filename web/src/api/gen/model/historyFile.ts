export interface HistoryFile {
  /** Such as Episode 1 */
  items: string[];
  /** How the file was placed: a placed-by value of GET /works/{workId}/files */
  placedBy: string;
  quality: string;
  /** The qualities it replaced, in words; empty for none */
  replaced: string;
  /** The label of the file it replaced; empty for none */
  replacedLabel: string;
}
