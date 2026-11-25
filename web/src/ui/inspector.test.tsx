import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Inspector, InspectorSection } from "./inspector";

test("the Inspector is a labelled complementary landmark that closes", async () => {
  const closed: string[] = [];
  renderKit(
    <Inspector
      title="Dune: Part Two"
      onClose={() => {
        closed.push("close");
      }}
    >
      <InspectorSection title="Release">
        <p>Dune.Part.Two.2024.2160p</p>
      </InspectorSection>
    </Inspector>,
  );
  const panel = page.getByRole("complementary", { name: "Dune: Part Two" });
  await expect.element(panel.getByRole("region", { name: "Release" })).not.toBeInTheDocument();
  await expect.element(panel.getByRole("heading", { name: "Release" })).toBeVisible();
  await panel.getByRole("button", { name: "Close" }).click();
  expect(closed).toStrictEqual(["close"]);
});

test("Escape closes the Inspector", async () => {
  const closed: string[] = [];
  renderKit(
    <Inspector
      title="Dune: Part Two"
      onClose={() => {
        closed.push("close");
      }}
    >
      <p>Dune.Part.Two.2024.2160p</p>
    </Inspector>,
  );
  await expect.element(page.getByRole("complementary", { name: "Dune: Part Two" })).toBeVisible();
  await userEvent.keyboard("{Escape}");
  expect(closed).toStrictEqual(["close"]);
});
