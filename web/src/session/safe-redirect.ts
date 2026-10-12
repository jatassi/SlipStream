import { z } from "zod";

import { LiveSessionEnded } from "../api/gen/zod/slipStream.schemas.zod";

export const isSafeRedirect = (raw: string): boolean =>
  raw.startsWith("/") && !raw.startsWith("//") && !raw.startsWith("/\\");

export const safeRedirect = (raw: string | undefined): string =>
  raw !== undefined && isSafeRedirect(raw) ? raw : "/";

export const signInSearch = z.object({
  redirect: z.string().refine(isSafeRedirect).optional().catch(undefined),
  reason: LiveSessionEnded.shape.reason.optional().catch(undefined),
});
