/**
 * The kind of job that asked for it. backup copies a Backup to a Backup destination, or back to the Controller, move copies a library file or a torrent's payload to another Node, and conversion copies a Conversion's source to the Node that converts it, or its made file back to its root
 */
export type TransfersTransferPurpose = "import" | "backup" | "move" | "conversion";
