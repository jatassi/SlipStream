import type { WorksDateValue } from "./worksDateValue.ts";
import type { WorksDurationValue } from "./worksDurationValue.ts";
import type { WorksLinkValue } from "./worksLinkValue.ts";
import type { WorksNumberValue } from "./worksNumberValue.ts";
import type { WorksPeopleValue } from "./worksPeopleValue.ts";
import type { WorksTextsValue } from "./worksTextsValue.ts";
import type { WorksTextValue } from "./worksTextValue.ts";

export type WorksFieldValue =
  | WorksTextValue
  | WorksNumberValue
  | WorksDateValue
  | WorksDurationValue
  | WorksTextsValue
  | WorksPeopleValue
  | WorksLinkValue;
