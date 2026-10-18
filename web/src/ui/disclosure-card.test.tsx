import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { DisclosureCard } from "./disclosure-card";

function Card({ opened }: { opened: boolean }) {
  const [open, setOpen] = useState(opened);
  return (
    <DisclosureCard title="Acquire" open={open} onOpenChange={setOpen} aside={<span>1 of 2</span>}>
      <p>Searching</p>
    </DisclosureCard>
  );
}

test("the title opens and closes the body, and the aside stays in the header", async () => {
  renderKit(<Card opened={false} />);
  const title = page.getByRole("button", { name: "Acquire" });
  await expect.element(page.getByRole("region", { name: "Acquire" })).toBeVisible();
  await expect.element(page.getByText("1 of 2")).toBeVisible();
  await expect.element(title).toHaveAttribute("aria-expanded", "false");
  await expect.element(page.getByText("Searching")).not.toBeInTheDocument();
  await userEvent.click(title);
  await expect.element(page.getByText("Searching")).toBeVisible();
  await userEvent.click(title);
  await expect.element(page.getByText("Searching")).not.toBeInTheDocument();
});
