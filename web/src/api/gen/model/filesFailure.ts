import type { FilesConversionFailure } from "./filesConversionFailure.ts";
import type { FilesGrabFailure } from "./filesGrabFailure.ts";

export type FilesFailure = FilesGrabFailure | FilesConversionFailure;
