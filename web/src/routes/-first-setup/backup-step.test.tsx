import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { done, firstSetup, metadataPlace, stepHeading, todo } from "../../test/first-setup";
import { renderApp } from "../../test/render-app";

const words = "acid bench cobra dune ember fjord gecko haven iris jolt";

test("the backup step shows the passphrase and moves on once it's saved", async () => {
  const server = firstSetup([done("address"), done("passkey"), todo("backup"), todo("language")]);
  const saved: string[] = [];
  renderApp({
    path: "/setup/backup",
    handlers: [
      ...server.handlers,
      ...metadataPlace().handlers,
      http.get("*/api/v1/backups/passphrase", () => HttpResponse.json({ passphrase: words })),
      http.post("*/api/v1/backups/passphrase/saved", () => {
        saved.push("saved");
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  });
  await expect.element(stepHeading("Save your Backup passphrase")).toBeVisible();
  await expect.element(page.getByText(words)).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "I saved it" }));
  await expect.element(stepHeading("Language and region")).toBeVisible();
  expect(saved).toEqual(["saved"]);
  expect(server.writes).toEqual([{ id: "backup", outcome: "done" }]);
});
