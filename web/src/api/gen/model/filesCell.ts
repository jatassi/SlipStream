import type { FilesDownloadingCell } from "./filesDownloadingCell.ts";
import type { FilesFailedCell } from "./filesFailedCell.ts";
import type { FilesMetCell } from "./filesMetCell.ts";
import type { FilesMissingCell } from "./filesMissingCell.ts";
import type { FilesUnreleasedCell } from "./filesUnreleasedCell.ts";
import type { FilesUpgradableCell } from "./filesUpgradableCell.ts";

export type FilesCell =
  | FilesUnreleasedCell
  | FilesMissingCell
  | FilesDownloadingCell
  | FilesFailedCell
  | FilesUpgradableCell
  | FilesMetCell;
