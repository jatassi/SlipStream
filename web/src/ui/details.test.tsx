import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Details } from "./details";

test("Details shows the plain sentence and hides the machinery until opened", async () => {
  renderKit(<Details summary="SlipStream couldn't reach this page.">Code: internal_error</Details>);
  await expect.element(page.getByText("SlipStream couldn't reach this page.")).toBeVisible();
  const disclosure = page.getByRole("button", { name: "Details" });
  await expect.element(disclosure).toHaveAttribute("aria-expanded", "false");
  await expect.element(page.getByText("Code: internal_error")).not.toBeInTheDocument();
  await userEvent.tab();
  await userEvent.keyboard("{Enter}");
  await expect.element(disclosure).toHaveAttribute("aria-expanded", "true");
  await expect.element(page.getByText("Code: internal_error")).toBeVisible();
  await userEvent.keyboard(" ");
  await expect.element(page.getByText("Code: internal_error")).not.toBeInTheDocument();
});
