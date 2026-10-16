import type { ListTasks401Code } from "./listTasks401Code.ts";
import type { Problem } from "./problem.ts";

export type ListTasks401 = Problem & {
  code?: ListTasks401Code;
};
