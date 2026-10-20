import type { ListTorrents500Code } from "./listTorrents500Code.ts";
import type { Problem } from "./problem.ts";

export type ListTorrents500 = Problem & {
  code?: ListTorrents500Code;
};
