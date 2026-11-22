/**
 * The kind of job that asked for it. backup copies a Backup to a Backup destination, or back to the Controller, and move copies a library file or a torrent's payload to another Node
 */
export type TransfersTransferPurpose = "import" | "backup" | "move";
