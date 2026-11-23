import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { RowSwitch } from "./row-switch";

function Episode({ start }: { start: boolean | "mixed" }) {
  const [on, setOn] = useState(start);
  return <RowSwitch label="Monitor S01E02" checked={on} onCheckedChange={setOn} />;
}

const name = "Monitor S01E02";

test("a row switch is a switch that a click and Space toggle", async () => {
  renderKit(<Episode start={false} />);
  const control = page.getByRole("switch", { name });
  await expect.element(control).toHaveAttribute("aria-checked", "false");
  await control.click();
  await expect.element(control).toHaveAttribute("aria-checked", "true");
  await expect.element(control).toHaveFocus();
  await userEvent.keyboard(" ");
  await expect.element(control).toHaveAttribute("aria-checked", "false");
});

test("a mixed row switch reads mixed and turns on", async () => {
  renderKit(<Episode start="mixed" />);
  const control = page.getByRole("switch", { name });
  await expect.element(control).toHaveAttribute("aria-checked", "mixed");
  await control.click();
  await expect.element(control).toHaveAttribute("aria-checked", "true");
});

test("a disabled row switch doesn't change", async () => {
  let changes = 0;
  renderKit(
    <RowSwitch
      label={name}
      checked
      disabled
      onCheckedChange={() => {
        changes += 1;
      }}
    />,
  );
  const control = page.getByRole("switch", { name });
  await expect.element(control).toBeDisabled();
  await control.click({ force: true });
  expect(changes).toBe(0);
});
