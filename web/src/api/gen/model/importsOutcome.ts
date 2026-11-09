import type { ImportsAddedOutcome } from "./importsAddedOutcome.ts";
import type { ImportsFailedOutcome } from "./importsFailedOutcome.ts";

export type ImportsOutcome = ImportsAddedOutcome | ImportsFailedOutcome;
