import type { WorksFieldValue } from "./worksFieldValue.ts";

export interface WorksCandidate {
  /** The Work's fields */
  fields: WorksFieldValue[];
  key: string;
  label: string;
  /**
   * The Work, once it is in the library
   * @nullable
   */
  workId: string | null;
}
