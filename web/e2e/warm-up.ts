import { createServer } from "node:http";

import { type Browser, errors } from "@playwright/test";
import { z } from "zod";

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
  const server = createServer((_request, response) => {
    response.writeHead(200, {
      "Content-Type": "text/html",
      "Cross-Origin-Opener-Policy": "same-origin",
    });
    response.end("<!doctype html><title>warm-up</title>");
  });
  await new Promise<void>((resolve) => {
    server.listen(0, "127.0.0.1", resolve);
  });
  const { port } = z.object({ port: z.number() }).parse(server.address());
  try {
    await loadInFreshContext(browser, `http://127.0.0.1:${port}/`, 3);
  } finally {
    server.close();
  }
}
