import type { ListDownloadClients401Code } from "./listDownloadClients401Code.ts";
import type { Problem } from "./problem.ts";

export type ListDownloadClients401 = Problem & {
  code?: ListDownloadClients401Code;
};
