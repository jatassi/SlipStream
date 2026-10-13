import type { WorksDateValue } from "./worksDateValue.ts";
import type { WorksDurationValue } from "./worksDurationValue.ts";
import type { WorksNumberValue } from "./worksNumberValue.ts";
import type { WorksTextValue } from "./worksTextValue.ts";

export type WorksFieldValue =
  | WorksTextValue
  | WorksNumberValue
  | WorksDateValue
  | WorksDurationValue;
