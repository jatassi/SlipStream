import type { ListNotificationEvents500Code } from "./listNotificationEvents500Code.ts";
import type { Problem } from "./problem.ts";

export type ListNotificationEvents500 = Problem & {
  code?: ListNotificationEvents500Code;
};
