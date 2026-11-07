import { z } from "zod";

import { fieldErrors, isApiError, sentenceOf } from "../api/errors";
import type {
  IndexersDefinitionField,
  IndexersIndexer,
  IndexersIndexerEdit,
} from "../api/gen/model";
import { submitErrors } from "../ui/form";
import type { SettingsForm } from "./definition-form";
import { addressOf, savedAddress } from "./fields/address-field";
import { holdsValue, settingAt, startValue } from "./fields/settings";

const tested = new Set(["indexer_test_failed", "indexer_unavailable"]);

/**
 * A failed Test's sentence, or why the Test couldn't run, shows whole above
 * Save; a refused setting or address shows beside its field.
 */
export function refusal(error: unknown, fields: readonly IndexersDefinitionField[] = []) {
  if (isApiError(error) && tested.has(error.code)) {
    return { form: sentenceOf(error), fields: {} };
  }
  if (isApiError(error) && error.code === "invalid") {
    return {
      fields: Object.fromEntries(
        Object.entries(fieldErrors(error)).map(([at, message]) => [
          at === "baseAddress" ? "baseAddress.typed" : settingAt(fields, at),
          message,
        ]),
      ),
    };
  }
  return submitErrors(error, { indexer_name_taken: "name" });
}

const count = z
  .string()
  .trim()
  .regex(/^\d{1,6}$/u, "Enter a whole number")
  .refine((value) => Number(value) <= 100_000, "Use 100,000 or fewer");

export const limitSchema = z.object({ count, per: z.enum(["hour", "day"]) });

export const nameSchema = z
  .string()
  .trim()
  .min(1, "Enter a name")
  .max(100, "Use 100 characters or fewer");

export const prioritySchema = z
  .string()
  .trim()
  .regex(/^\d{1,2}$/u, "Enter a number from 1 to 50")
  .refine((value) => Number(value) >= 1 && Number(value) <= 50, "Enter a number from 1 to 50");

/** A limit as the API takes it, from the form's count. */
export const limitOf = (limit: z.output<typeof limitSchema>) => ({
  count: Number(limit.count),
  per: limit.per,
});

export const editSchema = z.object({
  name: nameSchema,
  address: z.string().trim(),
  apiKey: z.string(),
  baseAddress: z.object({ choice: z.string(), typed: z.string() }),
  values: z.array(z.string()),
  code: z.string(),
  priority: prioritySchema,
  searches: limitSchema,
  grabs: limitSchema,
});

export type EditValues = z.input<typeof editSchema>;

/** The pin a catalog or native Indexer's settings and address are saved on. */
const pinOf = (indexer: IndexersIndexer) => indexer.definition ?? indexer.port;

/**
 * The settings as saved: each plain one's value, or its default when it
 * was left out, and each sealed one empty, since it is never returned.
 */
const savedValues = (indexer: IndexersIndexer, settings: SettingsForm | undefined) =>
  settings?.form.fields.map((field) =>
    field.sealed ? "" : (pinOf(indexer)?.values[field.name] ?? startValue(field)),
  ) ?? [];

export const valuesOf = (indexer: IndexersIndexer, settings?: SettingsForm): EditValues => ({
  name: indexer.name,
  address: "",
  apiKey: "",
  baseAddress: savedAddress(pinOf(indexer)?.baseAddress ?? "", settings?.form.links ?? []),
  values: savedValues(indexer, settings),
  code: "",
  priority: String(indexer.priority),
  searches: { count: String(indexer.searches.count), per: indexer.searches.per },
  grabs: { count: String(indexer.grabs.count), per: indexer.grabs.per },
});

/** The settings that changed: a plain one whose value differs, a sealed one replaced. */
function changedSettings(
  indexer: IndexersIndexer,
  settings: SettingsForm | undefined,
  values: readonly string[],
) {
  const saved = savedValues(indexer, settings);
  return Object.fromEntries(
    (settings?.form.fields ?? []).flatMap((field, index) => {
      const value = values[index] ?? "";
      const changed = field.sealed ? value !== "" : value !== saved[index];
      return holdsValue(field) && changed ? [[field.name, value]] : [];
    }),
  );
}

/**
 * Only what changed, so a PATCH tests the Indexer only when something its
 * Test reads is new, with the two-factor code when the code step shows.
 */
export function editOf(
  indexer: IndexersIndexer,
  value: EditValues,
  settings?: SettingsForm,
  codeShown = false,
): IndexersIndexerEdit {
  const next = editSchema.parse(value);
  const edit: IndexersIndexerEdit = {};
  const values = changedSettings(indexer, settings, next.values);
  if (Object.keys(values).length > 0) {
    edit.values = values;
  }
  const address = addressOf(next.baseAddress);
  if (settings !== undefined && address !== pinOf(indexer)?.baseAddress) {
    edit.baseAddress = address;
  }
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
  const searches = limitOf(next.searches);
  if (searches.count !== indexer.searches.count || searches.per !== indexer.searches.per) {
    edit.searches = searches;
  }
  const grabs = limitOf(next.grabs);
  if (grabs.count !== indexer.grabs.count || grabs.per !== indexer.grabs.per) {
    edit.grabs = grabs;
  }
  if (codeShown && Object.keys(edit).length > 0) {
    edit.code = next.code.trim();
  }
  return edit;
}
