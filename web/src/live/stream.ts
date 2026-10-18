import type { InvalidateQueryFilters, QueryClient } from "@tanstack/react-query";

import { isProblem } from "../api/errors";
import { getSession } from "../api/gen/query/identity/identity";
import { LiveEvent } from "../api/gen/zod/slipStream.schemas.zod";
import { effects } from "./effects";
import { refresh } from "./refresh";
import { type Source, openSource } from "./source";
import type {
  LiveEffect,
  LiveEffects,
  LiveState,
  PayloadOf,
  ProductEventName,
  SessionEndReason,
} from "./types";

const streamUrl = "/api/v1/events";
const hiddenRelease = 10_000;
const reopenDelays = [1000, 2000, 5000, 10_000, 30_000] as const;
const table: LiveEffects = effects;

const eventNames = LiveEvent.options.flatMap((branch) => branch.shape.event.options);

type ParsedEvent = ReturnType<typeof LiveEvent.parse>;
type ControlEvent = Extract<ParsedEvent, { event: `live.${string}` }>;

const isControl = (event: ParsedEvent): event is ControlEvent => event.event.startsWith("live.");

export type LiveStream = {
  readonly connect: () => Promise<void>;
  readonly stop: () => void;
  readonly getState: () => LiveState;
  readonly subscribe: (listener: () => void) => () => void;
};

type Options = {
  queryClient: QueryClient;
  onSessionEnded: (reason: SessionEndReason) => void;
  onAccessChanged: () => void;
};

function pending(): { ready: Promise<void>; settle: () => void } {
  let settle!: () => void;
  const ready = new Promise<void>((resolve) => {
    settle = resolve;
  });
  return { ready, settle };
}

class Stream {
  state: LiveState = "idle";
  readonly listeners = new Set<() => void>();
  private running: { ready: Promise<void>; settle: () => void } | undefined;
  private source: Source | undefined;
  private reopens = 0;
  private reopenTimer: ReturnType<typeof setTimeout> | undefined;
  private releaseTimer: ReturnType<typeof setTimeout> | undefined;

  private readonly options: Options;

  constructor(options: Options) {
    this.options = options;
  }

  connect(): Promise<void> {
    if (this.running !== undefined) {
      return this.running.ready;
    }
    this.running = pending();
    const { ready, settle } = this.running;
    document.addEventListener("visibilitychange", this.onVisibilityChange);
    if (document.visibilityState === "hidden") {
      this.setState("paused");
      settle();
    } else {
      this.open();
    }
    return ready;
  }

  stop(): void {
    if (this.running === undefined) {
      return;
    }
    this.running.settle();
    this.running = undefined;
    document.removeEventListener("visibilitychange", this.onVisibilityChange);
    clearTimeout(this.releaseTimer);
    this.releaseTimer = undefined;
    this.close();
    this.setState("idle");
  }

  private setState(next: LiveState) {
    if (this.state === next) {
      return;
    }
    this.state = next;
    for (const listener of this.listeners) {
      listener();
    }
  }

  private readonly open = () => {
    this.source = openSource(streamUrl, eventNames, {
      open: () => {
        this.reopens = 0;
        this.setState("open");
      },
      error: (closed) => {
        this.onError(closed);
      },
      message: this.onMessage,
    });
    if (this.state === "idle" || this.state === "paused") {
      this.setState("connecting");
    }
  };

  private close() {
    this.source?.close();
    this.source = undefined;
    clearTimeout(this.reopenTimer);
    this.reopenTimer = undefined;
  }

  private onError(closed: boolean) {
    this.running?.settle();
    this.setState("reconnecting");
    if (closed) {
      this.close();
      void this.recover();
    }
  }

  private async recover() {
    const signedOut = await getSession().then(
      () => false,
      (error: unknown) => isProblem(error, "signed_out"),
    );
    if (this.state !== "reconnecting" || this.source !== undefined) {
      return;
    }
    if (signedOut) {
      this.stop();
      this.options.onSessionEnded("expired");
      return;
    }
    const delay = reopenDelays[Math.min(this.reopens, reopenDelays.length - 1)];
    this.reopens += 1;
    this.reopenTimer = setTimeout(this.open, delay);
  }

  private readonly onMessage = (message: MessageEvent<string>) => {
    this.running?.settle();
    const event = parse(message);
    if (event === undefined) {
      this.resync();
    } else if (isControl(event)) {
      this.onControl(event);
    } else {
      this.runEffect(event);
    }
  };

  private onControl(event: ControlEvent) {
    switch (event.event) {
      case "live.resync":
        this.resync();
        return;
      case "live.session-ended":
        this.stop();
        this.options.onSessionEnded(event.data.reason);
        return;
      case "live.access-changed":
        this.stop();
        this.options.onAccessChanged();
        return;
      default:
        event satisfies never;
    }
  }

  private readonly refreshQueries = (filters?: InvalidateQueryFilters) => {
    void refresh(this.options.queryClient, filters);
  };

  private resync() {
    this.refreshQueries();
  }

  private runEffect<N extends ProductEventName>(event: { event: N; data: PayloadOf<N> }) {
    const effect: LiveEffect<N> = table[event.event];
    effect(event.data, { queryClient: this.options.queryClient, refresh: this.refreshQueries });
  }

  private readonly onVisibilityChange = () => {
    if (document.visibilityState === "hidden") {
      this.releaseTimer ??= setTimeout(this.release, hiddenRelease);
      return;
    }
    clearTimeout(this.releaseTimer);
    this.releaseTimer = undefined;
    if (this.state === "paused") {
      this.open();
    }
  };

  private readonly release = () => {
    this.releaseTimer = undefined;
    this.close();
    this.setState("paused");
  };
}

function parse(message: MessageEvent<string>): ParsedEvent | undefined {
  try {
    const data: unknown = JSON.parse(message.data);
    return LiveEvent.parse({ event: message.type, data });
  } catch (error) {
    reportError(error);
    return undefined;
  }
}

export function createLiveStream(options: Options): LiveStream {
  const stream = new Stream(options);
  return {
    connect: () => stream.connect(),
    stop: () => {
      stream.stop();
    },
    getState: () => stream.state,
    subscribe: (listener) => {
      stream.listeners.add(listener);
      return () => {
        stream.listeners.delete(listener);
      };
    },
  };
}
