/**
 * The settings that aren't secret, by name. Secret settings, such as passwords, are never returned
 */
export type IndexersPinValues = { [key: string]: string };
