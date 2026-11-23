import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { StateButton } from "./state-square";

test("a cell is a button named with its state", async () => {
  renderKit(<StateButton state="missing" label="S01E02 Archive" />);
  await expect.element(page.getByRole("button", { name: "S01E02 Archive, Missing" })).toBeVisible();
});
