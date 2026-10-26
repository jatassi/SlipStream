import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  ceremony,
  iPhonePasskey,
  passkeysAt,
  passkeysHere,
  stubPasskeys,
} from "../../test/account";
import { done, firstSetup, metadataPlace, stepHeading, todo } from "../../test/first-setup";
import { renderApp } from "../../test/render-app";

const registration = [
  http.post("*/api/v1/passkeys/registration", () => ceremony("r-1")),
  http.post("*/api/v1/passkeys/registration/r-1", () =>
    HttpResponse.json({ ...iPhonePasskey, id: "p-new", name: "Chrome on Mac" }),
  ),
];

function open(steps = [done("address"), todo("passkey"), todo("language")]) {
  const server = firstSetup(steps);
  renderApp({
    path: "/setup/passkey",
    handlers: [
      ...server.handlers,
      passkeysAt(passkeysHere),
      ...registration,
      ...metadataPlace().handlers,
    ],
  });
  return server;
}

test("the passkey step adds a passkey and moves on, with the frame's Skip as its only Skip", async () => {
  const { create } = stubPasskeys();
  create.mockResolvedValue({ id: "cred-2", type: "public-key" });
  const server = open();
  await expect.element(stepHeading("Add a passkey")).toBeVisible();
  await expect.element(page.getByText("Step 2 of 3")).toBeVisible();
  await expect.element(page.getByRole("button", { name: /^Skip/u })).toHaveTextContent("Skip");
  await userEvent.click(page.getByRole("button", { name: "Add a passkey" }));
  await expect.element(page.getByText("Added Chrome on Mac.")).toBeVisible();
  await expect.element(stepHeading("Language and region")).toBeVisible();
  expect(server.writes).toEqual([{ id: "passkey", outcome: "done" }]);
});

test("a passkey added elsewhere leaves Continue in place of Add a passkey", async () => {
  stubPasskeys();
  const server = open([done("address"), done("passkey"), todo("language")]);
  await expect.element(page.getByText("Your Account has a passkey.")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Add a passkey" })).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(stepHeading("Language and region")).toBeVisible();
  expect(server.writes).toEqual([]);
});
