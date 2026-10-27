import type { FilesFailedNeed } from "./filesFailedNeed.ts";
import type { FilesNoRootNeed } from "./filesNoRootNeed.ts";
import type { FilesUpgradeNeed } from "./filesUpgradeNeed.ts";
import type { FilesWaitingNeed } from "./filesWaitingNeed.ts";

export type FilesNeed = FilesFailedNeed | FilesUpgradeNeed | FilesWaitingNeed | FilesNoRootNeed;
