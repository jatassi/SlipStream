import * as zod from "zod";

import {
  FormatsCustomFormat,
  FormatsFormatChange,
  FormatsFormatDeletion,
  FormatsFormatList,
  FormatsNewFormat,
  FormatsParse,
  FormatsParseResults,
  FormatsPaste,
  FormatsPasted,
  FormatsSchema,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Custom Formats, each with the Targets that score it
 */
export const ListCustomFormatsResponse = FormatsFormatList;

/**
 * @summary Make a Custom Format in SlipStream's own values. No Target scores it until one is given a score for it
 */
export const CreateCustomFormatBody = FormatsNewFormat;

export const CreateCustomFormatResponse = FormatsCustomFormat;

/**
 * @summary Add Custom Formats from Radarr, Sonarr or TRaSH Guides JSON, replacing or keeping beside each one of the same name, or preview what the paste adds and replaces
 */
export const pasteCustomFormatsQueryUnwantedDefault = `delete`;
export const pasteCustomFormatsQueryConfirmMin = 0;

export const pasteCustomFormatsQueryConfirmMadeMin = 0;

export const PasteCustomFormatsQueryParams = zod.object({
  preview: zod.boolean().exactOptional().describe("Return the change's Impact and save nothing"),
  unwanted: zod
    .enum(["delete", "keep"])
    .default(pasteCustomFormatsQueryUnwantedDefault)
    .describe("What happens to the files no Target uses after the change"),
  confirm: zod
    .int()
    .min(pasteCustomFormatsQueryConfirmMin)
    .exactOptional()
    .describe(
      "The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed",
    ),
  confirmMade: zod
    .int()
    .min(pasteCustomFormatsQueryConfirmMadeMin)
    .exactOptional()
    .describe(
      "The made file count the Admin confirmed from the preview: a save that would make more is refused with impact_changed",
    ),
});

export const PasteCustomFormatsBody = FormatsPaste;

export const PasteCustomFormatsResponse = FormatsPasted;

/**
 * @summary List the condition kinds each app's Custom Formats take, with each kind's fields and values in words
 */
export const CustomFormatSchemaResponse = FormatsSchema;

/**
 * @summary Delete a Custom Format and every Target's score for it, and re-rank the files of every Work whose Targets scored it, or preview what the change would do
 */
export const DeleteCustomFormatParams = zod.object({
  formatId: zod.uuid(),
});

export const deleteCustomFormatQueryUnwantedDefault = `delete`;
export const deleteCustomFormatQueryConfirmMin = 0;

export const deleteCustomFormatQueryConfirmMadeMin = 0;

export const DeleteCustomFormatQueryParams = zod.object({
  preview: zod.boolean().exactOptional().describe("Return the change's Impact and save nothing"),
  unwanted: zod
    .enum(["delete", "keep"])
    .default(deleteCustomFormatQueryUnwantedDefault)
    .describe("What happens to the files no Target uses after the change"),
  confirm: zod
    .int()
    .min(deleteCustomFormatQueryConfirmMin)
    .exactOptional()
    .describe(
      "The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed",
    ),
  confirmMade: zod
    .int()
    .min(deleteCustomFormatQueryConfirmMadeMin)
    .exactOptional()
    .describe(
      "The made file count the Admin confirmed from the preview: a save that would make more is refused with impact_changed",
    ),
});

export const DeleteCustomFormatResponse = FormatsFormatDeletion;

/**
 * @summary Get a Custom Format
 */
export const GetCustomFormatParams = zod.object({
  formatId: zod.uuid(),
});

export const GetCustomFormatResponse = FormatsCustomFormat;

/**
 * @summary Replace a Custom Format's name, renaming flag and conditions, and re-rank the files of every Work whose Targets score it, or preview what the change would do
 */
export const UpdateCustomFormatParams = zod.object({
  formatId: zod.uuid(),
});

export const updateCustomFormatQueryUnwantedDefault = `delete`;
export const updateCustomFormatQueryConfirmMin = 0;

export const updateCustomFormatQueryConfirmMadeMin = 0;

export const UpdateCustomFormatQueryParams = zod.object({
  preview: zod.boolean().exactOptional().describe("Return the change's Impact and save nothing"),
  unwanted: zod
    .enum(["delete", "keep"])
    .default(updateCustomFormatQueryUnwantedDefault)
    .describe("What happens to the files no Target uses after the change"),
  confirm: zod
    .int()
    .min(updateCustomFormatQueryConfirmMin)
    .exactOptional()
    .describe(
      "The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed",
    ),
  confirmMade: zod
    .int()
    .min(updateCustomFormatQueryConfirmMadeMin)
    .exactOptional()
    .describe(
      "The made file count the Admin confirmed from the preview: a save that would make more is refused with impact_changed",
    ),
});

export const UpdateCustomFormatBody = FormatsNewFormat;

export const UpdateCustomFormatResponse = FormatsFormatChange;

/**
 * @summary Copy a Custom Format with its app and conditions as they are, under its name with a number after it. No Target scores the copy until one is given a score for it
 */
export const DuplicateCustomFormatParams = zod.object({
  formatId: zod.uuid(),
});

export const DuplicateCustomFormatResponse = FormatsCustomFormat;

/**
 * @summary Read release or file names as a search would: the facts, the Work each matches, every Custom Format with each condition's result, and the score and verdict at each Target
 */
export const ParseReleasesBody = FormatsParse;

export const ParseReleasesResponse = FormatsParseResults;
