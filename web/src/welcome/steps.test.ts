import { expect, test } from "vitest";

import {
  type WelcomeFacts,
  type WelcomeRow,
  nextWelcome,
  placeWelcome,
  shownWelcome,
  welcomeSteps,
} from "./steps";

// A later spec's row, put where S41 (Member portal) and S42 (Portal inbox and
// Member notifications) put theirs: before done.
const later: WelcomeRow = {
  id: "later",
  title: "Later",
  shows: (facts) => facts.servers > 0,
  Step: () => null,
};
const rows = [...welcomeSteps.slice(0, -1), later, ...welcomeSteps.slice(-1)];

const ids = (shown: readonly WelcomeRow[]) => shown.map((row) => row.id);

test.each<[WelcomeFacts, string[]]>([
  [{ passkeys: true, servers: 2 }, ["passkey", "watch", "later", "done"]],
  [{ passkeys: false, servers: 1 }, ["watch", "later", "done"]],
  [{ passkeys: true, servers: 0 }, ["passkey", "done"]],
  [{ passkeys: false, servers: 0 }, ["done"]],
])("with %o the welcome steps walk %o", (facts, walk) => {
  const shown = shownWelcome(rows, facts);
  expect(ids(shown)).toEqual(walk);

  let at = placeWelcome(shown);
  const walked = [at.id];
  for (let step = 1; step < shown.length; step += 1) {
    at = nextWelcome(shown, at);
    walked.push(at.id);
  }
  expect(walked).toEqual(walk);
  expect(nextWelcome(shown, placeWelcome(shown, "done")).id).toBe("done");

  for (const row of rows) {
    expect(placeWelcome(shown, row.id).id).toBe(walk.includes(row.id) ? row.id : walk[0]);
  }
  expect(placeWelcome(shown, "nowhere").id).toBe(walk[0]);
});
