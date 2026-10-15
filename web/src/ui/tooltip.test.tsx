import { Activity01Icon } from "@hugeicons/core-free-icons";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit, type Density } from "../test/render-kit";
import { Icon } from "./icon";
import { Tooltip } from "./tooltip";

// The pointer starts at the page's top left, so the row sits clear of it.
function RailRow() {
  return (
    <nav className="p-10">
      <button type="button">Collapse</button>
      <Tooltip label="Activity" count={2} side="right">
        <a href="#activity" className="inline-flex">
          <Icon icon={Activity01Icon} />
        </a>
      </Tooltip>
    </nav>
  );
}

for (const density of ["console", "touch"] satisfies Density[]) {
  test(`a ${density} tooltip shows its label and count when its trigger takes focus`, async () => {
    renderKit(<RailRow />, { density });
    const trigger = page.getByRole("link", { name: "Activity, 2" });
    await expect.element(page.getByText("Activity", { exact: true })).not.toBeInTheDocument();

    await userEvent.click(page.getByRole("button", { name: "Collapse" }));
    await userEvent.tab();
    await expect.element(trigger).toHaveFocus();
    await expect.element(page.getByText("Activity", { exact: true })).toBeVisible();
    await expect.element(page.getByText("2", { exact: true })).toBeVisible();
  });
}

test("a tooltip shows on hover", async () => {
  renderKit(<RailRow />);
  await userEvent.hover(page.getByRole("link", { name: "Activity, 2" }));
  await expect.element(page.getByText("Activity", { exact: true })).toBeVisible();
});
