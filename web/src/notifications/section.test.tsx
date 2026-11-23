import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { events, failing, family, hook, recommendedEvents } from "../test/notifications";
import { notificationsWorld } from "../test/notification-writes";
import { renderApp } from "../test/render-app";
import { whenWords } from "../ui/time-words";

const path = "/settings/notifications";

const recommendedChecks = {
  Grabbed: "false",
  "Grab failed": "true",
  Imported: "true",
  Upgraded: "true",
  "Works added": "false",
  "Works added by a Library scan": "false",
  "Works deleted": "false",
  "New Health item": "true",
  "Health item cleared": "true",
};
const dialog = () => page.getByRole("dialog");

test("with no connections the section says what one is for", async () => {
  const world = notificationsWorld({ start: [] });
  renderApp({ path, handlers: world.handlers });
  await expect
    .element(
      page.getByText(
        "Add a connection to get a message when SlipStream imports something or a Grab fails.",
      ),
    )
    .toBeVisible();
});

test("a healthy connection's section counts it and its row says nothing was sent yet", async () => {
  const world = notificationsWorld({ start: [family] });
  renderApp({ path, handlers: world.handlers });
  await expect.element(page.getByText("1 connection sends notifications.")).toBeVisible();
  const row = page.getByRole("listitem", { name: "Family" });
  await expect.element(row.getByText("Nothing sent yet.")).toBeVisible();
});

test("a failing connection's section names it with the time it started failing", async () => {
  const since = new Date(Date.now() - 60_000).toISOString();
  const at = whenWords(since, "since", { now: new Date(), zone: undefined });
  const world = notificationsWorld({ start: [failing(family, since), hook] });
  renderApp({ path, handlers: world.handlers });
  await expect
    .element(
      page.getByText(
        `2 connections send notifications. Messages to Discord 'Family' have failed ${at}.`,
      ),
    )
    .toBeVisible();
  await expect
    .element(
      page
        .getByRole("listitem", { name: "Family" })
        .getByText(`Failing ${at}. Discord didn't respond. SlipStream retries automatically.`),
    )
    .toBeVisible();
});

test("a connection that gave messages up says how many and why", async () => {
  const world = notificationsWorld({
    start: [{ ...family, status: { ...family.status, givenUp: 1200 } }],
  });
  renderApp({ path, handlers: world.handlers });
  await expect
    .element(
      page.getByText(
        "1,200 messages weren't sent, because they waited too long or Discord refused them.",
      ),
    )
    .toBeVisible();
});

test("the provider cards are one radio group the arrow keys move through", async () => {
  renderApp({ path, handlers: notificationsWorld({ start: [] }).handlers });
  await userEvent.click(page.getByRole("button", { name: "Add a connection" }));
  const group = dialog().getByRole("radiogroup", { name: "Provider" });
  await expect.element(group.getByRole("radio", { name: "Discord" })).toBeChecked();
  await userEvent.click(group.getByRole("radio", { name: "Discord" }));
  await userEvent.keyboard("{ArrowRight}");
  await expect.element(group.getByRole("radio", { name: "Webhook" })).toBeChecked();
  await expect.element(group.getByRole("radio", { name: "Webhook" })).toHaveFocus();
  await userEvent.click(dialog().getByRole("button", { name: "Next" }));
  await expect.element(page.getByRole("dialog", { name: "Add Webhook" })).toBeVisible();
});

test("a new connection starts with the recommended events and sends only the typed secret", async () => {
  const world = notificationsWorld({ start: [] });
  renderApp({ path, handlers: world.handlers });
  await userEvent.click(page.getByRole("button", { name: "Add a connection" }));
  await userEvent.click(dialog().getByRole("radio", { name: "Discord" }));
  await userEvent.click(dialog().getByRole("button", { name: "Next" }));
  await expect.element(page.getByRole("dialog", { name: "Add Discord" })).toBeVisible();
  await expect.element(dialog().getByRole("checkbox", { name: "Grab failed" })).toBeChecked();
  const checked = Object.fromEntries(
    events
      .filter(({ fixed }) => !fixed)
      .map(({ label }) => [
        label,
        dialog()
          .getByRole("checkbox", { name: label, exact: true })
          .element()
          .getAttribute("aria-checked"),
      ]),
  );
  expect(checked).toEqual(recommendedChecks);
  expect(dialog().getByRole("checkbox", { name: "Test" }).query()).toBeNull();

  await userEvent.fill(
    dialog().getByLabelText("Webhook URL"),
    "https://discord.com/api/webhooks/1/abc",
  );
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Added Discord")).toBeVisible();
  expect(world.writes).toEqual([
    {
      path: "add",
      body: {
        provider: "discord",
        name: "Discord",
        on: true,
        events: recommendedEvents,
        warnings: false,
        posters: true,
        values: {},
        secrets: { webhookUrl: "https://discord.com/api/webhooks/1/abc" },
      },
    },
  ]);
});

test("a Test that passes names the Node that sent the message", async () => {
  const world = notificationsWorld({ start: [family] });
  renderApp({ path, handlers: world.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit Discord 'Family'" }));
  await userEvent.click(dialog().getByRole("button", { name: "Test" }));
  await expect.element(dialog().getByText("The Test passed. home sent the message.")).toBeVisible();
  expect(world.writes).toEqual([{ path: `test ${family.id}`, body: {} }]);
});

test("a Test that fails says why and marks the field it names", async () => {
  const world = notificationsWorld({
    start: [family],
    testFails: {
      detail:
        "Discord responded that this address no longer exists. Edit the connection with a new one.",
      field: "webhookUrl",
    },
  });
  renderApp({ path, handlers: world.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit Discord 'Family'" }));
  await userEvent.click(dialog().getByRole("button", { name: "Test" }));
  await expect
    .element(dialog().getByRole("alert"))
    .toMatchTextContent(/Test failed.+Discord.+no longer exists/u);
  await expect
    .element(
      dialog()
        .getByRole("group", { name: "Webhook URL" })
        .getByText("Change this, then test again."),
    )
    .toBeVisible();
});

test("saved secrets read Saved with Replace, and an optional one can be removed", async () => {
  const world = notificationsWorld({ start: [hook] });
  renderApp({ path, handlers: world.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit Webhook 'Home'" }));
  const url = dialog().getByRole("group", { name: "URL", exact: true });
  await expect.element(url.getByText("Saved")).toBeVisible();
  await expect.element(url.getByText("hooks.lan")).toBeVisible();
  await expect.element(url.getByRole("button", { name: "Replace URL" })).toBeVisible();
  expect(url.getByRole("button", { name: "Remove URL" }).query()).toBeNull();

  await userEvent.click(dialog().getByRole("button", { name: "Details" }));
  await userEvent.click(dialog().getByRole("button", { name: "Remove Password" }));
  await expect.element(dialog().getByText("Removed. Save to apply.")).toBeVisible();
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved Home")).toBeVisible();
  expect(world.writes).toEqual([
    { path: `edit ${hook.id}`, body: { secrets: { password: null } } },
  ]);
});

test("the settings are the Health wait and the two Delivery times, with no Keep", async () => {
  renderApp({ path, handlers: notificationsWorld().handlers });
  await expect.element(page.getByText("Wait before sending a new Health item")).toBeVisible();
  await expect.element(page.getByText("Give up on a message after")).toBeVisible();
  await expect.element(page.getByText("Warn when a connection has failed for")).toBeVisible();
  await expect.element(page.getByText(/^Keep /u)).not.toBeInTheDocument();
});
