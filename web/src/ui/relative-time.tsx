import { useClock } from "./clock";
import { dataFont } from "./data";
import { HoverNote } from "./defined";
import { type ValueForm, momentWords, placeOf, timeForms } from "./time-words";

type Shared = {
  /** The instant, as the API gives it. */
  readonly at: string;
  /** An IANA zone when the time answers to a setting stated in it, such as the Tasks window. */
  readonly zone?: string | undefined;
};

export type TimeProps = Shared &
  (
    | {
        readonly form: "clock" | "time" | "stamp";
        /** Mono, for a value someone compares down a column (Copy or Compare Rule). */
        readonly data?: boolean;
      }
    | { readonly form: "due" | "countdown" | "age"; readonly data?: never }
  );

function useTime({ at, zone, form, data = false }: TimeProps) {
  const clock = useClock(zone);
  return {
    words: timeForms[form satisfies ValueForm](placeOf(at, clock)),
    moment: momentWords(at, clock).join("\n"),
    className: data ? `${dataFont} text-data` : "tabular-nums",
  };
}

/** A time standing on its own, whose whole moment opens on hover, tap and Enter. */
export function RelativeTime(props: TimeProps) {
  const { words, moment, className } = useTime(props);
  return (
    <HoverNote name={words} note={moment}>
      <time dateTime={props.at} className={className}>
        {words}
      </time>
    </HoverNote>
  );
}

/** A time inside a link or button, where nothing can open, with its whole moment as its title. */
export function PlainTime(props: TimeProps) {
  const { words, moment, className } = useTime(props);
  return (
    <time dateTime={props.at} title={moment} className={className}>
      {words}
    </time>
  );
}
