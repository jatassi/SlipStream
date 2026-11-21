import * as zod from "zod";

import {
  BackupSettingsInputBody,
  BackupsAdded,
  BackupsBackupList,
  BackupsBackupSettings,
  BackupsChecked,
  BackupsDestinationList,
  BackupsNewDestination,
  BackupsPassphrase,
  CheckInputBody,
  PullInputBody,
  ReplaceInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Backup destinations and their copies
 */
export const ListBackupDestinationsResponse = BackupsDestinationList;

/**
 * @summary Make a folder on a Node a Backup destination, and copy every Backup there
 */
export const AddBackupDestinationBody = BackupsNewDestination;

export const AddBackupDestinationResponse = BackupsAdded;

/**
 * @summary Stop copying Backups to a destination, leaving its copies in its folder
 */
export const removeBackupDestinationPathDestinationIdRegExp = new RegExp("^[0-9a-f]{32}$");

export const RemoveBackupDestinationParams = zod.object({
  destinationId: zod.string().regex(removeBackupDestinationPathDestinationIdRegExp),
});

export const RemoveBackupDestinationResponse = zod.void();

/**
 * @summary Get the Backups settings
 */
export const GetBackupSettingsResponse = BackupsBackupSettings;

/**
 * @summary Change Backups settings by key, all or none
 */
export const UpdateBackupSettingsBody = BackupSettingsInputBody;

export const UpdateBackupSettingsResponse = BackupsBackupSettings;

/**
 * @summary List the Backups, newest first, with their copies and the passphrase's state
 */
export const ListBackupsResponse = BackupsBackupList;

/**
 * @summary Start a Backup now. The Backup task's run shows how it went
 */
export const BackUpNowResponse = zod.void();

/**
 * @summary Show the passphrase that opens every new Backup
 */
export const GetBackupPassphraseResponse = BackupsPassphrase;

/**
 * @summary Make a new Backup passphrase and start a Backup with it
 */
export const ReplaceBackupPassphraseBody = ReplaceInputBody;

export const ReplaceBackupPassphraseResponse = zod.void();

/**
 * @summary Check a saved copy against the current Backup passphrase
 */
export const CheckBackupPassphraseBody = CheckInputBody;

export const CheckBackupPassphraseResponse = BackupsChecked;

/**
 * @summary Record that you saved the current Backup passphrase
 */
export const SaveBackupPassphraseResponse = zod.void();

/**
 * @summary Confirm it's you before downloading the recovery sheet
 */
export const PrepareRecoverySheetResponse = zod.void();

/**
 * @summary Delete a Backup and its copies, and list the Backups left
 */
export const deleteBackupPathBackupIdRegExp = new RegExp("^[0-9a-f]{32}$");

export const DeleteBackupParams = zod.object({
  backupId: zod.string().regex(deleteBackupPathBackupIdRegExp),
});

export const DeleteBackupResponse = BackupsBackupList;

/**
 * @summary Confirm it's you before downloading a Backup's file
 */
export const prepareBackupDownloadPathBackupIdRegExp = new RegExp("^[0-9a-f]{32}$");

export const PrepareBackupDownloadParams = zod.object({
  backupId: zod.string().regex(prepareBackupDownloadPathBackupIdRegExp),
});

export const PrepareBackupDownloadResponse = zod.void();

/**
 * @summary Copy a Backup back from a destination into this machine's Backups folder
 */
export const pullBackupPathBackupIdRegExp = new RegExp("^[0-9a-f]{32}$");

export const PullBackupParams = zod.object({
  backupId: zod.string().regex(pullBackupPathBackupIdRegExp),
});

export const PullBackupBody = PullInputBody;

export const PullBackupResponse = zod.void();
