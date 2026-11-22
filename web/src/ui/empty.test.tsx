import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Button } from "./button";
import { Empty } from "./empty";

test("an empty shows its sentence and its way forward", async () => {
  renderKit(
    <Empty look="sheet" action={<Button>Clear filters</Button>}>
      No entries match these filters.
    </Empty>,
  );
  await expect.element(page.getByText("No entries match these filters.")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Clear filters" })).toBeVisible();
});
