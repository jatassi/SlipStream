export interface NamingProblem {
  /** The setting key, such as <type>.naming.file.standard */
  key: string;
  /** What is wrong, then what passes */
  sentence: string;
}
