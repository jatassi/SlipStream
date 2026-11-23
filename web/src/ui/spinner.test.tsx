import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";

import { Spinner } from "./spinner";

test("an unlabelled spinner is hidden from assistive technology", async () => {
  renderKit(
    <button type="button">
      <Spinner />
      Search
    </button>,
  );
  await expect.element(page.getByRole("button", { name: "Search" })).toBeVisible();
  await expect.element(page.getByRole("status")).not.toBeInTheDocument();
});
