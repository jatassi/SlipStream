import { onTestFinished } from "vitest";
import { page } from "vitest/browser";

import type { GrabsManualSearch } from "../api/gen/model";
import { closeManualSearch } from "../manual/sheet-state";
import { manualServer } from "./manual-server";

/**
 * Manual search's endpoints answering one plan. The sheet's state outlives
 * the app, so it closes once the test ends.
 */
export function manualEntry(plan: GrabsManualSearch) {
  onTestFinished(closeManualSearch);
  return manualServer({ plan: () => plan });
}

/** The open Manual search sheet, titled with its plan's scope in words. */
export const manualSheet = (plan: GrabsManualSearch) =>
  page.getByRole("dialog", { name: plan.title });
