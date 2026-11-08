import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { type Lane, LaneMeter } from "./lane-meter";
import type { ShellKind } from "./shell-context";

const ratio = (marks: Omit<Lane, "label" | "words">): Lane => ({
  label: "Ratio",
  words: (value) => value.toFixed(1),
  ...marks,
});

const lane = (name = "Ratio") => page.getByRole("meter", { name });
const mark = (kind: string) => lane().element().querySelector<HTMLElement>(`[data-mark=${kind}]`);
const colourOf = (element: Element | null) =>
  element === null ? "" : getComputedStyle(element).backgroundColor;

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a lane runs to its goal with a tick at the rule",
  async (shell) => {
    renderKit(<LaneMeter lanes={[ratio({ value: 0.5, rule: 1, goal: 2 })]} risk={false} />, {
      shell,
    });
    await expect.element(lane()).toHaveAttribute("aria-valuetext", "0.5, rule 1.0, goal 2.0");
    await expect.element(lane()).toHaveAttribute("aria-valuemax", "2");
    expect(mark("rule")?.style.left).toBe("50%");
    expect(mark("goal")).toBeNull();
  },
);

test("the rule's tick turns amber while a hit and run is at risk", async () => {
  renderKit(
    <>
      <LaneMeter lanes={[ratio({ value: 0.5, rule: 1, goal: 2 })]} risk />
      <span data-testid="warn" className="bg-warn" />
    </>,
  );
  await expect
    .element(lane())
    .toHaveAttribute("aria-valuetext", "0.5, rule 1.0, at risk, goal 2.0");
  expect(mark("rule")).toBeNull();
  expect(colourOf(mark("risk"))).toBe(colourOf(page.getByTestId("warn").element()));
});

test("a lane the rule doesn't count has no tick", async () => {
  renderKit(<LaneMeter lanes={[ratio({ value: 0.5, goal: 2 })]} risk />);
  await expect.element(lane()).toHaveAttribute("aria-valuetext", "0.5, goal 2.0");
  expect(mark("rule")).toBeNull();
  expect(mark("risk")).toBeNull();
});

test("a goal short of the rule is marked, and the lane runs on to the rule", async () => {
  renderKit(<LaneMeter lanes={[ratio({ value: 0.5, rule: 2, goal: 1 })]} risk={false} />);
  await expect.element(lane()).toHaveAttribute("aria-valuemax", "2");
  expect(mark("goal")?.style.left).toBe("50%");
  expect(mark("rule")?.style.left).toBe("100%");
});

test("with no goal, a lane past its rule is full", async () => {
  renderKit(<LaneMeter lanes={[ratio({ value: 3, rule: 1 })]} risk={false} />);
  await expect.element(lane()).toHaveAttribute("aria-valuenow", "1");
  await expect.element(lane()).toHaveAttribute("aria-valuemax", "1");
});

test("a lane neither the rule nor the goal counts says so and draws no fill", async () => {
  renderKit(<LaneMeter lanes={[ratio({ value: 1.2 })]} risk={false} />);
  await expect.element(lane()).toHaveAttribute("aria-valuetext", "1.2, not counted");
  expect(lane().element().querySelector("[class*=bg-ink-2]")).toBeNull();
});

test("the large meter names each lane and its value beside it", async () => {
  renderKit(
    <LaneMeter
      lanes={[
        ratio({ value: 0.5, rule: 1, goal: 2 }),
        { label: "Seeding time", value: 3, goal: 14, words: (days) => `${days} days` },
      ]}
      risk={false}
      size="large"
    />,
  );
  await expect.element(lane("Seeding time")).toBeVisible();
  expect(document.body.textContent).toContain("Ratio0.5 of 2.0Seeding time3 days of 14 days");
});
