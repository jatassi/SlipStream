import { z } from "zod";

import { type ApiError, isProblem, sentenceOf } from "../api/errors";
import type {
  NotificationsConnection as Connection,
  NotificationsDraftBody as Draft,
  NotificationsEditBody as Edit,
  NotificationsEvent as Event,
  NotificationsField as Field,
  NotificationsProvider as Provider,
} from "../api/gen/model";
import type { TestOutcome } from "../settings/test-line";
import { type SubmitErrors, submitErrors } from "../ui/form-context";

/**
 * The sheet's draft. Each setting is the text its control edits, or a
 * switch's boolean. A secret stays "" unless a new one is typed, so an empty
 * one keeps the saved value, and null removes an optional one.
 */
export type ConnectionValues = {
  on: boolean;
  name: string;
  events: string[];
  warnings: boolean;
  posters: boolean;
  node: string;
  settings: Record<string, string | boolean>;
  secrets: Record<string, string | null>;
};

const settingsOf = (provider: Provider): Field[] =>
  provider.fields.filter((field) => !field.secret && field.kind !== "node");

const secretsOf = (provider: Provider): Field[] => provider.fields.filter((field) => field.secret);

const lines = (text: string): string[] =>
  text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "");

/** A saved value as its control edits it: lists one entry per line. */
function controlOf(field: Field, value: unknown): string | boolean {
  if (field.kind === "switch") {
    return value === true;
  }
  if (value === undefined || value === null) {
    return "";
  }
  if (Array.isArray(value)) {
    return value.map(String).join("\n");
  }
  return typeof value === "string" ? value : JSON.stringify(value);
}

/** A control's value as the API takes it; null clears an optional number. */
function wireOf(field: Field, value: string | boolean): unknown {
  if (typeof value === "boolean") {
    return value;
  }
  if (field.kind === "number" || field.kind === "duration") {
    return value.trim() === "" ? null : Number(value);
  }
  return field.kind === "list" ? lines(value) : value.trim();
}

const headerOf = (line: string) => {
  const at = line.indexOf(":");
  return at === -1
    ? { name: line, value: "" }
    : { name: line.slice(0, at).trim(), value: line.slice(at + 1).trim() };
};

/** A typed secret as the API takes it. A list or headers secret travels as its JSON. */
function secretWire(field: Field, value: string): string {
  if (field.kind === "list") {
    return JSON.stringify(lines(value));
  }
  return field.kind === "headers"
    ? JSON.stringify(lines(value).map((line) => headerOf(line)))
    : value;
}

export const recommended = (events: readonly Event[]): string[] =>
  events.filter((event) => event.onByDefault && !event.fixed).map((event) => event.id);

export function newValues(provider: Provider, events: readonly Event[]): ConnectionValues {
  return {
    on: true,
    name: provider.label,
    events: recommended(events),
    warnings: false,
    posters: true,
    node: "",
    settings: Object.fromEntries(
      settingsOf(provider).map((field) => [field.key, controlOf(field, field.default)]),
    ),
    secrets: Object.fromEntries(secretsOf(provider).map((field) => [field.key, ""])),
  };
}

export function valuesOf(saved: Connection, provider: Provider): ConnectionValues {
  return {
    on: saved.on,
    name: saved.name,
    events: [...saved.events],
    warnings: saved.warnings,
    posters: saved.posters,
    node: saved.node ?? "",
    settings: Object.fromEntries(
      settingsOf(provider).map((field) => [
        field.key,
        controlOf(field, saved.values[field.key] ?? field.default),
      ]),
    ),
    secrets: Object.fromEntries(secretsOf(provider).map((field) => [field.key, ""])),
  };
}

const takesNode = (provider: Provider) => provider.fields.some((field) => field.kind === "node");

/** The draft as the add and draft Test calls take it, a secret only when one is typed. */
export function draftOf(values: ConnectionValues, provider: Provider): Draft {
  const typed = secretsOf(provider).flatMap((field): [string, string][] => {
    const value = values.secrets[field.key];
    return typeof value === "string" && value !== "" ? [[field.key, secretWire(field, value)]] : [];
  });
  const settings = settingsOf(provider).flatMap((field): [string, unknown][] => {
    const value = wireOf(field, values.settings[field.key] ?? "");
    return value === null || value === "" ? [] : [[field.key, value]];
  });
  return {
    provider: provider.id,
    name: values.name.trim(),
    on: values.on,
    events: values.events,
    warnings: values.warnings,
    posters: values.posters,
    ...(takesNode(provider) && values.node !== "" && { node: values.node }),
    values: Object.fromEntries(settings),
    secrets: Object.fromEntries(typed),
  };
}

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

/** Only what changed from saved, with typed secrets, and null for a removed one. */
export function editOf(saved: Connection, values: ConnectionValues, provider: Provider): Edit {
  const name = values.name.trim();
  const settings = settingsOf(provider).flatMap((field): [string, unknown][] => {
    const value = wireOf(field, values.settings[field.key] ?? "");
    const was = wireOf(field, controlOf(field, saved.values[field.key] ?? field.default));
    return same(value, was) ? [] : [[field.key, value]];
  });
  const secrets = secretsOf(provider).flatMap((field): [string, string | null][] => {
    const value = values.secrets[field.key];
    if (value === null) {
      return [[field.key, null]];
    }
    return value === undefined || value === "" ? [] : [[field.key, secretWire(field, value)]];
  });
  return {
    ...(name !== saved.name && { name }),
    ...(values.on !== saved.on && { on: values.on }),
    ...(!same(values.events.toSorted(), saved.events.toSorted()) && { events: values.events }),
    ...(values.warnings !== saved.warnings && { warnings: values.warnings }),
    ...(values.posters !== saved.posters && { posters: values.posters }),
    ...(takesNode(provider) && values.node !== (saved.node ?? "") && { node: values.node }),
    ...(settings.length > 0 && { values: Object.fromEntries(settings) }),
    ...(secrets.length > 0 && { secrets: Object.fromEntries(secrets) }),
  };
}

/** The form's name for a field the API names by its key. */
export function formNameOf(provider: Provider, key: string): string {
  const field = provider.fields.find((each) => each.key === key);
  if (field === undefined || field.kind === "node") {
    return key;
  }
  return field.secret ? `secrets.${key}` : `settings.${key}`;
}

/** Whether a field the API names sits under Details. */
export const underDetails = (provider: Provider, key: string): boolean =>
  provider.fields.some((field) => field.key === key && field.details);

const Named = z.object({ field: z.string().nullish() });

const namedOf = (error: ApiError): string | undefined => Named.parse(error.body).field ?? undefined;

export const marking = "Change this, then test again.";

/** What a refused save or Test shows: a failed Test on the Test line with its field marked, or the refusal on its field. */
export function answerOf(
  error: unknown,
  provider: Provider,
): { outcome?: TestOutcome; errors: SubmitErrors; field?: string } {
  if (isProblem(error, "notification_test_failed")) {
    const field = namedOf(error);
    return {
      outcome: { passed: false, sentence: `The Test failed. ${sentenceOf(error)}` },
      errors: { fields: field === undefined ? {} : { [formNameOf(provider, field)]: marking } },
      ...(field !== undefined && { field }),
    };
  }
  if (isProblem(error, "notification_field_refused")) {
    const field = namedOf(error);
    return field === undefined
      ? { errors: { form: sentenceOf(error), fields: {} } }
      : { errors: { fields: { [formNameOf(provider, field)]: sentenceOf(error) } }, field };
  }
  return { errors: submitErrors(error) };
}
