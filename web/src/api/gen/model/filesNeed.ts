import type { FilesConversionFailedNeed } from "./filesConversionFailedNeed.ts";
import type { FilesFailedNeed } from "./filesFailedNeed.ts";
import type { FilesNoRootNeed } from "./filesNoRootNeed.ts";
import type { FilesNotFoundNeed } from "./filesNotFoundNeed.ts";
import type { FilesUpgradeNeed } from "./filesUpgradeNeed.ts";
import type { FilesWaitingNeed } from "./filesWaitingNeed.ts";

export type FilesNeed =
  | FilesFailedNeed
  | FilesConversionFailedNeed
  | FilesUpgradeNeed
  | FilesNotFoundNeed
  | FilesWaitingNeed
  | FilesNoRootNeed;
