import * as zod from "zod";

import {
  NamingNamingList,
  NamingPreview,
  NamingPreviewInputBody,
  NamingTokenList,
  NamingTypeNaming,
  SetNamingInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List each Content Type's naming settings, with what the save check refuses once renaming is on
 */
export const ListNamingResponse = NamingNamingList;

/**
 * @summary Change a Content Type's naming settings by key, all or none. Every setting must pass the save check as the change leaves them
 */
export const setNamingPathContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const SetNamingParams = zod.object({
  contentType: zod.string().regex(setNamingPathContentTypeRegExp),
});

export const SetNamingBody = SetNamingInputBody;

export const SetNamingResponse = NamingTypeNaming;

/**
 * @summary Name a Content Type's samples, a Work's files and tested file names under a draft of its naming settings, with what the save check refuses. Nothing is stored
 */
export const previewNamingPathContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const PreviewNamingParams = zod.object({
  contentType: zod.string().regex(previewNamingPathContentTypeRegExp),
});

export const PreviewNamingBody = NamingPreviewInputBody;

export const PreviewNamingResponse = NamingPreview;

/**
 * @summary List the tokens each of a Content Type's templates may name
 */
export const listNamingTokensPathContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const ListNamingTokensParams = zod.object({
  contentType: zod.string().regex(listNamingTokensPathContentTypeRegExp),
});

export const ListNamingTokensResponse = NamingTokenList;
