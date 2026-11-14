import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderApp } from "../test/render-app";
import { housekeeping, succeededRun, taskServer } from "../test/tasks";

test("a succeeded unit's note shows on its line in the run", async () => {
  const tasks = taskServer({ tasks: [housekeeping] });
  const [runsUnit, seedboxUnit] = succeededRun.units;
  if (runsUnit === undefined || seedboxUnit === undefined) {
    throw new Error("the succeeded run has two units");
  }
  tasks.state.runs = [
    { ...succeededRun, units: [{ ...runsUnit, note: "Pruned 120 runs." }, seedboxUnit] },
  ];
  renderApp({ path: "/tasks/housekeeping", handlers: tasks.handlers });
  await userEvent.click(page.getByRole("region", { name: "Runs" }).getByRole("button").first());
  await expect
    .element(page.getByRole("list", { name: "Units in this run" }).getByRole("listitem").first())
    .toMatchTextContent(/Run history.*Pruned 120 runs\./u);
});
