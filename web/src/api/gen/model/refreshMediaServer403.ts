import type { Problem } from "./problem.ts";
import type { RefreshMediaServer403Code } from "./refreshMediaServer403Code.ts";

export type RefreshMediaServer403 = Problem & {
  code?: RefreshMediaServer403Code;
};
