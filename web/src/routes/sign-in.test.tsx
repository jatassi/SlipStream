import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { CreateSessionBody } from "../api/gen/zod/identity/identity.zod";
import { ada, problem, signedOut } from "../test/handlers";
import { renderApp } from "../test/render-app";

function adaServer() {
  let signedIn = false;
  return [
    http.get("*/api/v1/session", () =>
      signedIn
        ? HttpResponse.json({ account: ada })
        : problem(401, "signed_out", "Sign in to continue"),
    ),
    http.post("*/api/v1/session", async ({ request }) => {
      const { pin } = CreateSessionBody.parse(await request.json());
      if (pin !== "1234") {
        return problem(401, "pin_wrong", "The email or PIN is wrong");
      }
      signedIn = true;
      return HttpResponse.json({ account: ada });
    }),
  ];
}

async function typePin(pin: string) {
  await userEvent.click(page.getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard(pin);
}

test("a signed-out visit to / lands on sign-in with a redirect back", async () => {
  const { router } = renderApp({ path: "/", handlers: [signedOut] });
  await expect.element(page.getByRole("heading", { name: "Sign in to SlipStream" })).toBeVisible();
  expect(router.state.location.href).toBe("/sign-in?redirect=%2F");
});

test("a wrong PIN shows the server's title and clears the PIN, and the right one signs in", async () => {
  const { router } = renderApp({ path: "/", handlers: adaServer() });
  await userEvent.fill(page.getByLabelText("Email"), "ada@example.test");
  await typePin("1235");
  await expect.element(page.getByRole("alert")).toBeVisible();
  expect(page.getByRole("alert").element().textContent).toBe("The email or PIN is wrong");
  await expect.element(page.getByRole("textbox", { name: "PIN", exact: true })).toHaveValue("");

  await typePin("1234");
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  expect(router.state.location.href).toBe("/library");
});

test("an expired session says so", async () => {
  renderApp({ path: "/sign-in?reason=expired", handlers: [signedOut] });
  await expect
    .element(page.getByRole("status"))
    .toHaveTextContent("Your session ended. Sign in again.");
});
