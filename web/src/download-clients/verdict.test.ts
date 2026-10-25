import { expect, test } from "vitest";

import { homeCard, unbound } from "../test/download-clients";
import { verdictOf, verdictWords } from "./verdict";

const bound = { ...unbound, bound: "192.168.1.20:6881" };

test.each([
  {
    name: "a reachable port names the port peers reached",
    card: {
      ...homeCard,
      client: { ...bound, reach: "reachable", reachedAt: "2026-10-12T09:05:00Z" },
    },
    words: {
      signal: "ok",
      chip: "Reachable",
      sentence: "Listening on 192.168.1.20:6881. Peers from the internet have reached port 6881.",
    },
  },
  {
    name: "an unproven port is neutral and says what to check",
    card: homeCard,
    words: {
      sentence:
        "Listening on 192.168.1.20:6881. No peer from the internet has connected to port 6881 yet, so it may not be reachable.",
      hint: "If no peer connects, check that your router forwards port 6881 to this computer. Downloads still work, but with fewer peers.",
    },
  },
  {
    name: "a port it can't bind is a failure with the Node's reason",
    card: {
      ...homeCard,
      client: { ...unbound, listenError: "address already in use" },
    },
    words: {
      signal: "fail",
      chip: "Not listening",
      sentence: "Can't listen on 0.0.0.0:6881: address already in use.",
      hint: "No peer can connect to it until it listens. If another program is using the port, free it or choose another, then restart.",
    },
  },
  {
    name: "no listener yet and no reason reads as not yet",
    card: { ...homeCard, client: unbound },
    words: {
      signal: "fail",
      chip: "Not listening",
      sentence: "Not listening on 0.0.0.0:6881 yet.",
      hint: "No peer can connect to it until it listens. If another program is using the port, free it or choose another, then restart.",
    },
  },
  {
    name: "an offline Node can't report its client",
    card: {
      nodeId: homeCard.nodeId,
      nodeName: "home",
      inStep: false,
      limits: homeCard.limits,
      torrents: 0,
      totals: homeCard.totals,
    },
    words: {
      sentence: "home is offline, so its client's status is unknown.",
      hint: "Changed settings apply when it reconnects.",
    },
  },
  {
    name: "an in-step Node whose client hasn't reported yet says so",
    card: {
      nodeId: homeCard.nodeId,
      nodeName: "home",
      inStep: true,
      limits: homeCard.limits,
      torrents: 0,
      totals: homeCard.totals,
    },
    words: { sentence: "home's client hasn't reported yet." },
  },
] as const)("$name", ({ card, words }) => {
  expect(verdictWords(verdictOf(card), card.nodeName)).toEqual(words);
});
