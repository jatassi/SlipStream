import type { ImportsHint } from "./importsHint.ts";
import type { ImportsImportEntrySource } from "./importsImportEntrySource.ts";
import type { ImportsImportEntryStage } from "./importsImportEntryStage.ts";
import type { ImportsImportFile } from "./importsImportFile.ts";
import type { ImportsRootLabel } from "./importsRootLabel.ts";
import type { ImportsSuggestion } from "./importsSuggestion.ts";

export interface ImportsImportEntry {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  createdAt: string;
  /** For a root Import in path order, else by position */
  files: ImportsImportFile[];
  /**
   * Null until every file is placed or settled
   * @nullable
   */
  finishedAt: string | null;
  /** For a root Import, its Work folder relative to the root, or the file's path when it sits loose at the root's top */
  folder: string;
  /** What the folder or file name says */
  hint: ImportsHint;
  id: string;
  /**
   * Null while it needs the Admin
   * @nullable
   */
  ignoredAt: string | null;
  /** The library root the folder is in; null outside a root Import */
  root: ImportsRootLabel | null;
  /** root: a folder a library scan found in a root and couldn't place */
  source: ImportsImportEntrySource;
  stage: ImportsImportEntryStage;
  /** The best candidate; null when none matched */
  suggestion: ImportsSuggestion | null;
}
