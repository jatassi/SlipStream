import type { WorksContentType, WorksFieldValue } from "../api/gen/model";

export function fieldText(value: WorksFieldValue): string {
  if (value.kind === "text") {
    return value.text;
  }
  if (value.kind === "number") {
    return String(value.number);
  }
  if (value.kind === "date") {
    return value.date;
  }
  const minutes = Math.round(value.ms / 60_000);
  return minutes < 60 ? `${minutes}m` : `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
}

export function yearOf(fields: WorksFieldValue[]): string | undefined {
  const year = fields.find((field) => field.key === "year");
  return year === undefined ? undefined : fieldText(year);
}

export function enabledType(types: WorksContentType[]): WorksContentType | undefined {
  return types.find((type) => type.enabled);
}

export function typeOf(types: WorksContentType[], id: string): WorksContentType {
  const type = types.find((candidate) => candidate.id === id);
  if (type === undefined) {
    throw new Error(`no Content Type ${id} is registered`);
  }
  return type;
}
