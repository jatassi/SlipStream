import { LogEvent } from "../api/gen/zod/slipStream.schemas.zod";
import { nodeLogsStreamUrl } from "../api/urls";
import { type Source, openSource } from "./source";

export type LogStreamEvent = ReturnType<typeof LogEvent.parse>;

const eventNames = LogEvent.options.flatMap((branch) => branch.shape.event.options);
const hiddenRelease = 10_000;
const reopenDelays = [1000, 2000, 5000, 10_000, 30_000] as const;

class LogStream {
  private source: Source | undefined;
  private after: number | undefined;
  private queued: LogStreamEvent[] = [];
  private frame: number | undefined;
  private reopens = 0;
  private reopenTimer: ReturnType<typeof setTimeout> | undefined;
  private releaseTimer: ReturnType<typeof setTimeout> | undefined;

  private readonly nodeId: string;
  private readonly onEvents: (events: readonly LogStreamEvent[]) => void;

  constructor(nodeId: string, onEvents: (events: readonly LogStreamEvent[]) => void) {
    this.nodeId = nodeId;
    this.onEvents = onEvents;
    document.addEventListener("visibilitychange", this.onVisibilityChange);
    this.open();
  }

  close() {
    document.removeEventListener("visibilitychange", this.onVisibilityChange);
    clearTimeout(this.releaseTimer);
    if (this.frame !== undefined) {
      cancelAnimationFrame(this.frame);
    }
    this.shut();
  }

  private readonly open = () => {
    this.reopenTimer = undefined;
    this.source = openSource(nodeLogsStreamUrl(this.nodeId, this.after), eventNames, {
      open: () => {
        this.reopens = 0;
      },
      message: this.receive,
      error: (closed) => {
        if (closed) {
          this.reopen();
        }
      },
    });
  };

  private shut() {
    this.source?.close();
    this.source = undefined;
    clearTimeout(this.reopenTimer);
    this.reopenTimer = undefined;
  }

  private reopen() {
    this.shut();
    const delay = reopenDelays[Math.min(this.reopens, reopenDelays.length - 1)];
    this.reopens += 1;
    this.reopenTimer = setTimeout(this.open, delay);
  }

  private readonly receive = (message: MessageEvent<string>) => {
    const data: unknown = JSON.parse(message.data);
    const event = LogEvent.parse({ event: message.type, data });
    if (event.event === "log.record") {
      this.after = event.data.seq;
    }
    this.queued.push(event);
    this.frame ??= requestAnimationFrame(this.flush);
  };

  private readonly flush = () => {
    this.frame = undefined;
    const events = this.queued;
    this.queued = [];
    this.onEvents(events);
  };

  private readonly onVisibilityChange = () => {
    if (document.visibilityState === "hidden") {
      this.releaseTimer ??= setTimeout(this.release, hiddenRelease);
      return;
    }
    clearTimeout(this.releaseTimer);
    this.releaseTimer = undefined;
    if (this.source === undefined && this.reopenTimer === undefined) {
      this.open();
    }
  };

  private readonly release = () => {
    this.releaseTimer = undefined;
    this.shut();
  };
}

/**
 * One Node's log, as events in arrival order, handed over once per animation
 * frame so a busy Node redraws the list once a frame. After an error or a
 * release while the tab was hidden, it reopens after the last record it had.
 */
export function openLogStream(
  nodeId: string,
  onEvents: (events: readonly LogStreamEvent[]) => void,
): { readonly close: () => void } {
  const stream = new LogStream(nodeId, onEvents);
  return {
    close: () => {
      stream.close();
    },
  };
}
