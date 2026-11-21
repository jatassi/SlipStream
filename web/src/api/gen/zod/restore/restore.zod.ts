import * as zod from "zod";

import {
  ConfirmRestoreInputBody,
  OpenRestoreInputBody,
  RestoreStaged,
  RestoreSummary,
  RestoreUploaded,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Upload a Backup to restore on this Controller
 */
export const UploadBackupRestoreResponse = RestoreUploaded;

/**
 * @summary Restore the staged Backup. SlipStream backs up the library first, then restarts to finish it
 */
export const confirmBackupRestorePathIdMax = 64;

export const ConfirmBackupRestoreParams = zod.object({
  id: zod.string().max(confirmBackupRestorePathIdMax),
});

export const ConfirmBackupRestoreBody = ConfirmRestoreInputBody;

export const ConfirmBackupRestoreResponse = zod.void();

/**
 * @summary Open the uploaded Backup with its passphrase and summarize it
 */
export const openBackupRestorePathIdMax = 64;

export const OpenBackupRestoreParams = zod.object({
  id: zod.string().max(openBackupRestorePathIdMax),
});

export const OpenBackupRestoreBody = OpenRestoreInputBody;

export const OpenBackupRestoreResponse = RestoreSummary;

/**
 * @summary Prepare the restore of a listed Backup and summarize it
 */
export const stageBackupRestorePathBackupIdRegExp = new RegExp("^[0-9a-f]{32}$");

export const StageBackupRestoreParams = zod.object({
  backupId: zod.string().regex(stageBackupRestorePathBackupIdRegExp),
});

export const StageBackupRestoreResponse = RestoreStaged;

/**
 * @summary Upload a Backup to restore on this Node
 */
export const UploadRestoreResponse = RestoreUploaded;

/**
 * @summary Restore the opened Backup. The Node restarts to finish it, then opens sign-in
 */
export const confirmRestorePathIdMax = 64;

export const ConfirmRestoreParams = zod.object({
  id: zod.string().max(confirmRestorePathIdMax),
});

export const ConfirmRestoreBody = ConfirmRestoreInputBody;

export const ConfirmRestoreResponse = zod.void();

/**
 * @summary Open the uploaded Backup with its passphrase and summarize it
 */
export const openRestorePathIdMax = 64;

export const OpenRestoreParams = zod.object({
  id: zod.string().max(openRestorePathIdMax),
});

export const OpenRestoreBody = OpenRestoreInputBody;

export const OpenRestoreResponse = RestoreSummary;
