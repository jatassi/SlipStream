import type { FieldError } from "./fieldError.ts";

export interface Problem {
  code: string;
  detail?: string;
  errors?: FieldError[];
  status: number;
  title: string;
}
