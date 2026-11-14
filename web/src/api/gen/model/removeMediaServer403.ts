import type { Problem } from "./problem.ts";
import type { RemoveMediaServer403Code } from "./removeMediaServer403Code.ts";

export type RemoveMediaServer403 = Problem & {
  code?: RemoveMediaServer403Code;
};
