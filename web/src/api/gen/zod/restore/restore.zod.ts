import * as zod from "zod";

import {
  ConfirmRestoreInputBody,
  OpenRestoreInputBody,
  RestoreSummary,
  RestoreUploaded,
} from "../slipStream.schemas.zod.ts";

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
