import type { Problem } from "./problem.ts";
import type { SetMetadataKey403Code } from "./setMetadataKey403Code.ts";

export type SetMetadataKey403 = Problem & {
  code?: SetMetadataKey403Code;
};
