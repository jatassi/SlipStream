import type { ListNotifications422Code } from "./listNotifications422Code.ts";
import type { Problem } from "./problem.ts";

export type ListNotifications422 = Problem & {
  code?: ListNotifications422Code;
};
