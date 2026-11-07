import { isProblem } from "../api/errors";
import type { IndexersCaptchaImage, TestIndexer422Cause } from "../api/gen/model";

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

const codeNeeded: TestIndexer422Cause = "code_needed";

/** A failed Test that met a sign-in needing a two-factor code. */
export function needsCode(error: unknown): boolean {
  if (!isProblem(error, "indexer_test_failed")) {
    return false;
  }
  const { body } = error;
  return typeof body === "object" && body !== null && "cause" in body && body.cause === codeNeeded;
}
