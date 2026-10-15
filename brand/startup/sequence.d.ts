type Span = readonly [start: number, end: number];

export declare const DURATION: number;

export declare const PHASES: {
  readonly gridIn: Span;
  readonly popA: number;
  readonly popB: number;
  readonly launch: number;
  readonly depart: number;
  readonly arrive: number;
  readonly tau: number;
  readonly bloom: Span;
  readonly ack: Span;
  readonly lockup: Span;
  readonly wipe: Span;
  readonly settle: Span;
  readonly exit: Span;
};

export declare const STILL: number;

export declare function build(stage: HTMLElement): Promise<void>;

export declare function render(t: number, stage: HTMLElement): void;

export declare function liftMark(stage: HTMLElement): HTMLDivElement;
