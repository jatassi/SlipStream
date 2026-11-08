import { expect, test } from "vitest";

import type { DownloadclientsExternalClient } from "../api/gen/model";

import { attic, atticAway, atticRefused, transmissionKind } from "../test/external-clients";
import { type ClientStatus, statusOf } from "./client-status";

test.each<{ name: string; client: DownloadclientsExternalClient; words: ClientStatus }>([
  {
    name: "a responding client names the version its Node reached",
    client: attic,
    words: { signal: "ok", chip: "Responding", sentence: "home reached Transmission 4.1.0." },
  },
  {
    name: "a client that isn't responding says what to check",
    client: atticAway,
    words: {
      signal: "fail",
      chip: "Not responding",
      sentence:
        "Transmission isn't responding. Check that it's running and that home can reach 192.168.1.20:9091.",
    },
  },
  {
    name: "a refused sign-in says polling stopped and how to start it",
    client: atticRefused,
    words: {
      signal: "fail",
      chip: "Sign-in refused",
      sentence:
        "Transmission rejected the saved password. SlipStream stopped checking it to avoid a lockout. Change the password, or select Check now once it's fixed.",
    },
  },
  {
    name: "a client that needs attention carries its problem's sentence",
    client: {
      ...attic,
      reach: "needs_attention",
      problem: {
        cause: "label_plugin_off",
        sentence:
          "Transmission's Label plugin is off. Turn it on in Transmission's preferences and test again.",
        fields: [],
      },
    },
    words: {
      signal: "fail",
      chip: "Needs attention",
      sentence:
        "Transmission's Label plugin is off. Turn it on in Transmission's preferences and test again.",
    },
  },
  {
    name: "a client turned off says SlipStream isn't checking it",
    client: { ...attic, on: false, reach: "off" },
    words: {
      signal: undefined,
      chip: "Turned off",
      sentence: "SlipStream isn't checking Transmission while it's turned off.",
    },
  },
  {
    name: "a client on an offline Node waits for the Node",
    client: { ...attic, reach: "node_offline" },
    words: {
      signal: undefined,
      chip: "Node offline",
      sentence: "home is offline. SlipStream checks Transmission again once home is back.",
    },
  },
  {
    name: "a client its Node hasn't polled yet says so",
    client: { ...attic, reach: "unchecked", since: null, productVersion: "" },
    words: {
      signal: undefined,
      chip: "Not checked yet",
      sentence: "home hasn't checked Transmission yet.",
    },
  },
])("$name", ({ client, words }) => {
  expect(statusOf(client, transmissionKind, "home")).toEqual(words);
});
