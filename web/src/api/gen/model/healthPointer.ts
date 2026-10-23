import type { HealthCategoryID } from "./healthCategoryID.ts";

export interface HealthPointer {
  category: HealthCategoryID;
  key: string;
  text: string;
}
