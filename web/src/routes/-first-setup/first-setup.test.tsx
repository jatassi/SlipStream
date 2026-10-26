import { type HttpHandler, HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../../api/gen/query/works/works.msw";
import { done, firstSetup, skipped, stepHeading, todo } from "../../test/first-setup";
import { cinema } from "../../test/library";
import { nodesAre } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { zines } from "../../test/roots-page";
import { media } from "../../test/roots";

const world = [
  getListContentTypesMockHandler({ contentTypes: [cinema, zines] }),
  nodesAre(media),
  http.get("*/api/v1/roots", () => HttpResponse.json({ roots: [] })),
];

function open(
  path: string,
  steps = [done("address"), todo("media"), todo("roots")],
  handlers: readonly HttpHandler[] = [],
) {
  const server = firstSetup(steps);
  const app = renderApp({ path, handlers: [...handlers, ...server.handlers, ...world] });
  return { app, server };
}

const pathname = (app: ReturnType<typeof renderApp>) => app.router.state.location.pathname;

test("an app URL brings an Admin whose first setup is open back to its current step", async () => {
  const { app } = open("/library");
  await expect.element(stepHeading("Kinds of media")).toBeVisible();
  expect(pathname(app)).toBe("/setup/media");
  await expect.element(page.getByText("Step 2 of 3")).toBeVisible();
  await expect.element(page.getByText("After this: Library roots")).toBeVisible();
  await expect
    .element(page.getByText("SlipStream shows only the kinds of media you turn on."))
    .toBeVisible();
});

test("a step the browser hides, or no step at all, holds the Admin on the current one", async () => {
  const { app } = open("/setup/passkey", [done("address"), todo("passkey"), todo("media")]);
  await expect.element(stepHeading("Kinds of media")).toBeVisible();
  expect(pathname(app)).toBe("/setup/media");
  await expect.element(page.getByText("Step 2 of 2")).toBeVisible();
});

test("Skip stores the step skipped and moves on, and Back shows what an earlier step holds", async () => {
  const { app, server } = open("/setup/media");
  await userEvent.click(page.getByRole("button", { name: "Skip" }));
  await expect.element(stepHeading("Library roots")).toBeVisible();
  expect(server.writes).toEqual([{ id: "media", outcome: "skipped" }]);
  expect(pathname(app)).toBe("/setup/roots");

  await userEvent.click(page.getByRole("button", { name: "Back" }));
  await expect.element(stepHeading("Kinds of media")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(stepHeading("Library roots")).toBeVisible();
  expect(server.writes).toEqual([
    { id: "media", outcome: "skipped" },
    { id: "media", outcome: "done" },
  ]);
});

test("Continue on a step that already reads done moves on without a write", async () => {
  const { server } = open("/setup/media", [done("media"), todo("roots")]);
  await userEvent.click(page.getByRole("button", { name: "Continue" }));
  await expect.element(stepHeading("Library roots")).toBeVisible();
  expect(server.writes).toEqual([]);
});

test("Finish setup on any step lands in the Library, and setup never comes back", async () => {
  const { app, server } = open("/setup/media");
  await userEvent.click(page.getByRole("button", { name: "Finish setup" }));
  await expect.element(page.getByRole("heading", { name: "Library", level: 1 })).toBeVisible();
  expect(server.finishes).toHaveLength(1);
  await app.router.navigate({ to: "/setup/$step", params: { step: "roots" } });
  await expect.poll(() => pathname(app)).toBe("/library");
});

test("the done screen lists each step as set or skipped, and Go to Library finishes", async () => {
  const { app, server } = open("/library", [done("address"), skipped("media"), done("roots")]);
  await expect.element(stepHeading("First setup")).toBeVisible();
  expect(pathname(app)).toBe("/setup/done");
  const summary = page.getByRole("list", { name: "First setup" });
  await expect
    .element(summary.getByRole("listitem", { name: "Address" }))
    .toHaveTextContent("AddressSet");
  await expect
    .element(summary.getByRole("listitem", { name: "Kinds of media" }))
    .toHaveTextContent("Kinds of mediaSet it in SettingsSkipped");
  await expect
    .element(summary.getByRole("listitem", { name: "Library roots" }).getByRole("link"))
    .not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Go to Library" }));
  await expect.element(page.getByRole("heading", { name: "Library", level: 1 })).toBeVisible();
  expect(server.finishes).toHaveLength(1);
});

test("a skipped step's link finishes first setup and opens its place in Settings", async () => {
  const { app, server } = open("/setup/done", [done("address"), skipped("media"), done("roots")]);
  await userEvent.click(page.getByRole("link", { name: "Set it in Settings" }));
  await expect.poll(() => pathname(app)).toBe("/settings/library-roots");
  expect(server.finishes).toHaveLength(1);
});

test("with email not set up, the done screen links its settings, and the link finishes first setup", async () => {
  const unset = http.get("*/api/v1/mail/settings", () =>
    HttpResponse.json({
      fromAddress: "",
      fromName: "",
      host: "",
      passwordSet: false,
      port: 587,
      security: "starttls",
      username: "",
    }),
  );
  const { app, server } = open("/setup/done", [done("address"), done("roots")], [unset]);
  await expect.element(page.getByText("Email isn't set up.")).toBeVisible();
  await userEvent.click(page.getByRole("link", { name: "Set it in Settings" }));
  await expect.poll(() => pathname(app)).toBe("/settings/email");
  expect(server.finishes).toHaveLength(1);
});

test("with email set up, the done screen leaves it out", async () => {
  open("/setup/done", [done("address"), done("roots")]);
  await expect.element(stepHeading("First setup")).toBeVisible();
  await expect.element(page.getByText("Email isn't set up.")).not.toBeInTheDocument();
});

test("a tab follows another browser's finish into the Library", async () => {
  const { app, server } = open("/setup/media");
  await expect.element(stepHeading("Kinds of media")).toBeVisible();
  server.state.open = false;
  server.send({ event: "firstrun.changed", data: {} });
  await expect.poll(() => pathname(app)).toBe("/library");
  expect(server.finishes).toHaveLength(0);
});
