import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { PageTabs, pageTab } from "./page-tabs";

test("page tabs are links in a nav named for the page, with the current one marked", async () => {
  renderKit(
    <PageTabs label="media-server">
      <a href="/nodes/a" className={pageTab}>
        Overview
      </a>
      <a href="/nodes/a/logs" className={pageTab} aria-current="page">
        Logs
      </a>
    </PageTabs>,
  );
  const tabs = page.getByRole("navigation", { name: "media-server" });
  await expect
    .element(tabs.getByRole("link", { name: "Logs" }))
    .toHaveAttribute("aria-current", "page");
  expect(
    tabs
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Overview", "Logs"]);
});
