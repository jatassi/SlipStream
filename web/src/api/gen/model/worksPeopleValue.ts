import type { WorksPeopleValueKind } from "./worksPeopleValueKind.ts";
import type { WorksPerson } from "./worksPerson.ts";

export interface WorksPeopleValue {
  key: string;
  kind: WorksPeopleValueKind;
  /** In the source's order */
  people: WorksPerson[];
}
