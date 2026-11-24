import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { CreateSessionBody } from "../api/gen/zod/identity/identity.zod";
import { readPref, writePref } from "../shell/prefs";
import { ada, problem, signedOut } from "../test/handlers";
import { renderApp } from "../test/render-app";
import { whenWords } from "../ui/time-words";

const own = () => ({ now: new Date(), zone: undefined });

function adaServer() {
  let signedIn = false;
  return [
    http.get("*/api/v1/session", () =>
      signedIn
        ? HttpResponse.json({ account: ada, pinRequired: false, keyPrompt: "ask" })
        : problem(401, "signed_out", "Sign in to continue"),
    ),
    http.post("*/api/v1/session", async ({ request }) => {
      const { pin } = CreateSessionBody.parse(await request.json());
      if (pin !== "1234") {
        return problem(401, "pin_wrong", "The email or PIN is wrong");
      }
      signedIn = true;
      return HttpResponse.json({ account: ada, pinRequired: false, keyPrompt: "ask" });
    }),
  ];
}

async function typePin(pin: string) {
  await userEvent.click(page.getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard(pin);
}

test("a wrong PIN shows the server's title and clears the PIN, and the right one signs in", async () => {
  const { router } = renderApp({ path: "/library", handlers: adaServer() });
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
  await expect.element(page.getByRole("status")).toMatchTextContent(/session ended/u);
});

test("the browser remembers the email that signed in and fills it next time", async () => {
  renderApp({ path: "/library", handlers: adaServer() });
  await userEvent.fill(page.getByLabelText("Email"), " ada@example.test ");
  await typePin("1234");
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  expect(readPref("lastEmail")).toBe("ada@example.test");
});

test("a remembered email is filled in", async () => {
  writePref("lastEmail", "grace@example.test");
  renderApp({ path: "/sign-in", handlers: [signedOut] });
  await expect.element(page.getByLabelText("Email")).toHaveValue("grace@example.test");
});

test.each([
  [
    "too many failures from this address",
    () =>
      HttpResponse.json(
        {
          status: 429,
          code: "too_many_attempts",
          title: "Too many failed sign-ins from this IP address",
          detail: "Try again at 15:42.",
          until: "2026-10-16T15:42:00Z",
        },
        { status: 429, headers: { "Content-Type": "application/problem+json" } },
      ),
    `Too many failed sign-ins from this IP address. Try again ${whenWords("2026-10-16T15:42:00Z", "next", own())}.`,
  ],
])("%s says so", async (_case, answer, alert) => {
  renderApp({
    path: "/sign-in",
    handlers: [signedOut, http.post("*/api/v1/session", answer)],
  });
  await userEvent.fill(page.getByLabelText("Email"), "ada@example.test");
  await typePin("1234");
  await expect.element(page.getByRole("alert")).toHaveTextContent(alert);
});
