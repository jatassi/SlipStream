import * as zod from "zod";

import {
  ReconcilingForgetRequest,
  ReconcilingPage,
  ReconcilingReaddRequest,
  ReconcilingRehomeRequest,
  ReconcilingRehomed,
  ReconcilingReviewPage,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Read what Reconciling waits on after a restore
 */
export const GetReconcilingResponse = ReconcilingPage;

/**
 * @summary Finish Reconciling now
 */
export const FinishReconcilingResponse = zod.void();

/**
 * @summary Stop waiting for a Node's torrents
 */
export const ContinueReconcilingNodeParams = zod.object({
  nodeId: zod.uuid(),
});

export const ContinueReconcilingNodeResponse = zod.void();

/**
 * @summary Save the old Controller's library roots at their places on this machine
 */
export const RehomeReconcilingBody = ReconcilingRehomeRequest;

export const RehomeReconcilingResponse = ReconcilingRehomed;

/**
 * @summary Stop waiting for a library root's scan
 */
export const ContinueReconcilingRootParams = zod.object({
  rootId: zod.uuid(),
});

export const ContinueReconcilingRootResponse = zod.void();

/**
 * @summary List a page of the review list
 */
export const listReconcilingTorrentsQueryAfterMax = 200;

export const listReconcilingTorrentsQueryLimitDefault = 100;
export const listReconcilingTorrentsQueryLimitMax = 500;

export const ListReconcilingTorrentsQueryParams = zod.object({
  reason: zod
    .enum(["not_held", "conflict"])
    .exactOptional()
    .describe("Only this group's reason; left out, any"),
  found: zod
    .enum(["not_looked", "no_files", "files"])
    .exactOptional()
    .describe("Only torrents whose files were found so; left out, any"),
  after: zod
    .string()
    .max(listReconcilingTorrentsQueryAfterMax)
    .exactOptional()
    .describe("The next of the previous page; left out, the first page"),
  limit: zod
    .int()
    .min(1)
    .max(listReconcilingTorrentsQueryLimitMax)
    .default(listReconcilingTorrentsQueryLimitDefault),
});

export const ListReconcilingTorrentsResponse = ReconcilingReviewPage;

/**
 * @summary Forget torrents no Node holds
 */
export const ForgetReconcilingTorrentsBody = ReconcilingForgetRequest;

export const ForgetReconcilingTorrentsResponse = zod.void();

/**
 * @summary Re-add torrents on the review list
 */
export const ReaddReconcilingTorrentsBody = ReconcilingReaddRequest;

export const ReaddReconcilingTorrentsResponse = zod.void();
