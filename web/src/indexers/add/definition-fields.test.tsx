import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { addIndexerServer } from "../../test/add-indexer-server";
import { renderApp } from "../../test/render-app";

const formOf = (id: string) => `/settings/indexers/new/catalog/${id}`;
const posted = (writes: ReturnType<typeof addIndexerServer>["writes"]) =>
  writes.filter((write) => write.method === "POST").map((write) => write.body);

test("a definition draws each field type, sealing secrets and keeping info as plain text", async () => {
  const { router } = renderApp({ path: formOf("meadow"), handlers: addIndexerServer().handlers });
  await expect.element(page.getByRole("heading", { name: "Add Meadow", level: 1 })).toBeVisible();
  await expect
    .element(
      page.getByText(
        "Meadow is a private Tracker. SlipStream signs in to it from home and keeps the session there. Its sign-in shows a captcha, which comes after these fields.",
      ),
    )
    .toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "Username" })).toBeVisible();
  await expect
    .element(page.getByLabelText("Password", { exact: true }))
    .toHaveAttribute("type", "password");
  await expect
    .element(page.getByLabelText("Two-factor code", { exact: true }))
    .toHaveAttribute("type", "password");
  await expect.element(page.getByRole("checkbox", { name: "Only freeleech" })).not.toBeChecked();
  await expect
    .element(page.getByRole("combobox", { name: "Sort by" }))
    .toHaveTextContent("Seeders");
  await expect.element(page.getByText("Ratio", { exact: true })).toBeVisible();
  await expect.element(page.getByText("Keep a ratio of 0.8 or more.")).toBeVisible();
  await expect.element(page.getByText("Copy the cookie from your browser.")).toBeVisible();
  await expect.element(page.getByText("Copy your browser's User-Agent.")).toBeVisible();
  await expect
    .element(page.getByText("Category 8000 holds everything the site doesn't sort."))
    .toBeVisible();
  expect(page.getByText("Use FlareSolverr.").query()).toBeNull();
  expect(page.getByText("CAPTCHA").query()).toBeNull();
  await userEvent.click(page.getByRole("link", { name: "Set a Challenge solver on home" }));
  expect(router.state.location.pathname).toBe("/nodes/4a2c6e8f-1b3d-4f5a-9c7e-2d4f6a8b0c1e");
});

test("each field's value is sent by the definition's own name", async () => {
  const server = addIndexerServer();
  renderApp({ path: formOf("meadow"), handlers: server.handlers });
  await userEvent.fill(page.getByRole("textbox", { name: "Username" }), "ada");
  await userEvent.fill(page.getByLabelText("Password", { exact: true }), "hunter2");
  await userEvent.fill(page.getByLabelText("Two-factor code", { exact: true }), "424242");
  await userEvent.click(page.getByRole("checkbox", { name: "Only freeleech" }));
  await userEvent.click(page.getByRole("combobox", { name: "Sort by" }));
  await userEvent.click(page.getByRole("option", { name: "Added" }));
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await userEvent.fill(page.getByRole("textbox", { name: "Code" }), "K7Q2");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("heading", { name: "Meadow", level: 1 })).toBeVisible();
  expect(posted(server.writes)).toEqual([
    expect.objectContaining({
      kind: "catalog",
      definition: "meadow",
      name: "Meadow",
      homeNodeId: "4a2c6e8f-1b3d-4f5a-9c7e-2d4f6a8b0c1e",
      values: {
        username: "ada",
        password: "hunter2",
        "2fa-code": "424242",
        freeleech: "true",
        sort: "added",
      },
      captcha: { attempt: "attempt-1", answer: "K7Q2" },
    }),
  ]);
});

test("a definition with no session says it searches from home; its address and categories wait under Details", async () => {
  const server = addIndexerServer();
  renderApp({ path: formOf("quarry"), handlers: server.handlers });
  await expect
    .element(page.getByText("Quarry is a semi-private Tracker. SlipStream searches it from home."))
    .toBeVisible();
  await expect
    .element(page.getByText("Your API key is on your Quarry profile page."))
    .toBeVisible();
  await expect
    .element(page.getByText("It searches from home at priority 25, with no limits."))
    .toBeVisible();
  await userEvent.fill(page.getByLabelText("API key", { exact: true }), "q-key");
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await userEvent.click(page.getByRole("combobox", { name: "Address" }));
  await userEvent.click(page.getByRole("option", { name: "https://quarry.example.net/" }));
  await userEvent.click(page.getByRole("checkbox", { name: "TV" }));
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("heading", { name: "Quarry", level: 1 })).toBeVisible();
  const [body] = posted(server.writes);
  expect(body).toMatchObject({
    baseAddress: "https://quarry.example.net/",
    categories: [2000],
    values: { apikey: "q-key" },
  });
  expect(body).not.toHaveProperty("captcha");
});

test("a failed Test shows Go's sentence above the buttons, and nothing is added", async () => {
  const sentence =
    "Quarry refused the API key with 'Invalid API Key'. Check the account, then test again.";
  const { router } = renderApp({
    path: formOf("quarry"),
    handlers: addIndexerServer({ failSaves: [sentence] }).handlers,
  });
  await userEvent.fill(page.getByLabelText("API key", { exact: true }), "wrong");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert").getByText(sentence)).toBeVisible();
  expect(router.state.location.pathname).toBe(formOf("quarry"));
});

test("a cookie definition asks for the cookie and User-Agent and sends them together", async () => {
  const server = addIndexerServer();
  renderApp({ path: formOf("cinder"), handlers: server.handlers });
  await expect
    .element(page.getByText("Cinder is behind a browser challenge.", { exact: false }))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(page.getByText("Paste the cookie of a browser signed in to the Tracker."))
    .toBeVisible();
  expect(posted(server.writes)).toEqual([]);
  await userEvent.fill(page.getByRole("textbox", { name: "Cookie" }), "cf_clearance=abc; uid=7");
  await userEvent.fill(page.getByRole("textbox", { name: "User-Agent" }), "Mozilla/5.0 Test");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("heading", { name: "Cinder", level: 1 })).toBeVisible();
  expect(posted(server.writes)).toEqual([
    expect.objectContaining({
      values: {},
      cookie: { cookie: "cf_clearance=abc; uid=7", userAgent: "Mozilla/5.0 Test" },
    }),
  ]);
});
