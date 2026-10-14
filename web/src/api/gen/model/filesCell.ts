import type { FilesDownloadingCell } from "./filesDownloadingCell.ts";
import type { FilesFailedCell } from "./filesFailedCell.ts";
import type { FilesMetCell } from "./filesMetCell.ts";
import type { FilesMissingCell } from "./filesMissingCell.ts";

export type FilesCell = FilesMissingCell | FilesDownloadingCell | FilesFailedCell | FilesMetCell;
