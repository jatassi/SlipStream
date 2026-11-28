import type { TargetsCause, TargetsCauseClass, TargetsCauseWhy } from "../api/gen/model";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { RelativeTime } from "../ui/relative-time";
import { SignalLine } from "../ui/signal-line";
import { worksNounOf } from "../wanted/wanted-words";

const classVideo: { readonly [C in TargetsCauseClass]: string } = {
  h264_8: "8-bit H.264",
  hevc_8: "8-bit HEVC",
  hevc_10: "10-bit HEVC",
  av1_8: "8-bit AV1",
  av1_10: "10-bit AV1",
  hdr10: "HDR10",
  hlg: "HLG",
  dv5: "Dolby Vision Profile 5",
  dv81: "Dolby Vision Profile 8.1",
  decode_sdr: "processor-decoded",
  decode_hdr10: "processor-decoded HDR10",
};

const hdrClasses: ReadonlySet<TargetsCauseClass> = new Set([
  "hdr10",
  "hlg",
  "dv5",
  "dv81",
  "decode_hdr10",
]);

/** What a cause's Conversions need, such as "Dolby Vision Profile 5 tone mapping". */
function needOf(cause: TargetsCause): string {
  const toSdr = cause.shape.split(".")[2] === "sdr";
  return hdrClasses.has(cause.class) && toSdr
    ? `${classVideo[cause.class]} tone mapping`
    : `${classVideo[cause.class]} video converted`;
}

const whyWords: {
  readonly [W in TargetsCauseWhy]: (need: string, node: string, target: string) => string;
} = {
  no_role: () => "No Node has the Converting Role. Give one the Role on the Nodes page.",
  none_online: () => "No Node with the Converting Role is online. They start when one is back.",
  no_holder: () =>
    "No Node with the Converting Role is ready to convert. They start when one is ready.",
  lapsed: (_need, node) => `${node} is offline. They start when ${node} is back.`,
  runs_elsewhere: (_need, node, target) =>
    `${target} runs only on ${node}, which can't convert them now. They start when ${node} can, or change Run on for ${target}.`,
  pinned_elsewhere: (_need, node) =>
    `They're set to run again on ${node}, which can't convert them now. They start when ${node} can.`,
  failed_here: (need, node) =>
    `They need ${need}, and they already failed on ${node}. They start when another Node can.`,
  untested: (need, node) =>
    `They need ${need}, and ${node} hasn't tested it yet. They start after ${node}'s test run.`,
  test_failed: (need, node) =>
    `They need ${need}, and ${node}'s test run for it failed. Rerun the test run on ${node}.`,
  retesting: (need, node) =>
    `They need ${need}, and ${node} is rerunning its test runs. They start once they pass.`,
  proof_lost: (need, node) =>
    `They need ${need}, and ${node}'s test run failed after its driver changed. Rerun the test run on ${node}.`,
  processor_off: (need, node) =>
    `They need ${need}, which only the processor can do, and Use the processor when no GPU can is off on ${node}.`,
};

/** One cause's sentence, such as "No Node can convert 3 movies. They need …". */
function causeWords(cause: TargetsCause, target: string, works: string): string {
  const node = cause.node?.name ?? "the Node";
  return `No Node can convert ${String(cause.count)} ${works}. ${whyWords[cause.why](needOf(cause), node, target)}`;
}

/** One warning sentence per open cause, each with when the Admin was notified. */
export function CauseLines({
  causes,
  target,
  contentType,
}: {
  causes: readonly TargetsCause[];
  target: string;
  contentType: string;
}) {
  const type = useListContentTypesSuspense().data.contentTypes.find(
    (candidate) => candidate.id === contentType,
  );
  const noun = worksNounOf(type);
  return (
    <ul aria-label={`Why ${target} can't convert`} className="flex flex-col gap-1.5 pt-2">
      {causes.map((cause) => (
        <li key={`${cause.class}:${cause.shape}:${cause.why}`}>
          <SignalLine tone="warn">
            {causeWords(cause, target, cause.count === 1 ? noun.one : noun.many)} Notified{" "}
            <RelativeTime at={cause.notifiedAt} form="age" />.
          </SignalLine>
        </li>
      ))}
    </ul>
  );
}
