import type { BrowserContext, Page } from "@playwright/test";

declare global {
  interface Window {
    reportLeaving: (leaving: boolean) => Promise<void>;
    reportResizeLoop: (deliveries: string) => Promise<void>;
  }
}

const refusal = /^Fetch API cannot load .+ due to access control checks\.$/u;
const resizeLoop = /ResizeObserver loop/u;

// WebKit refuses a fetch a page starts after its beforeunload and reports it
// as a page error; Chromium and Firefox drop it silently.
export async function pageErrors(context: BrowserContext): Promise<string[]> {
  const errors: string[] = [];
  const leaving = new Set<Page>();
  // Chromium fires a ResizeObserver loop at the window alone and WebKit also
  // reports it as a page error, so the deliveries join that page error and
  // never stand as one of their own. Either can arrive first.
  const unnamedLoops: number[] = [];
  const deliveries: string[] = [];
  const name = (): void => {
    while (unnamedLoops.length > 0 && deliveries.length > 0) {
      const at = unnamedLoops.shift() ?? 0;
      errors[at] += ` (the ResizeObserver deliveries before it: ${deliveries.shift() ?? ""})`;
    }
  };
  context.on("weberror", (error) => {
    const page = error.page();
    const text = String(error.error());
    if (page === null || !leaving.has(page) || !refusal.test(text)) {
      if (resizeLoop.test(text)) {
        unnamedLoops.push(errors.length);
      }
      errors.push(text);
      name();
    }
  });
  await context.exposeBinding("reportResizeLoop", (_source, delivered: string) => {
    deliveries.push(delivered);
    name();
  });
  await context.addInitScript(resizeLoopReporter);
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

/**
 * Wraps ResizeObserver so a "ResizeObserver loop" error names, for each
 * callback of the last few frames, the observer's creator and the targets
 * it was given with their sizes: the error alone says nothing of which
 * observer looped.
 */
function resizeLoopReporter(): void {
  const Native = window.ResizeObserver;
  const recent: { frame: number; line: string }[] = [];
  let frame = 0;
  const tick = (): void => {
    frame += 1;
    requestAnimationFrame(tick);
  };
  requestAnimationFrame(tick);
  // oxlint-disable-next-line unicorn/consistent-function-scoping -- an init script runs alone in the page
  const describe = (el: Element): string => {
    const attrs = [...el.attributes]
      .filter((a) => a.name === "id" || a.name === "role" || a.name.startsWith("data-"))
      .map((a) => `${a.name}=${a.value.slice(0, 30)}`);
    return `<${el.tagName.toLowerCase()} ${attrs.join(" ")} class=${String(el.getAttribute("class")).slice(0, 60)}>`;
  };
  window.ResizeObserver = class extends Native {
    constructor(callback: ResizeObserverCallback) {
      const creator = (new Error().stack ?? "").split("\n").slice(1, 4).join(" < ");
      super((entries, observer) => {
        const sizes = entries.map(
          (e) =>
            `${describe(e.target)} ${Math.round(e.contentRect.width)}x${Math.round(e.contentRect.height)}`,
        );
        recent.push({ frame, line: `frame ${frame} [${creator}] ${sizes.join(", ")}` });
        while ((recent[0]?.frame ?? frame) < frame - 2) {
          recent.shift();
        }
        callback(entries, observer);
      });
    }
  };
  window.addEventListener("error", (event) => {
    if (event.message.includes("ResizeObserver loop")) {
      void window.reportResizeLoop(recent.map((r) => r.line).join(" | "));
    }
  });
}
