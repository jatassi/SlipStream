import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Checkbox } from "./checkbox";

function Monitor() {
  const [on, setOn] = useState(false);
  return <Checkbox label="Monitor" checked={on} onCheckedChange={setOn} />;
}

test("a checkbox toggles with Space and with a click on its label", async () => {
  renderKit(<Monitor />);
  const box = page.getByRole("checkbox", { name: "Monitor" });
  await expect.element(box).not.toBeChecked();
  await userEvent.tab();
  await userEvent.keyboard(" ");
  await expect.element(box).toBeChecked();
  await page.getByText("Monitor").click();
  await expect.element(box).not.toBeChecked();
});
