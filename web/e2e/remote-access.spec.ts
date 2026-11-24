import type { Page } from "@playwright/test";

import { expect, expectAccessible, test } from "./fixtures.ts";
import { isPhone } from "./shell.ts";

test.use({ scenario: "remote-own-proxy", at: "atlas" });
test.describe.configure({ timeout: 300_000 });

const checked = { timeout: 120_000 };

const suggestion = (page: Page, subject: string) =>
  page.getByRole("list", { name: "Suggestions" }).getByRole("listitem", { name: subject });

const route = (page: Page, subject: string) => page.getByRole("region", { name: subject });

async function expectVerdict(page: Page, subject: string, verdict: string, tested: boolean) {
  const card = route(page, subject);
  await expect(card.getByRole("button", { name: verdict, exact: true })).toBeVisible(checked);
  await expect(card.getByRole("button", { name: "Not tested", exact: true })).toHaveCount(
    tested ? 0 : 1,
  );
}

// The hop's checks open in the Inspector, or in a sheet on phone.
const hopChecks = (page: Page, title: string) =>
  isPhone(page)
    ? page.getByRole("dialog", { name: title })
    : page.getByRole("complementary", { name: title });

async function closeHop(page: Page, title: string) {
  if (isPhone(page)) {
    await page.keyboard.press("Escape");
  } else {
    await hopChecks(page, title).getByRole("button", { name: "Close" }).click();
  }
  await expect(hopChecks(page, title)).toHaveCount(0);
}

test("the UI and portal and Plex, set up through the Admin's own proxy, read Reachable, and an expired certificate fails Plex's TLS hop until it is renewed", async ({
  adminPage: page,
  world,
}) => {
  await page.goto("/nodes");
  await page.getByRole("link", { name: /^Remote access/u }).click();
  await expect(page).toHaveURL(/\/nodes\/remote-access$/u);

  await page.getByRole("button", { name: "Choose what to use away from home" }).click();
  const plex = suggestion(page, "Plex");
  await expect(plex.getByRole("textbox", { name: "Public address" })).toHaveValue(
    /^https:\/\/plex\.proxy\.fake\.localhost/u,
  );
  await plex.getByRole("button", { name: "Add", exact: true }).click();
  const ui = suggestion(page, "SlipStream UI and portal");
  await ui
    .getByRole("textbox", { name: "Address" })
    .fill("https://slipstream.proxy.fake.localhost");
  await ui.getByRole("button", { name: "Save and add" }).click();
  await expect(page.getByRole("list", { name: "Suggestions" })).toHaveCount(0);

  await expectVerdict(page, "SlipStream UI and portal", "Reachable", true);
  await expectVerdict(page, "Plex", "Reachable", true);
  await expectAccessible(page);

  await world.fw.fault.set({ id: "the-vps", target: "proxy", node: "atlas", fault: "tls=expired" });
  await page.getByRole("button", { name: "Run again", exact: true }).click();
  await expectVerdict(page, "Plex", "Fixable", false);
  await route(page, "Plex").getByRole("button", { name: "TLS: Fails" }).click();
  const tls = hopChecks(page, "TLS · Plex");
  await expect(tls.getByRole("heading", { name: /^Fails/u })).toBeVisible();
  await expectAccessible(page);
  await closeHop(page, "TLS · Plex");

  await world.fw.fault.clear("the-vps");
  await route(page, "Plex").getByRole("button", { name: "Plex actions" }).click();
  await (
    isPhone(page)
      ? page
          .getByRole("dialog", { name: "Plex actions" })
          .getByRole("button", { name: "Run again" })
      : page.getByRole("menuitem", { name: "Run again" })
  ).click();
  await expectVerdict(page, "Plex", "Reachable", true);
});
