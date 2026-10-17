import { z } from "zod";

import { node } from "../harness/manifest.ts";
import { submitPin } from "../harness/sign-in.ts";
import { addVirtualAuthenticator } from "./authenticator.ts";
import { expect, test } from "./fixtures.ts";
import { signOut } from "./shell.ts";

test.use({ scenario: "serving", at: "seedbox" });

// The first-PR fact "passkey origin": the browser makes and uses a passkey
// at seedbox's address while home answers every request through it, so
// home verifies the ceremony against the origin the browser saw.
test(
  "a passkey added at a Serving Worker's address signs in there",
  { tag: "@cdp" },
  async ({ newContext, world }) => {
    const context = await newContext();
    const page = await context.newPage();
    await addVirtualAuthenticator(context, page);
    await page.goto("/sign-in");
    const signedIn = await submitPin(page, world.admin().email, world.admin().pin);
    expect(signedIn.status()).toBe(200);

    const prompt = page.getByRole("dialog", { name: "Sign in faster here with a passkey" });
    const registered = page.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        /\/api\/v1\/passkeys\/registration\/[^/]+$/u.test(new URL(response.url()).pathname),
    );
    await prompt.getByRole("button", { name: "Add a passkey" }).click();
    expect((await registered).status()).toBe(201);
    await expect(prompt).toBeHidden();

    const asserted = page.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        /\/api\/v1\/passkeys\/assertion\/[^/]+$/u.test(new URL(response.url()).pathname),
    );
    await signOut(page);
    expect((await asserted).status()).toBe(200);
    await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
    expect(new URL(page.url()).origin).toBe(new URL(world.baseURL).origin);
    const system = await page.evaluate(async () => {
      const response = await fetch("/api/v1/system");
      return (await response.json()) as unknown;
    });
    expect(
      z.object({ dataDir: z.string() }).parse(system).dataDir,
      "the Node answering seedbox's address",
    ).toBe(node(world.manifest, "home").dataDir);
  },
);
