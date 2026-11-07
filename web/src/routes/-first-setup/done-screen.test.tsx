import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { addServer, anthologyCatalog, cinemaCatalog } from "../../test/adding";
import { done, firstSetup, stepHeading } from "../../test/first-setup";
import { nodesAre } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { media } from "../../test/roots";
import { zines } from "../../test/roots-page";

const disabledZines = { ...cinemaCatalog(), type: { ...zines, enabled: false } };

function open() {
  const server = firstSetup([done("address"), done("roots")]);
  const app = renderApp({
    path: "/setup/done",
    handlers: [
      ...server.handlers,
      ...addServer({ catalogs: [cinemaCatalog(), anthologyCatalog(), disabledZines] }).handlers,
      nodesAre(media),
    ],
  });
  return { app, server };
}

test("the done screen offers an Add for each enabled Content Type beside Go to Library", async () => {
  open();
  await expect.element(stepHeading("First setup")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Add a Film" })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Add an Anthology" })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Go to Library" })).toBeVisible();
  expect(page.getByRole("button", { name: `Add a ${zines.work.one}` }).query()).toBeNull();
});

test("an Add finishes first setup and opens that type's add page", async () => {
  const { app, server } = open();
  await userEvent.click(page.getByRole("button", { name: "Add an Anthology" }));
  await expect.element(page.getByRole("heading", { name: "Add an Anthology" })).toBeVisible();
  expect(server.finishes).toHaveLength(1);
  expect(app.router.state.location.href).toBe("/add?contentType=anthology");
});
