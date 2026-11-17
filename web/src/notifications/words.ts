import type {
  NotificationsConnection as Connection,
  NotificationsProvider as Provider,
} from "../api/gen/model";
import { type Clock, countWords, laterWords, pastAtWords, sinceWords } from "../format";

const connectionNoun = { one: "connection", many: "connections" } as const;

const grouped = new Intl.NumberFormat("en");

export const providerOf = (providers: readonly Provider[], id: string): Provider => {
  const provider = providers.find((each) => each.id === id);
  if (provider === undefined) {
    throw new Error(`the providers list has no ${id}`);
  }
  return provider;
};

/** A connection as every sentence names it: "Discord 'Family'". */
export const labelOf = (provider: Provider, connection: Connection): string =>
  `${provider.label} '${connection.name}'`;

function onWords(connections: readonly Connection[]): string {
  const on = connections.filter(({ status }) => status.shown !== "off").length;
  if (on === 0) {
    const all = connections.length;
    return `${countWords(all, connectionNoun)} ${all === 1 ? "is" : "are"} off.`;
  }
  return `${countWords(on, connectionNoun)} ${on === 1 ? "sends" : "send"} notifications.`;
}

function troubleWords(label: string, connection: Connection, clock: Clock): string | undefined {
  const { shown, since, until } = connection.status;
  if (shown === "failing" && since !== null) {
    return `Messages to ${label} have failed since ${sinceWords(since, clock)}.`;
  }
  if (shown === "waiting" && until !== null) {
    return `Messages to ${label} wait until ${laterWords(until, clock).words}.`;
  }
  if (shown === "paused") {
    return `${label} is paused.`;
  }
  if (shown === "holding") {
    return `Messages to ${label} wait until its Node is back.`;
  }
  return undefined;
}

/**
 * The section's lead: how many connections send, then one sentence for each
 * one that is failing, waiting, paused or holding its messages.
 */
export function sectionSentence(
  connections: readonly Connection[],
  providers: readonly Provider[],
  clock: Clock,
): string {
  if (connections.length === 0) {
    return "Add a connection to get a message when SlipStream imports something or a Grab fails.";
  }
  const troubles = connections.flatMap((connection) => {
    const words = troubleWords(
      labelOf(providerOf(providers, connection.provider), connection),
      connection,
      clock,
    );
    return words === undefined ? [] : [words];
  });
  return [onWords(connections), ...troubles].join(" ");
}

export type StatusLevel = "ok" | "warn" | "fail" | "off";

type Shown = Connection["status"]["shown"];

const lines: {
  readonly [S in Shown]: {
    level: StatusLevel;
    text: (status: Connection["status"], clock: Clock) => string;
  };
} = {
  off: { level: "off", text: () => "Off." },
  ok: {
    level: "ok",
    text: ({ lastSent }, clock) =>
      lastSent === null ? "Nothing sent yet." : `Sent ${pastAtWords(lastSent, clock)}.`,
  },
  failing: {
    level: "warn",
    text: ({ since, reason }, clock) =>
      since === null ? reason : `Failing since ${sinceWords(since, clock)}. ${reason}`,
  },
  waiting: { level: "warn", text: ({ reason }) => reason },
  holding: { level: "warn", text: ({ reason }) => reason },
  paused: { level: "fail", text: ({ reason }) => `Paused. ${reason}` },
};

/** A connection's row line: when it last sent, or why it isn't sending and what to do. */
export function statusLine(
  connection: Connection,
  clock: Clock,
): { level: StatusLevel; text: string } {
  const line = lines[connection.status.shown];
  return { level: line.level, text: line.text(connection.status, clock) };
}

/** How many messages weren't sent since the connection was last saved or tested, when any. */
export function givenUpNote(connection: Connection, provider: Provider): string | undefined {
  const count = connection.status.givenUp;
  if (count === 0) {
    return undefined;
  }
  return count === 1
    ? `1 message wasn't sent, because it waited too long or ${provider.label} refused it.`
    : `${grouped.format(count)} messages weren't sent, because they waited too long or ${provider.label} refused them.`;
}
