/**
 * How the Target gets its file: download, made from another Target's file (derive), or downloaded, or made from another Target's file after a wait (download_then_derive)
 */
export type TargetsEntrySourcing = "download" | "derive" | "download_then_derive";
