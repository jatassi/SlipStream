import { z } from "zod";

import { isApiError, sentenceOf } from "../api/errors";
import type { IndexersIndexer, IndexersIndexerEdit } from "../api/gen/model";
import { submitErrors } from "../ui/form";

const tested = new Set(["indexer_test_failed", "indexer_unavailable"]);

/** A failed Test's sentence, or why the Test couldn't run, shows whole above Save. */
export function refusal(error: unknown) {
  return isApiError(error) && tested.has(error.code)
    ? { form: sentenceOf(error), fields: {} }
    : submitErrors(error, { indexer_name_taken: "name" });
}

const count = z
  .string()
  .trim()
  .regex(/^\d{1,6}$/u, "Enter a whole number")
  .refine((value) => Number(value) <= 100_000, "Use 100,000 or fewer");

const per = z.enum(["hour", "day"]);

export const editSchema = z.object({
  name: z.string().trim().min(1, "Enter a name").max(100, "Use 100 characters or fewer"),
  address: z.string().trim(),
  apiKey: z.string(),
  priority: z
    .string()
    .trim()
    .regex(/^\d{1,2}$/u, "Enter a number from 1 to 50")
    .refine((value) => Number(value) >= 1 && Number(value) <= 50, "Enter a number from 1 to 50"),
  searches: count,
  searchesPer: per,
  grabs: count,
  grabsPer: per,
});

export type EditValues = z.input<typeof editSchema>;

export const valuesOf = (indexer: IndexersIndexer): EditValues => ({
  name: indexer.name,
  address: "",
  apiKey: "",
  priority: String(indexer.priority),
  searches: String(indexer.searches.count),
  searchesPer: indexer.searches.per,
  grabs: String(indexer.grabs.count),
  grabsPer: indexer.grabs.per,
});

/** Only what changed, so a PATCH tests the Indexer only when its address or key is new. */
export function editOf(indexer: IndexersIndexer, value: EditValues): IndexersIndexerEdit {
  const next = editSchema.parse(value);
  const edit: IndexersIndexerEdit = {};
  if (next.name !== indexer.name) {
    edit.name = next.name;
  }
  if (next.address !== "") {
    edit.address = next.address;
  }
  if (next.apiKey !== "") {
    edit.apiKey = next.apiKey;
  }
  if (Number(next.priority) !== indexer.priority) {
    edit.priority = Number(next.priority);
  }
  const searches = { count: Number(next.searches), per: next.searchesPer };
  if (searches.count !== indexer.searches.count || searches.per !== indexer.searches.per) {
    edit.searches = searches;
  }
  const grabs = { count: Number(next.grabs), per: next.grabsPer };
  if (grabs.count !== indexer.grabs.count || grabs.per !== indexer.grabs.per) {
    edit.grabs = grabs;
  }
  return edit;
}
