import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Mark } from "./mark";

test("a decorative mark is hidden from assistive technology", async () => {
  renderKit(
    <a href="/">
      <Mark decorative />
      Home
    </a>,
  );
  await expect.element(page.getByRole("link", { name: "Home" })).toBeVisible();
  await expect.element(page.getByRole("img")).not.toBeInTheDocument();
});
