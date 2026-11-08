import type {
  DownloadclientsExternalClient as External,
  DownloadclientsKind as Kind,
} from "../api/gen/model";
import type { Signal } from "../ui/signal-chip";

/** An external client's poll state in words: a chip, and the sentence that says what to do when it carries one. */
export type ClientStatus = {
  readonly signal: Signal | undefined;
  readonly chip: string;
  readonly sentence: string | undefined;
};

type Words = (client: External, kind: Kind, node: string) => ClientStatus;

const failing =
  (chip: string): Words =>
  (client) => ({ signal: "fail", chip, sentence: client.problem?.sentence });

const statuses: { readonly [R in External["reach"]]: Words } = {
  responding: (client, kind, node) => ({
    signal: "ok",
    chip: "Responding",
    sentence: `${node} reached ${kind.name} ${client.productVersion}.`,
  }),
  not_responding: failing("Not responding"),
  sign_in_refused: failing("Sign-in refused"),
  needs_attention: failing("Needs attention"),
  off: (client) => ({
    signal: undefined,
    chip: "Turned off",
    sentence: `SlipStream isn't checking ${client.name} while it's turned off.`,
  }),
  node_offline: (client, _, node) => ({
    signal: undefined,
    chip: "Node offline",
    sentence: `${node} is offline. SlipStream checks ${client.name} again once ${node} is back.`,
  }),
  unchecked: (client, _, node) => ({
    signal: undefined,
    chip: "Not checked yet",
    sentence: `${node} hasn't checked ${client.name} yet.`,
  }),
};

export const statusOf = (client: External, kind: Kind, node: string): ClientStatus =>
  statuses[client.reach](client, kind, node);
