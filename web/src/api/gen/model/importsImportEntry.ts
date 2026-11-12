import type { ImportsHint } from "./importsHint.ts";
import type { ImportsImportEntrySource } from "./importsImportEntrySource.ts";
import type { ImportsImportEntryStage } from "./importsImportEntryStage.ts";
import type { ImportsImportFailure } from "./importsImportFailure.ts";
import type { ImportsImportFile } from "./importsImportFile.ts";
import type { ImportsImportGrab } from "./importsImportGrab.ts";
import type { ImportsRootLabel } from "./importsRootLabel.ts";
import type { ImportsSuggestion } from "./importsSuggestion.ts";

export interface ImportsImportEntry {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  createdAt: string;
  /**
   * The key of the cut its files fill; null for any under any cut
   * @nullable
   */
  edition: string | null;
  /** Why the whole Import failed; null unless it did */
  failure: ImportsImportFailure | null;
  /** For a root Import in path order, else by position */
  files: ImportsImportFile[];
  /**
   * Null until every file is placed or settled
   * @nullable
   */
  finishedAt: string | null;
  /** For a root Import, its Work folder relative to the root, or the file's path when it sits loose at the root's top. The folder on its Node for a folder Import, and empty for a download, whose file paths are relative to its torrent */
  folder: string;
  /** The Grab whose download this is; null for a folder or a root */
  grab: ImportsImportGrab | null;
  /** What the folder or file name says */
  hint: ImportsHint;
  id: string;
  /**
   * Null while it needs the Admin
   * @nullable
   */
  ignoredAt: string | null;
  /** A folder Import moves its files into the library and deletes the folder once it is empty */
  moveFiles: boolean;
  /**
   * The Node the files are on, for a download or a folder; null for a root
   * @nullable
   */
  nodeId: string | null;
  /** The library root the folder is in; null outside a root Import */
  root: ImportsRootLabel | null;
  /** The download keeps seeding while a file waits */
  seeding: boolean;
  /** root: a folder a library scan found in a root and couldn't place */
  source: ImportsImportEntrySource;
  stage: ImportsImportEntryStage;
  /** The best candidate; null when none matched */
  suggestion: ImportsSuggestion | null;
  /**
   * The Work its files go into; null for a root Import, whose suggestion names one
   * @nullable
   */
  workId: string | null;
}
