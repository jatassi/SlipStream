import type { ListTasks500Code } from "./listTasks500Code.ts";
import type { Problem } from "./problem.ts";

export type ListTasks500 = Problem & {
  code?: ListTasks500Code;
};
