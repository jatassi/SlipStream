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

const fontPage = `<!doctype html>
<title>warm-up</title>
<style>
  @font-face { font-family: "Geist Variable"; src: url(/sans.woff2) format("woff2"); font-weight: 100 900; }
  @font-face { font-family: "Geist Mono Variable"; src: url(/mono.woff2) format("woff2"); font-weight: 100 900; }
  body { font-family: "Geist Variable", "Helvetica Neue", sans-serif; }
  code { font-family: "Geist Mono Variable", ui-monospace, monospace; }
</style>
<p>Set up SlipStream</p>
<code>MPN9-CCVY-BKTX</code>`;

// A fresh Firefox on a Windows runner froze for 10 to 15 seconds at the
// first text it laid out in the app's font stacks, so a throwaway page lays
// out text in them before any test does.
export async function warmUpFirefox(browser: Browser): Promise<void> {
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
        response.end(fontPage);
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
