import type { ImportsIntoAction } from "./importsIntoAction.ts";
import type { ImportsMatchAction } from "./importsMatchAction.ts";

export type ImportsRootAction = ImportsMatchAction | ImportsIntoAction;
