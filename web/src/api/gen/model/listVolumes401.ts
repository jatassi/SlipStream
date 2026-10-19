import type { ListVolumes401Code } from "./listVolumes401Code.ts";
import type { Problem } from "./problem.ts";

export type ListVolumes401 = Problem & {
  code?: ListVolumes401Code;
};
