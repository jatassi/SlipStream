import type { FilesRenameDraftValues } from "./filesRenameDraftValues.ts";

export interface FilesRenameDraft {
  /** The Content Type whose naming settings the draft changes */
  contentType: string;
  /** Each naming setting's unsaved value by key, over the stored ones */
  values: FilesRenameDraftValues;
}
