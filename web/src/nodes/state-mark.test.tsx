import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { type NodeState, nodeStates } from "./state";
import { NodeLamp, NodeStateMark } from "./state-mark";

const lampOf = async (container: HTMLElement) =>
  getComputedStyle(
    await vi.waitFor(() => {
      const lamp = container.querySelector('[aria-hidden="true"]');
      if (lamp === null) {
        throw new Error("the lamp hasn't rendered");
      }
      return lamp;
    }),
  );

test.each([
  ["online", "rgb(95, 211, 148)"],
  ["reconnecting", "rgb(255, 181, 71)"],
  ["offline", "rgb(255, 181, 71)"],
  ["mismatched", "rgb(255, 181, 71)"],
  ["lapsed", "rgb(255, 107, 97)"],
] satisfies [NodeState, string][])(
  "a %s Node's lamp is %s, as DESIGN.md colours it",
  async (state, colour) => {
    expect((await lampOf(renderKit(<NodeLamp state={state} />))).backgroundColor).toBe(colour);
  },
);

test("a restarting Node's lamp is ink-1 with the Work's glow", async () => {
  const lamp = await lampOf(renderKit(<NodeLamp state="restarting" />));
  expect(lamp.backgroundColor).toBe("rgb(242, 244, 247)");
  expect(lamp.boxShadow).not.toBe("none");
});

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
    expect(getComputedStyle(page.getByText(word).element()).textDecorationStyle).toBe("dotted");
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
