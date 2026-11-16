import type { ListNotifications500Code } from "./listNotifications500Code.ts";
import type { Problem } from "./problem.ts";

export type ListNotifications500 = Problem & {
  code?: ListNotifications500Code;
};
