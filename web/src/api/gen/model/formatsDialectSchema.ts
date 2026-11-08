import type { FormatsDialectSchemaDialect } from "./formatsDialectSchemaDialect.ts";
import type { FormatsOfferedKind } from "./formatsOfferedKind.ts";

export interface FormatsDialectSchema {
  dialect: FormatsDialectSchemaDialect;
  /** In upstream's order; a kind only the other app has is left out */
  kinds: FormatsOfferedKind[];
}
