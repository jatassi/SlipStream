import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getGetStatusMockHandler } from "../api/gen/query/system/system.msw";
import { ada, noPasskeys, problem, signedOut } from "../test/handlers";
import { renderApp } from "../test/render-app";

const unclaimed = [
  signedOut,
  getGetStatusMockHandler({ claim: "unclaimed", passkeys: noPasskeys }),
];
const apiKey = `ssk_${"A".repeat(43)}`;

function claimServer(answer: () => Response) {
  const claims: unknown[] = [];
  let claimed = false;
  const handlers = [
    http.get("*/api/v1/session", () =>
      claimed
        ? HttpResponse.json({ account: ada, pinRequired: false, keyPrompt: "ask" })
        : problem(401, "signed_out", "Sign in to continue"),
    ),
    http.get("*/api/v1/status", () =>
      HttpResponse.json({ claim: claimed ? "controller" : "unclaimed" }),
    ),
    http.post("*/api/v1/setup/controller", async ({ request }) => {
      claims.push({ code: request.headers.get("X-Setup-Code"), body: await request.json() });
      const response = answer();
      claimed = response.status === 201;
      return response;
    }),
  ];
  return { claims, handlers };
}

async function fillSetup(pinAgain: string) {
  await userEvent.type(page.getByLabelText("Setup code"), "7k2m q9xd 4hbt");
  await userEvent.fill(page.getByLabelText("Email"), "ada@example.test");
  await userEvent.fill(page.getByLabelText("Display name"), "Ada Okafor");
  await userEvent.click(page.getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard("1234");
  await userEvent.click(page.getByRole("textbox", { name: "PIN again", exact: true }));
  await userEvent.keyboard(pinAgain);
  await userEvent.click(page.getByRole("button", { name: "Create Admin" }));
}

test("an Unclaimed Node sends / to setup", async () => {
  const { router } = renderApp({ path: "/", handlers: unclaimed });
  await expect.element(page.getByRole("heading", { name: "Set up SlipStream" })).toBeVisible();
  expect(router.state.location.href).toBe("/setup");
});

test("the Setup code shows grouped as it is typed", async () => {
  renderApp({ path: "/setup", handlers: unclaimed });
  await userEvent.type(page.getByLabelText("Setup code"), "7k2m q9xd 4hbt");
  await expect.element(page.getByLabelText("Setup code")).toHaveValue("7K2M-Q9XD-4HBT");
});

test("a setup link fills in the Setup code and leaves the address bar", async () => {
  const server = claimServer(() => HttpResponse.json({ account: ada, apiKey }, { status: 201 }));
  const { router } = renderApp({ path: "/setup#code=7K2M-Q9XD-4HBT", handlers: server.handlers });
  await expect.element(page.getByLabelText("Setup code")).toHaveValue("7K2M-Q9XD-4HBT");
  await expect
    .element(page.getByLabelText("Setup code"))
    .toHaveAccessibleDescription("Setup code filled in");
  await expect.poll(() => router.state.location.href).toBe("/setup");
  expect(router.history.location.hash).toBe("");

  await userEvent.fill(page.getByLabelText("Email"), "ada@example.test");
  await userEvent.fill(page.getByLabelText("Display name"), "Ada Okafor");
  await userEvent.click(page.getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard("1234");
  await userEvent.click(page.getByRole("textbox", { name: "PIN again", exact: true }));
  await userEvent.keyboard("1234");
  await userEvent.click(page.getByRole("button", { name: "Create Admin" }));
  await expect.element(page.getByLabelText("API key")).toHaveValue(apiKey);
  expect(server.claims).toEqual([
    {
      code: "7K2MQ9XD4HBT",
      body: { email: "ada@example.test", displayName: "Ada Okafor", pin: "1234" },
    },
  ]);
});

test("a setup link's code reads the way a typed one does", async () => {
  renderApp({ path: "/setup#code=7k2m%20q9xd%204hbt", handlers: unclaimed });
  await expect.element(page.getByLabelText("Setup code")).toHaveValue("7K2M-Q9XD-4HBT");
});

test("a typed Setup code says nothing was filled in", async () => {
  renderApp({ path: "/setup", handlers: unclaimed });
  await expect.element(page.getByLabelText("Setup code")).toHaveAccessibleDescription("");
});

test("mismatched PINs never reach the server", async () => {
  const server = claimServer(() => HttpResponse.json({ account: ada, apiKey }, { status: 201 }));
  renderApp({ path: "/setup", handlers: server.handlers });
  await fillSetup("4321");
  await expect.element(page.getByText("The PINs don't match")).toBeVisible();
  expect(server.claims).toEqual([]);
});

test("a claim shows the API key once and Continue enters the app", async () => {
  const server = claimServer(() => HttpResponse.json({ account: ada, apiKey }, { status: 201 }));
  const { router } = renderApp({ path: "/setup", handlers: server.handlers });
  await fillSetup("1234");
  await expect.element(page.getByLabelText("API key")).toHaveValue(apiKey);
  expect(server.claims).toEqual([
    {
      code: "7K2MQ9XD4HBT",
      body: { email: "ada@example.test", displayName: "Ada Okafor", pin: "1234" },
    },
  ]);

  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  expect(router.state.location.href).toBe("/library");
});

test("a wrong Setup code shows its title under the Setup code field", async () => {
  const server = claimServer(() => problem(403, "setup_code_wrong", "The Setup code is wrong"));
  renderApp({ path: "/setup", handlers: server.handlers });
  await fillSetup("1234");
  await expect
    .element(page.getByLabelText("Setup code"))
    .toHaveAccessibleDescription("The Setup code is wrong");
});
