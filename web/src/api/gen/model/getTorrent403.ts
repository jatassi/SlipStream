import type { GetTorrent403Code } from "./getTorrent403Code.ts";
import type { Problem } from "./problem.ts";

export type GetTorrent403 = Problem & {
  code?: GetTorrent403Code;
};
