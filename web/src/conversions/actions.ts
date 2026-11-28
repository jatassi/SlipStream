import { sentenceOf } from "../api/errors";
import type { ConversionsConversionRow } from "../api/gen/model";
import {
  cancelConversion,
  cancelConversionGroup,
  moveConversion,
  moveConversionUp,
  retryConversion,
} from "../api/gen/query/conversions/conversions";
import { toast } from "../ui/toast";

// Success shows nothing, since conversions.changed redraws the row in its new
// place. A refusal is a toast in the problem's words, or in a dialog the
// dialog's own failure line.

async function act(failed: string, run: () => Promise<unknown>): Promise<void> {
  try {
    await run();
  } catch (error) {
    toast(`${failed} ${sentenceOf(error)}`, { kind: "fail" });
  }
}

async function confirmed(run: () => Promise<unknown>): Promise<void> {
  try {
    await run();
  } catch (error) {
    throw new Error(sentenceOf(error), { cause: error });
  }
}

export const moveUp = (row: ConversionsConversionRow, subject: string) =>
  act(`${subject} wasn't moved up.`, () => moveConversionUp(row.id));

export const moveAfter = (dragged: string, after: string) =>
  act("The Conversion wasn't moved.", () => moveConversion(dragged, { after }));

export const retry = (row: ConversionsConversionRow, subject: string) =>
  act(`The Conversion of ${subject} wasn't retried.`, () => retryConversion(row.id));

export const cancel = (id: string) => confirmed(() => cancelConversion(id));

export const cancelGroup = (id: string) => confirmed(() => cancelConversionGroup(id));
