import type { ImportsConfirmFile } from "./importsConfirmFile.ts";
import type { ImportsDeclineFile } from "./importsDeclineFile.ts";
import type { ImportsForceFile } from "./importsForceFile.ts";
import type { ImportsMatchFile } from "./importsMatchFile.ts";
import type { ImportsNotThisReleaseFile } from "./importsNotThisReleaseFile.ts";
import type { ImportsRetryFile } from "./importsRetryFile.ts";

export type ImportsFileAction =
  | ImportsConfirmFile
  | ImportsRetryFile
  | ImportsDeclineFile
  | ImportsNotThisReleaseFile
  | ImportsForceFile
  | ImportsMatchFile;
