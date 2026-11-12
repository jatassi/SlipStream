import * as zod from "zod";

import {
  ImportFolderInputBody,
  ImportIDsInputBody,
  ImportsAddedList,
  ImportsExtensionImpact,
  ImportsFileAction,
  ImportsImportEntry,
  ImportsImportingSettings,
  ImportsImportsList,
  ImportsList,
  ImportsRootAction,
  ImportsSettings,
  LibraryScanSettingsInputBody,
  MoveFilesInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the unfinished Imports, with what each root's last library scan added
 */
export const listImportsQueryLimitMax = 500;

export const ListImportsQueryParams = zod.object({
  source: zod
    .enum(["grab", "root", "folder"])
    .exactOptional()
    .describe("Only Imports from this source; every source's when left out"),
  rootId: zod
    .uuid()
    .exactOptional()
    .describe("Only this root's Imports and lead; every root's when left out"),
  work: zod
    .uuid()
    .exactOptional()
    .describe("Only the Imports into this Work; every Work's when left out"),
  ignored: zod
    .boolean()
    .exactOptional()
    .describe("List the ignored Imports instead of the ones that need the Admin"),
  waiting: zod
    .boolean()
    .exactOptional()
    .describe("Only the Imports with a file that waits for the Admin"),
  after: zod
    .uuid()
    .exactOptional()
    .describe("The next value of the page before; the first page when left out"),
  limit: zod
    .int()
    .min(1)
    .max(listImportsQueryLimitMax)
    .exactOptional()
    .describe("The most Imports a page holds; every one when left out"),
});

export const ListImportsResponse = ImportsList;

/**
 * @summary Open the Imports of a folder on a Node, one per Work and Edition its files match
 */
export const ImportFolderBody = ImportFolderInputBody;

export const ImportFolderResponse = ImportsImportsList;

/**
 * @summary Add each root Import's suggested Work to its root and attach its files
 */
export const AddRootImportsBody = ImportIDsInputBody;

export const AddRootImportsResponse = ImportsAddedList;

/**
 * @summary Import every file of these Imports that waits for confirmation
 */
export const ConfirmImportsBody = ImportIDsInputBody;

export const ConfirmImportsResponse = ImportsImportsList;

/**
 * @summary Ignore Imports until their files change
 */
export const IgnoreImportsBody = ImportIDsInputBody;

export const IgnoreImportsResponse = zod.void();

/**
 * @summary Get the Importing settings
 */
export const GetImportingSettingsResponse = ImportsImportingSettings;

/**
 * @summary Change the Importing settings
 */
export const UpdateImportingSettingsBody = LibraryScanSettingsInputBody;

export const UpdateImportingSettingsResponse = ImportsImportingSettings;

/**
 * @summary Count the library and pending files a video extension list stops counting as video
 */
export const getExtensionImpactQueryExtensionsMax = 100;

export const GetExtensionImpactQueryParams = zod.object({
  extensions: zod
    .array(zod.string())
    .min(1)
    .max(getExtensionImpactQueryExtensionsMax)
    .nullish()
    .describe("The video file extensions to judge, such as mkv, without the dot"),
});

export const GetExtensionImpactResponse = ImportsExtensionImpact;

/**
 * @summary Turn Move files on or off for folder Imports
 */
export const SetImportMoveFilesBody = MoveFilesInputBody;

export const SetImportMoveFilesResponse = zod.void();

/**
 * @summary Get the Library scan settings
 */
export const GetLibraryScanSettingsResponse = ImportsSettings;

/**
 * @summary Change the Library scan settings
 */
export const UpdateLibraryScanSettingsBody = LibraryScanSettingsInputBody;

export const UpdateLibraryScanSettingsResponse = ImportsSettings;

/**
 * @summary Bring ignored Imports back to the ones that need the Admin
 */
export const UnignoreImportsBody = ImportIDsInputBody;

export const UnignoreImportsResponse = zod.void();

/**
 * @summary Get an Import with its files
 */
export const GetImportParams = zod.object({
  id: zod.uuid(),
});

export const GetImportResponse = ImportsImportEntry;

/**
 * @summary Match a root Import to a record at the Content Type's source, or import its chosen files into a Work in its root
 */
export const ActOnRootImportParams = zod.object({
  id: zod.uuid(),
});

export const ActOnRootImportBody = ImportsRootAction;

export const ActOnRootImportResponse = ImportsImportEntry;

/**
 * @summary Confirm, retry, decline, force or match one file of an Import
 */
export const actOnImportFilePathPositionMin = 0;

export const ActOnImportFileParams = zod.object({
  id: zod.uuid(),
  position: zod.int().min(actOnImportFilePathPositionMin),
});

export const ActOnImportFileBody = ImportsFileAction;

export const ActOnImportFileResponse = ImportsImportEntry;
