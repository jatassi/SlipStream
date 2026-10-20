import type { WorksFieldValue } from "./worksFieldValue.ts";

export interface WorksCandidate {
  /** The Work's fields */
  fields: WorksFieldValue[];
  key: string;
  label: string;
  /** The label in its original language; empty when it is the label */
  original: string;
  /** One line; empty when the source has none */
  overview: string;
  /**
   * The Work, once it is in the library
   * @nullable
   */
  workId: string | null;
}
