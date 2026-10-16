import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { PageTabs, pageTab } from "./page-tabs";

test("page tabs are links in a nav named for the page, with the current one underlined", async () => {
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
  const current = tabs.getByRole("link", { name: "Logs" });
  const other = tabs.getByRole("link", { name: "Overview" });
  await expect.element(current).toHaveAttribute("aria-current", "page");
  expect(
    tabs
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Overview", "Logs"]);
  expect(getComputedStyle(current.element()).height).toBe("36px");
  expect(getComputedStyle(current.element()).boxShadow).toMatch(/ 0px -2px 0px 0px inset$/u);
  expect(getComputedStyle(other.element()).boxShadow).toBe("none");
  expect(getComputedStyle(current.element()).color).not.toBe(
    getComputedStyle(other.element()).color,
  );
});
