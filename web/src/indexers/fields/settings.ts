import type { IndexersDefinitionField } from "../../api/gen/model";

/** A definition field that holds a value, which an info field and the captcha don't. */
export const holdsValue = ({ type }: IndexersDefinitionField) =>
  type === "text" ||
  type === "password" ||
  type === "checkbox" ||
  type === "select" ||
  type === "number" ||
  type === "multiselect";

export function startValue(field: IndexersDefinitionField): string {
  if (field.type === "checkbox") {
    return field.default === "true" ? "true" : "false";
  }
  if (field.type === "select") {
    return field.default ?? field.options[0]?.value ?? "";
  }
  return holdsValue(field) ? (field.default ?? "") : "";
}

/** A setting as the API takes it; an empty multiselect is a choice of none, so it is sent too. */
export const sends = (field: IndexersDefinitionField, value: string) =>
  holdsValue(field) && (value !== "" || field.type === "multiselect");

/** The settings by name, leaving out those left empty so they take their defaults. */
export const settingsOf = (fields: readonly IndexersDefinitionField[], values: readonly string[]) =>
  Object.fromEntries(
    fields.flatMap((field, index) => {
      const value = values[index] ?? "";
      return sends(field, value) ? [[field.name, value]] : [];
    }),
  );

/** Where a refusal of values.<name> lands: the setting's field by its index. */
export function settingAt(fields: readonly IndexersDefinitionField[], location: string): string {
  const index = fields.findIndex((field) => `values.${field.name}` === location);
  return index < 0 ? location : `values[${index}]`;
}

/** A multiselect's value, the option values it holds, comma-separated. */
export const chosenOf = (value: string) => (value === "" ? [] : value.split(","));

/** A number field's range in words, such as "From 1 to 5". */
export function rangeWords({ min, max }: IndexersDefinitionField): string | undefined {
  if (min !== null && max !== null) {
    return `from ${min} to ${max}`;
  }
  if (min !== null) {
    return `${min} or more`;
  }
  return max === null ? undefined : `${max} or less`;
}

const outside = (field: IndexersDefinitionField, number: number) =>
  (field.min !== null && number < field.min) || (field.max !== null && number > field.max);

/** A required setting left empty, which a saved sealed one isn't: its empty value keeps it. */
export const requiredError = (field: IndexersDefinitionField, saved: boolean, value: string) =>
  field.required && !saved && value === "" ? `${field.label} is required.` : undefined;

/** Why a number field's value doesn't fit it, checked before the Test; empty takes the default. */
export function numberError(field: IndexersDefinitionField, value: string): string | undefined {
  const text = value.trim();
  if (text === "") {
    return undefined;
  }
  if (!/^-?\d+$/u.test(text)) {
    return "Enter a whole number.";
  }
  if (!outside(field, Number(text))) {
    return undefined;
  }
  return field.min !== null && field.max !== null
    ? `Enter a number ${rangeWords(field)}.`
    : `Enter ${rangeWords(field)}.`;
}
