import { z } from "zod";

import { isProblem } from "../api/errors";
import type { IndexersCaptchaImage } from "../api/gen/model";

const isCaptcha = (value: unknown): value is IndexersCaptchaImage =>
  typeof value === "object" &&
  value !== null &&
  "attempt" in value &&
  typeof value.attempt === "string" &&
  "image" in value &&
  typeof value.image === "string" &&
  "expiresAt" in value &&
  typeof value.expiresAt === "string";

/** The captcha a failed Test hands back, since each one is answered once. */
export function captchaOf(error: unknown): IndexersCaptchaImage | undefined {
  if (!isProblem(error, "indexer_test_failed")) {
    return undefined;
  }
  const { body } = error;
  const captcha = typeof body === "object" && body !== null && "captcha" in body && body.captcha;
  return isCaptcha(captcha) ? captcha : undefined;
}

// indexer_test_failed carries no cause in the API yet, so this reads a
// cause member the Controller is asked to add beside the captcha.
const codeCause = z.object({ cause: z.literal("code_needed") });

/** A failed Test that met a sign-in needing a two-factor code. */
export const needsCode = (error: unknown): boolean =>
  isProblem(error, "indexer_test_failed") && codeCause.safeParse(error.body).success;
