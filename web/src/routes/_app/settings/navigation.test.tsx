import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MailSettings } from "../../../api/gen/model";
import { getGetAddressMockHandler } from "../../../api/gen/query/identity/identity.msw";
import { getGetMailSettingsMockHandler } from "../../../api/gen/query/mail/mail.msw";
import { getListContentTypesMockHandler } from "../../../api/gen/query/works/works.msw";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";
import { settingsGroupLabels } from "../../../test/settings";

const mail: MailSettings = {
  host: "smtp.example.test",
  port: 587,
  security: "starttls",
  username: "slipstream",
  passwordSet: true,
  fromAddress: "slipstream@example.test",
  fromName: "SlipStream",
};

const server = () => page.getByRole("textbox", { name: "Server", exact: true });

test("/settings opens on Content Types, with Settings' sections listed under their groups", async () => {
  const { router } = renderApp({
    path: "/settings",
    handlers: [getListContentTypesMockHandler({ contentTypes: [] })],
  });
  await expect.poll(() => router.state.location.pathname).toBe("/settings/content-types");
  const nav = page.getByRole("navigation", { name: "Settings" });
  await expect
    .element(nav.getByRole("link", { name: "Content Types" }))
    .toHaveAttribute("aria-current", "page");
  const groups = Object.entries(settingsGroupLabels);
  expect(nav.getByRole("list").all()).toHaveLength(groups.length);
  await Promise.all(
    groups.map(async ([group, sections], index) => {
      const list = nav.getByRole("list").nth(index);
      await expect.element(list).toHaveAccessibleName(group);
      await expect.element(nav.getByText(group, { exact: true })).toBeVisible();
      expect(
        list
          .getByRole("link")
          .all()
          .map((link) => link.element().textContent),
      ).toEqual(sections);
    }),
  );
});

test("a link in the Settings nav opens its section", async () => {
  const { router } = renderApp({
    path: "/settings/address",
    handlers: [
      getGetAddressMockHandler({ address: "", suggestedAddress: "http://media-pc:8484" }),
      getGetMailSettingsMockHandler(mail),
    ],
  });
  await userEvent.click(
    page.getByRole("navigation", { name: "Settings" }).getByRole("link", { name: "Email" }),
  );
  await expect.poll(() => router.state.location.pathname).toBe("/settings/email");
  await expect.element(server()).toHaveValue("smtp.example.test");
});

test("on phone, Settings lists its sections by group, and each pushes with a back link to Settings", async () => {
  await asPhone();
  const { router } = renderApp({
    path: "/settings",
    handlers: [getGetMailSettingsMockHandler(mail)],
  });
  await expect.element(page.getByRole("heading", { name: "Settings", level: 1 })).toBeVisible();
  await expect.element(page.getByRole("navigation", { name: "Settings" })).not.toBeInTheDocument();
  expect(
    page
      .getByRole("heading", { level: 2 })
      .all()
      .map((heading) => heading.element().textContent),
  ).toEqual(Object.keys(settingsGroupLabels));
  for (const [group, sections] of Object.entries(settingsGroupLabels)) {
    expect(
      page
        .getByRole("list", { name: group })
        .getByRole("link")
        .all()
        .map((link) => link.element().textContent),
    ).toEqual(sections);
  }

  const system = page.getByRole("list", { name: "System" });
  await userEvent.click(system.getByRole("link", { name: "Email" }));
  await expect.poll(() => router.state.location.pathname).toBe("/settings/email");
  const bar = page.getByRole("banner");
  await expect.element(bar.getByRole("heading", { name: "Email" })).toBeVisible();
  await expect.element(server()).toHaveValue("smtp.example.test");
  const back = bar.getByRole("link", { name: "Settings" });
  await expect.element(back).toHaveAttribute("href", "/settings");
  await userEvent.click(back);
  await expect.poll(() => router.state.location.pathname).toBe("/settings");
  await expect.element(system).toBeVisible();
});
