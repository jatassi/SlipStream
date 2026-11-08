import type { ImportsHint } from "./importsHint.ts";
import type { ImportsPendingFile } from "./importsPendingFile.ts";
import type { ImportsSuggestion } from "./importsSuggestion.ts";

export interface ImportsEntry {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** In path order */
  files: ImportsPendingFile[];
  /** Its Work folder relative to the root, or the file's path when it sits loose at the root's top */
  folder: string;
  foundAt: string;
  /** What the folder or file name says */
  hint: ImportsHint;
  id: string;
  /**
   * Null while it needs the Admin
   * @nullable
   */
  ignoredAt: string | null;
  rootId: string;
  /** The best candidate; null when none matched */
  suggestion: ImportsSuggestion | null;
}
