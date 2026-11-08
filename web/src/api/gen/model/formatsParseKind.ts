/**
 * How each name is read. release reads it as a release title. file reads it as a file path, and reads its folder's name when the file name gives no title.
 */
export type FormatsParseKind = "release" | "file";
