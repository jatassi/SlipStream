import type { ListTorrents422Code } from "./listTorrents422Code.ts";
import type { Problem } from "./problem.ts";

export type ListTorrents422 = Problem & {
  code?: ListTorrents422Code;
};
