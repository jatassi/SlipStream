import type { FilesRenameDraft } from "./filesRenameDraft.ts";

export interface WorkPlanInputBody {
  /** Unsaved naming settings to plan under instead of the stored ones */
  draft?: FilesRenameDraft;
}
