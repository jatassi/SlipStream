import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MailSettings } from "../../../api/gen/model";
import { getGetAddressMockHandler } from "../../../api/gen/query/identity/identity.msw";
import { getGetMailSettingsMockHandler } from "../../../api/gen/query/mail/mail.msw";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";

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

test("a tab opens its section", async () => {
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

test("on phone, Settings lists its sections, and each pushes with a back link to Settings", async () => {
  await asPhone();
  const { router } = renderApp({
    path: "/settings",
    handlers: [getGetMailSettingsMockHandler(mail)],
  });
  await expect.element(page.getByRole("heading", { name: "Settings", level: 1 })).toBeVisible();
  await expect.element(page.getByRole("navigation", { name: "Settings" })).not.toBeInTheDocument();
  const sections = page.getByRole("list", { name: "Settings" });
  expect(
    sections
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Address", "Email", "Sign-in", "API key", "Targets", "Profiles", "Editions"]);

  await userEvent.click(sections.getByRole("link", { name: "Email" }));
  await expect.poll(() => router.state.location.pathname).toBe("/settings/email");
  const bar = page.getByRole("banner");
  await expect.element(bar.getByRole("heading", { name: "Email" })).toBeVisible();
  await expect.element(server()).toHaveValue("smtp.example.test");
  const back = bar.getByRole("link", { name: "Settings" });
  await expect.element(back).toHaveAttribute("href", "/settings");
  await userEvent.click(back);
  await expect.poll(() => router.state.location.pathname).toBe("/settings");
  await expect.element(page.getByRole("list", { name: "Settings" })).toBeVisible();
});
