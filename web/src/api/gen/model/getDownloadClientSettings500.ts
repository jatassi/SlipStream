import type { GetDownloadClientSettings500Code } from "./getDownloadClientSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetDownloadClientSettings500 = Problem & {
  code?: GetDownloadClientSettings500Code;
};
