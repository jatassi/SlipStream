export type SourceHandlers = {
  readonly open: () => void;
  readonly message: (message: MessageEvent<string>) => void;
  readonly error: (closed: boolean) => void;
};

export type Source = { readonly close: () => void };

/**
 * The app's one EventSource. Each stream opens its server-sent events through
 * it and listens for the event names it declares. A closed source reports
 * nothing more, so a stream never hears from a connection it let go.
 */
export function openSource(url: string, names: readonly string[], on: SourceHandlers): Source {
  const source = new EventSource(url);
  let closed = false;
  source.addEventListener("open", () => {
    if (!closed) {
      on.open();
    }
  });
  source.addEventListener("error", () => {
    if (!closed) {
      on.error(source.readyState === EventSource.CLOSED);
    }
  });
  const onMessage = (message: MessageEvent<string>) => {
    if (!closed) {
      on.message(message);
    }
  };
  for (const name of names) {
    source.addEventListener(name, onMessage);
  }
  return {
    close: () => {
      closed = true;
      source.close();
    },
  };
}
