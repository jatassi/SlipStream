import * as zod from "zod";

import {
  ImportIDsInputBody,
  ImportsAddedList,
  ImportsImportEntry,
  ImportsList,
  ImportsRootAction,
  ImportsSettings,
  LibraryScanSettingsInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the unfinished Imports, with what each root's last library scan added
 */
export const ListImportsQueryParams = zod.object({
  source: zod
    .enum(["grab", "root", "folder"])
    .exactOptional()
    .describe("Only Imports from this source; every source's when left out"),
  rootId: zod
    .uuid()
    .exactOptional()
    .describe("Only this root's Imports and lead; every root's when left out"),
  ignored: zod
    .boolean()
    .exactOptional()
    .describe("List the ignored Imports instead of the ones that need the Admin"),
});

export const ListImportsResponse = ImportsList;

/**
 * @summary Add each root Import's suggested Work to its root and attach its files
 */
export const AddRootImportsBody = ImportIDsInputBody;

export const AddRootImportsResponse = ImportsAddedList;

/**
 * @summary Ignore Imports until their files change
 */
export const IgnoreImportsBody = ImportIDsInputBody;

export const IgnoreImportsResponse = zod.void();

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
