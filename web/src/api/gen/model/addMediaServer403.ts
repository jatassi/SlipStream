import type { AddMediaServer403Code } from "./addMediaServer403Code.ts";
import type { Problem } from "./problem.ts";

export type AddMediaServer403 = Problem & {
  code?: AddMediaServer403Code;
};
