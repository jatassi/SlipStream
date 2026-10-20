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
   * The artwork route for the result's poster, without a size; null when the source has none
   * @nullable
   */
  posterUrl: string | null;
  /**
   * The Work, once it is in the library
   * @nullable
   */
  workId: string | null;
}
