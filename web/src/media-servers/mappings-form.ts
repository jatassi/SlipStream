import type {
  MediaserversLibraries as Libraries,
  MediaserversMappingInput as MappingInput,
  MediaserversMappingView as MappingView,
} from "../api/gen/model";

export type MappingValue = { rootId: string; shown: boolean; serverFolder: string };

export type MappingsValues = { mappings: MappingValue[] };

export const mappingValuesOf = ({ mappings }: Libraries): MappingsValues => ({
  mappings: mappings.map((view) => ({
    rootId: view.rootId,
    shown: view.shown,
    serverFolder: view.serverFolder ?? "",
  })),
});

export const mappingsBodyOf = ({ mappings }: MappingsValues): MappingInput[] =>
  mappings.map(({ rootId, shown, serverFolder }) =>
    shown ? { rootId, shown, serverFolder: serverFolder.trim() } : { rootId, shown },
  );

/** Every library folder the server lists, once each, in its order. */
export const foldersOf = ({ libraries }: Libraries): readonly string[] => [
  ...new Set(libraries.flatMap((library) => library.folders)),
];

/**
 * What the row says under its folder. The guess and the library describe the
 * folder SlipStream filled in or last saved, so an edited folder shows neither.
 */
export function folderNoteOf(view: MappingView, folder: string, server: string): string {
  if (folder.trim() !== (view.serverFolder ?? "")) {
    return `Save checks that the folder is inside one of ${server}'s library folders.`;
  }
  if (view.guess) {
    return `SlipStream guessed this folder from the folder names. Check that ${server} sees the root here.`;
  }
  return view.library === undefined ? "" : `In ${view.library.name}.`;
}
