import { UnreachableError, fieldErrors, isApiError } from "../api/errors";
import { createFormHookContexts } from "@tanstack/react-form";

export const { fieldContext, formContext, useFieldContext, useFormContext } =
  createFormHookContexts();

export function errorText(errors: readonly unknown[]): string | undefined {
  for (const error of errors) {
    if (typeof error === "string") {
      return error;
    }
    if (
      typeof error === "object" &&
      error !== null &&
      "message" in error &&
      typeof error.message === "string"
    ) {
      return error.message;
    }
  }
  return undefined;
}

export type SubmitErrors = { form?: string; fields: Record<string, string> };

export function submitErrors(
  error: unknown,
  fieldForCode: Readonly<Record<string, string>> = {},
): SubmitErrors {
  if (error instanceof UnreachableError) {
    return { form: error.message, fields: {} };
  }
  if (!isApiError(error)) {
    throw error;
  }
  if (error.code === "invalid") {
    return { fields: fieldErrors(error) };
  }
  const field = fieldForCode[error.code];
  return field === undefined
    ? { form: error.title, fields: {} }
    : { fields: { [field]: error.title } };
}
