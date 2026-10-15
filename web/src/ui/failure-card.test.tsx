import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { FailureCard } from "./failure-card";
import type { ShellKind } from "./shell-context";

const shells = ["desktop", "phone"] as const satisfies readonly ShellKind[];

describe.each(shells)("in the %s shell", (shell) => {
  test("a failure card names its stage and offers the fix as a button", async () => {
    const tried: string[] = [];
    renderKit(
      <FailureCard
        stage="import"
        time="21:48"
        work="Dune: Part Two"
        facts={[
          { label: "Cause", value: "The file is not a video." },
          { label: "Recovery", value: "Search for another release." },
        ]}
        fix={{
          label: "Try again",
          onClick: () => {
            tried.push("fix");
          },
        }}
        actions={[
          {
            label: "Remove",
            onClick: () => {
              tried.push("remove");
            },
          },
        ]}
        track={{ skipped: [] }}
      />,
      { shell },
    );
    const card = page.getByRole("article", { name: "Dune: Part Two" });
    await expect.element(card.getByText("Failed at Import")).toBeVisible();
    await expect.element(card.getByText("The file is not a video.")).toBeVisible();
    await expect
      .element(card.getByRole("img", { name: "Dune: Part Two failed at Import" }))
      .toBeVisible();
    await card.getByRole("button", { name: "Try again" }).click();
    expect(tried).toStrictEqual(["fix"]);
  });
});

test("a phone failure card holds its other actions behind More", async () => {
  renderKit(
    <FailureCard
      stage="import"
      time="21:48"
      work="Dune: Part Two"
      facts={[{ label: "Cause", value: "The file is not a video." }]}
      fix={{ label: "Try again", onClick: () => {} }}
      actions={[{ label: "Remove", onClick: () => {} }]}
    />,
    { shell: "phone" },
  );
  await expect.element(page.getByRole("button", { name: "Remove" })).not.toBeInTheDocument();
  await page.getByRole("button", { name: "More" }).click();
  await expect.element(page.getByRole("button", { name: "Remove" })).toBeVisible();
});

test("a compact failure card drops its stage track", async () => {
  renderKit(
    <FailureCard
      form="compact"
      stage="import"
      time="21:48"
      work="Dune: Part Two"
      facts={[{ label: "Cause", value: "The file is not a video." }]}
      fix={{ label: "Try again", onClick: () => {} }}
      track={{ skipped: [] }}
    />,
  );
  await expect.element(page.getByRole("button", { name: "Try again" })).toBeVisible();
  await expect.element(page.getByRole("img")).not.toBeInTheDocument();
});
