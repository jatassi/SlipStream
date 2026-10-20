import type { BrowserContext, Page } from "@playwright/test";

declare global {
  interface Window {
    reportLeaving: (leaving: boolean) => Promise<void>;
  }
}

const refusal = /^Fetch API cannot load .+ due to access control checks\.$/u;

// WebKit refuses a fetch a page starts after its beforeunload and reports it
// as a page error; Chromium and Firefox drop it silently.
export async function pageErrors(context: BrowserContext): Promise<string[]> {
  const errors: string[] = [];
  const leaving = new Set<Page>();
  context.on("weberror", (error) => {
    const page = error.page();
    const text = String(error.error());
    if (page === null || !leaving.has(page) || !refusal.test(text)) {
      errors.push(text);
    }
  });
  if (context.browser()?.browserType().name() === "webkit") {
    await context.exposeBinding("reportLeaving", ({ page }, isLeaving: boolean) => {
      if (isLeaving) {
        leaving.add(page);
      } else {
        leaving.delete(page);
      }
    });
    await context.addInitScript(() => {
      void window.reportLeaving(false);
      window.addEventListener("beforeunload", () => {
        void window.reportLeaving(true);
      });
    });
  }
  return errors;
}
