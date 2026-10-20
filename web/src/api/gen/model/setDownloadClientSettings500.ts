import type { Problem } from "./problem.ts";
import type { SetDownloadClientSettings500Code } from "./setDownloadClientSettings500Code.ts";

export type SetDownloadClientSettings500 = Problem & {
  code?: SetDownloadClientSettings500Code;
};
