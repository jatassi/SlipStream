import type { FilesRenameDraft } from "./filesRenameDraft.ts";
import type { FilesRenameScope } from "./filesRenameScope.ts";

export interface PlanInputBody {
  /** Unsaved naming settings to plan under instead of the stored ones */
  draft?: FilesRenameDraft;
  scope: FilesRenameScope;
}
