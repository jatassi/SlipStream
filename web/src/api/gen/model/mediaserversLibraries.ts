import type { MediaserversLibraryView } from "./mediaserversLibraryView.ts";
import type { MediaserversMappingView } from "./mediaserversMappingView.ts";

export interface MediaserversLibraries {
  libraries: MediaserversLibraryView[];
  /** Every library root, pre-filled where it has no saved mapping */
  mappings: MediaserversMappingView[];
}
