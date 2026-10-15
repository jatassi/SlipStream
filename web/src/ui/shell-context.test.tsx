import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { useShell } from "./shell-context";

function Shell() {
  return <p>{useShell()}</p>;
}

test("a kit component reads the shell it renders in", async () => {
  renderKit(<Shell />, { shell: "phone" });
  await expect.element(page.getByText("phone")).toBeVisible();
});
