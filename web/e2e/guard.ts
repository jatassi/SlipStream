import type { BrowserContext } from "@playwright/test";
import { expect } from "@playwright/test";

import type { Manifest } from "../harness/manifest.ts";
import { expectAccessible } from "./accessible.ts";
import { pageErrors } from "./page-errors.ts";

declare global {
  interface Window {
    reportCspViolation: (violation: string) => Promise<void>;
  }
}

/** The Node's origin and the origins of the fronts allow names by target. */
export function origins(
  m: Manifest,
  origin: string,
  allow: readonly string[],
): ReadonlySet<string> {
  return new Set([
    origin,
    ...m.fronts.flatMap((f) =>
      allow.includes(f.target) && f.url !== undefined ? [new URL(f.url).origin] : [],
    ),
  ]);
}

/**
 * Records a context's requests to any origin but those given, its CSP
 * violations and page errors, and answers the check of them and of each
 * page's accessibility, run as the context closes.
 */
export async function guard(
  context: BrowserContext,
  allowed: ReadonlySet<string>,
): Promise<() => Promise<void>> {
  const foreign: string[] = [];
  const violations: string[] = [];
  const errors = await pageErrors(context);
  context.on("request", (request) => {
    const url = new URL(request.url());
    if (url.protocol !== "data:" && url.protocol !== "blob:" && !allowed.has(url.origin)) {
      foreign.push(request.url());
    }
  });
  await context.exposeBinding("reportCspViolation", (_source, violation: string) => {
    violations.push(violation);
  });
  // The page's CSP doesn't apply to an init script.
  await context.addInitScript(() => {
    document.addEventListener("securitypolicyviolation", (event) => {
      void window.reportCspViolation(
        `${event.violatedDirective} ${event.blockedURI} at ${event.sourceFile}:${event.lineNumber}`,
      );
    });
  });
  return async () => {
    expect.soft(foreign, "requests to an origin other than the Node's").toEqual([]);
    expect.soft(violations, "CSP violations").toEqual([]);
    expect.soft(errors, "page errors").toEqual([]);
    await Promise.all(context.pages().map((page) => expectAccessible(page)));
  };
}
