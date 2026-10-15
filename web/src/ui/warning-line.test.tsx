import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { FailureCard } from "./failure-card";
import { WarningLine, type WarningLineProps } from "./warning-line";

function relayWarning(fixed: string[]): WarningLineProps {
  return {
    chip: "Degraded",
    subject: "relay-fra",
    detail: "Half its probes time out.",
    since: "08:30",
    action: {
      label: "Check relay",
      onClick: () => {
        fixed.push("check");
      },
    },
  };
}

test("on desktop a warning line names its subject and offers one warn action", async () => {
  const fixed: string[] = [];
  renderKit(<WarningLine {...relayWarning(fixed)} />);
  await expect.element(page.getByText("Degraded")).toBeVisible();
  await expect.element(page.getByText("Half its probes time out.")).toBeVisible();
  await expect.element(page.getByText("since 08:30")).toBeVisible();
  await page.getByRole("button", { name: "Check relay" }).click();
  expect(fixed).toStrictEqual(["check"]);
});

test("on a phone a warning line is a row that opens a sheet with its detail and action", async () => {
  const fixed: string[] = [];
  renderKit(<WarningLine {...relayWarning(fixed)} />, { shell: "phone" });
  const row = page.getByRole("button", { name: "Degraded relay-fra" });
  await expect.element(row).toBeVisible();
  expect(getComputedStyle(row.element()).height).toBe("48px");
  await expect.element(page.getByText("Half its probes time out.")).not.toBeInTheDocument();

  await row.click();
  const sheet = page.getByRole("dialog", { name: "relay-fra" });
  await expect.element(sheet.getByText("Half its probes time out.")).toBeVisible();
  await expect.element(sheet.getByText("since 08:30")).toBeVisible();
  await sheet.getByRole("button", { name: "Check relay" }).click();
  expect(fixed).toStrictEqual(["check"]);
  await expect.element(sheet).not.toBeInTheDocument();
});

test("a warning attached to a failure card is the card's last strip", async () => {
  renderKit(
    <FailureCard
      stage="transfer"
      time="21:48"
      work="Dune: Part Two"
      facts={[{ label: "Cause", value: "The link to seedbox dropped." }]}
      fix={{ label: "Try again", onClick: () => {} }}
      warning={relayWarning([])}
    />,
  );
  const card = page.getByRole("article", { name: "Dune: Part Two" });
  await expect.element(card.getByRole("button", { name: "Check relay" })).toBeVisible();
  expect(card.element().lastElementChild?.textContent).toBe(
    "Degradedrelay-fraHalf its probes time out.since 08:30Check relay",
  );
});
