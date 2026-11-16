import type { ListNotifications401Code } from "./listNotifications401Code.ts";
import type { Problem } from "./problem.ts";

export type ListNotifications401 = Problem & {
  code?: ListNotifications401Code;
};
