import * as zod from "zod";

import {
  BackupsAdded,
  BackupsDestinationList,
  BackupsNewDestination,
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
