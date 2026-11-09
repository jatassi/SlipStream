import * as zod from "zod";

import {
  ImportIntoInputBody,
  ImportsAddedList,
  ImportsEntry,
  ImportsImported,
  ImportsList,
  ImportsSettings,
  LibraryScanSettingsInputBody,
  MatchPendingInputBody,
  PendingIDsInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the pending imports, with what each root's last library scan added
 */
export const ListPendingImportsQueryParams = zod.object({
  rootId: zod
    .uuid()
    .exactOptional()
    .describe("Only this root's pending imports and lead; every root's when left out"),
  ignored: zod
    .boolean()
    .exactOptional()
    .describe("List the ignored pending imports instead of the ones that need the Admin"),
});

export const ListPendingImportsResponse = ImportsList;

/**
 * @summary Add each pending import's suggested Work to its root and attach its files
 */
export const AddPendingImportsBody = PendingIDsInputBody;

export const AddPendingImportsResponse = ImportsAddedList;

/**
 * @summary Ignore pending imports until their files change
 */
export const IgnorePendingImportsBody = PendingIDsInputBody;

export const IgnorePendingImportsResponse = zod.void();

/**
 * @summary Bring ignored pending imports back to the ones that need the Admin
 */
export const UnignorePendingImportsBody = PendingIDsInputBody;

export const UnignorePendingImportsResponse = zod.void();

/**
 * @summary Attach a pending import's chosen files to a Work in its root, at the Items chosen
 */
export const ImportPendingIntoParams = zod.object({
  id: zod.uuid(),
});

export const ImportPendingIntoBody = ImportIntoInputBody;

export const ImportPendingIntoResponse = ImportsImported;

/**
 * @summary Make a record at the Content Type's source a pending import's suggestion
 */
export const MatchPendingImportParams = zod.object({
  id: zod.uuid(),
});

export const MatchPendingImportBody = MatchPendingInputBody;

export const MatchPendingImportResponse = ImportsEntry;

/**
 * @summary Get the Library scan settings
 */
export const GetLibraryScanSettingsResponse = ImportsSettings;

/**
 * @summary Change the Library scan settings
 */
export const UpdateLibraryScanSettingsBody = LibraryScanSettingsInputBody;

export const UpdateLibraryScanSettingsResponse = ImportsSettings;
