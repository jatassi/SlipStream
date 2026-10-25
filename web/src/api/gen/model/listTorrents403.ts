import type { ListTorrents403Code } from "./listTorrents403Code.ts";
import type { Problem } from "./problem.ts";

export type ListTorrents403 = Problem & {
  code?: ListTorrents403Code;
};
