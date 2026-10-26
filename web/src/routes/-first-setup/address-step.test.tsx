import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { SetAddressBody } from "../../api/gen/zod/identity/identity.zod";
import { done, firstSetup, metadataPlace, stepHeading, todo } from "../../test/first-setup";
import { renderApp } from "../../test/render-app";

function addressServer(suggestedAddress: string, initial = "") {
  const saved: string[] = [];
  let address = initial;
  return {
    saved,
    handlers: [
      http.get("*/api/v1/identity/address", () => HttpResponse.json({ address, suggestedAddress })),
      http.put("*/api/v1/identity/address", async ({ request }) => {
        address = SetAddressBody.parse(await request.json()).address;
        saved.push(address);
        return HttpResponse.json({ address, suggestedAddress });
      }),
    ],
  };
}

test("the address step suggests the address in use and Continue saves it before moving on", async () => {
  const server = firstSetup([todo("address"), todo("language")]);
  const address = addressServer("https://slipstream.example.com");
  renderApp({
    path: "/setup/address",
    handlers: [...server.handlers, ...address.handlers, ...metadataPlace().handlers],
  });
  await expect.element(stepHeading("Where people reach SlipStream")).toBeVisible();
  await expect
    .element(
      page.getByText(
        "Pick an address everyone in your household can reach, because Signup and Reset links use it.",
      ),
    )
    .toBeVisible();
  const field = page.getByRole("textbox", { name: "Address" });
  await expect.element(field).toHaveValue("https://slipstream.example.com");
  await expect
    .element(field)
    .toHaveAccessibleDescription(
      "Not set yet. SlipStream can't send Signup, Reset or email links until it is.",
    );
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(stepHeading("Language and region")).toBeVisible();
  expect(address.saved).toEqual(["https://slipstream.example.com"]);
  expect(server.writes).toEqual([{ id: "address", outcome: "done" }]);
});

test("the address step warns that a plain-HTTP address can't hold a passkey", async () => {
  const server = firstSetup([todo("address"), todo("language")]);
  const address = addressServer("http://media-box:8080");
  renderApp({
    path: "/setup/address",
    handlers: [...server.handlers, ...address.handlers, ...metadataPlace().handlers],
  });
  await expect
    .element(page.getByRole("textbox", { name: "Address" }))
    .toHaveAccessibleDescription(
      "Passkeys don't work on a plain-HTTP address. Use an https:// address.",
    );
});

test("Skip leaves the address unset", async () => {
  const server = firstSetup([todo("address"), done("language")]);
  const address = addressServer("https://slipstream.example.com");
  renderApp({
    path: "/setup/address",
    handlers: [...server.handlers, ...address.handlers, ...metadataPlace().handlers],
  });
  await userEvent.click(page.getByRole("button", { name: "Skip" }));
  await expect.element(stepHeading("Language and region")).toBeVisible();
  expect(address.saved).toEqual([]);
  expect(server.writes).toEqual([{ id: "address", outcome: "skipped" }]);
});

test("Continue on a done address step moves on without saving the same address again", async () => {
  const server = firstSetup([done("address"), todo("language")]);
  const address = addressServer("http://media-box:8080", "https://slipstream.example.com");
  renderApp({
    path: "/setup/address",
    handlers: [...server.handlers, ...address.handlers, ...metadataPlace().handlers],
  });
  await expect
    .element(page.getByRole("textbox", { name: "Address" }))
    .toHaveValue("https://slipstream.example.com");
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(stepHeading("Language and region")).toBeVisible();
  expect(address.saved).toEqual([]);
  expect(server.writes).toEqual([]);
});

test("a changed address on a done step is saved before moving on", async () => {
  const server = firstSetup([done("address"), todo("language")]);
  const address = addressServer("http://media-box:8080", "https://slipstream.example.com");
  renderApp({
    path: "/setup/address",
    handlers: [...server.handlers, ...address.handlers, ...metadataPlace().handlers],
  });
  const field = page.getByRole("textbox", { name: "Address" });
  await userEvent.fill(field, "https://media.example.com");
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(stepHeading("Language and region")).toBeVisible();
  expect(address.saved).toEqual(["https://media.example.com"]);
  expect(server.writes).toEqual([]);
});
