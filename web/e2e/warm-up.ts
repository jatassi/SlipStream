import { readFile } from "node:fs/promises";
import { type RequestListener, createServer } from "node:http";
import { fileURLToPath } from "node:url";

import { type Browser, errors } from "@playwright/test";
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

async function loadInFreshContext(browser: Browser, url: string, attempts: number): Promise<void> {
  const context = await browser.newContext();
  try {
    await (await context.newPage()).goto(url, { timeout: 5000 });
    return;
  } catch (error) {
    if (!(error instanceof errors.TimeoutError) || attempts === 1) {
      throw error;
    }
  } finally {
    await context.close();
  }
  await loadInFreshContext(browser, url, attempts - 1);
}

// WebKit's first cross-origin-isolated navigation in a fresh browser can hang
// mid process swap, so a throwaway page takes that navigation before any test.
export async function warmUpWebKit(browser: Browser): Promise<void> {
  await serve(
    (_request, response) => {
      response.writeHead(200, {
        "Content-Type": "text/html",
        "Cross-Origin-Opener-Policy": "same-origin",
      });
      response.end("<!doctype html><title>warm-up</title>");
    },
    (url) => loadInFreshContext(browser, url, 3),
  );
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
