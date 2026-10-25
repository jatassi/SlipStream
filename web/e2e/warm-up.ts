import { readFile } from "node:fs/promises";
import { type RequestListener, createServer } from "node:http";
import { fileURLToPath } from "node:url";

import { type Browser, type BrowserContext, type Page, errors, test } from "@playwright/test";
import { z } from "zod";

async function serve(
  listener: RequestListener,
  use: (url: string) => Promise<void>,
): Promise<void> {
  const server = createServer(listener);
  await new Promise<void>((resolve) => {
    server.listen(0, "127.0.0.1", resolve);
  });
  const { port } = z.object({ port: z.number() }).parse(server.address());
  try {
    await use(`http://127.0.0.1:${port}/`);
  } finally {
    server.close();
  }
}

// WebKit's first navigation of a page into a cross-origin-isolated origin can
// hang mid process swap, in a fresh browser or a warm one.
export function overrideNewPageWithWarmUp(context: BrowserContext, sameOriginUrl: string): void {
  const newPage = context.newPage.bind(context);
  const open = async (attempts: number): Promise<Page> => {
    const page = await newPage();
    try {
      await page.goto(sameOriginUrl, { timeout: 5000 });
      return page;
    } catch (error) {
      await page.close();
      if (!(error instanceof errors.TimeoutError) || attempts === 1) {
        throw error;
      }
    }
    test
      .info()
      .annotations.push({ type: "webkit warm-up", description: `${sameOriginUrl} timed out` });
    return open(attempts - 1);
  };
  context.newPage = () => open(3);
}

const fonts = {
  "/sans.woff2": "@fontsource-variable/geist/files/geist-latin-wght-normal.woff2",
  "/mono.woff2": "@fontsource-variable/geist-mono/files/geist-mono-latin-wght-normal.woff2",
};

const warmUpPage = `<!doctype html>
<title>warm-up</title>
<style>
  @font-face { font-family: "Geist Variable"; src: url(/sans.woff2) format("woff2"); font-weight: 100 900; }
  @font-face { font-family: "Geist Mono Variable"; src: url(/mono.woff2) format("woff2"); font-weight: 100 900; }
  body { font-family: "Geist Variable", "Helvetica Neue", sans-serif; background: linear-gradient(#202a3a, #0d0f12); color: #fff; }
  code { font-family: "Geist Mono Variable", ui-monospace, monospace; }
  .backdrop { position: fixed; inset: -8rem; z-index: -1; background: linear-gradient(135deg, #e0603a, #3a6ee0); filter: blur(60px) saturate(1.4) brightness(.6); }
  .panel {
    padding: 24px; border-radius: 28px; background: rgba(40, 40, 46, .6);
    box-shadow: inset 0 1px 0 rgba(255,255,255,.1), inset 0 0 0 1px rgba(255,255,255,.08), 0 30px 60px rgba(0,0,0,.6);
    backdrop-filter: blur(30px) saturate(1.7);
    transition: scale .2s, opacity .2s;
  }
  @starting-style { .panel { scale: .96; opacity: 0; } }
</style>
<div class="backdrop"></div>
<div class="panel">
  <p>Set up SlipStream</p>
  <code>MPN9-CCVY-BKTX</code>
</div>`;

// A fresh browser pays its first page's startup inside whichever test opens
// that page, so each worker's browser lays out and composites a throwaway
// page in the app's fonts and effects before any test runs. A fresh Firefox
// on a Windows runner froze for 10 to 15 seconds at the first text it laid
// out in the app's font stacks. A fresh WebKit on a Linux runner took 7 to 12
// seconds to open its first page, then seconds more to draw its first frames
// of blurred and shadowed layers while Mesa compiled their shaders.
export async function warmUp(browser: Browser): Promise<void> {
  const files = new Map(
    await Promise.all(
      Object.entries(fonts).map(
        async ([path, specifier]) =>
          [path, await readFile(fileURLToPath(import.meta.resolve(specifier)))] as const,
      ),
    ),
  );
  await serve(
    (request, response) => {
      const font = files.get(request.url ?? "");
      if (font === undefined) {
        response.writeHead(200, { "Content-Type": "text/html" });
        response.end(warmUpPage);
        return;
      }
      response.writeHead(200, { "Content-Type": "font/woff2" });
      response.end(font);
    },
    async (url) => {
      const context = await browser.newContext();
      try {
        const page = await context.newPage();
        await page.goto(url);
        await page.evaluate(async () => {
          await Promise.all([
            document.fonts.load('16px "Geist Variable"'),
            document.fonts.load('16px "Geist Mono Variable"'),
          ]);
          await new Promise<void>((resolve) => {
            requestAnimationFrame(() => {
              requestAnimationFrame(() => {
                resolve();
              });
            });
          });
        });
      } finally {
        await context.close();
      }
    },
  );
}
