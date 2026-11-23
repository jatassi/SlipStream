import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { type NodeState, nodeStates } from "./state";
import { NodeLamp, NodeStateMark } from "./state-mark";

test("a lamp says its word to a screen reader, unless the word sits beside it", async () => {
  renderKit(
    <ul>
      <li>
        <NodeLamp state="lapsed" />
        seedbox
      </li>
      <li>
        <NodeLamp state="lapsed" decorative />
        nas
      </li>
    </ul>,
  );
  const rows = page.getByRole("listitem");
  await expect.element(rows.nth(0)).toHaveTextContent("Offline too longseedbox");
  expect(rows.nth(1).element().textContent).toBe("nas");
});

test.each(["reconnecting", "offline", "lapsed", "mismatched"] satisfies NodeState[])(
  "a %s Node's word is a button described by its definition, which hovering opens",
  async (state) => {
    const { word, definition } = nodeStates[state];
    renderKit(<NodeStateMark state={state} />);
    const trigger = page.getByRole("button", { name: word });
    await expect.element(trigger).toHaveAccessibleDescription(definition);
    await userEvent.hover(trigger);
    await expect.element(page.getByRole("dialog", { name: word })).toHaveTextContent(definition);
  },
);

test.each(["online", "restarting"] satisfies NodeState[])(
  "a %s Node's word is plain",
  async (state) => {
    renderKit(<NodeStateMark state={state} />);
    await expect.element(page.getByText(nodeStates[state].word)).toBeVisible();
    await expect.element(page.getByRole("button")).not.toBeInTheDocument();
  },
);
