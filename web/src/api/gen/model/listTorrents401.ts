import type { ListTorrents401Code } from "./listTorrents401Code.ts";
import type { Problem } from "./problem.ts";

export type ListTorrents401 = Problem & {
  code?: ListTorrents401Code;
};
