import type { Bare } from "./bare.ts";

/**
 * Its listed fields by key: a string for text and date, a number for number and duration in ms, a list of strings for texts; a field it lacks is left out
 */
export type WorksLibraryRowValues = { [key: string]: Bare };
