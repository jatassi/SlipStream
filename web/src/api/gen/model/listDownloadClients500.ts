import type { ListDownloadClients500Code } from "./listDownloadClients500Code.ts";
import type { Problem } from "./problem.ts";

export type ListDownloadClients500 = Problem & {
  code?: ListDownloadClients500Code;
};
