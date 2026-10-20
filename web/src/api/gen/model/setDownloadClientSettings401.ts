import type { Problem } from "./problem.ts";
import type { SetDownloadClientSettings401Code } from "./setDownloadClientSettings401Code.ts";

export type SetDownloadClientSettings401 = Problem & {
  code?: SetDownloadClientSettings401Code;
};
