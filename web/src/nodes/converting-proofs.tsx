import type {
  ConversionsNodeConverting,
  ConversionsProof,
  ConversionsProofApi,
  ConversionsProofClass,
  ConversionsShape,
} from "../api/gen/model";
import { type Clock, useClock } from "../ui/clock";
import { Defined, type Term } from "../ui/defined";
import { Details } from "../ui/details";
import { whenWords } from "../ui/time-words";

type Api = NonNullable<ConversionsProofApi>;

/** The hardware a Node converts with, as its maker names it. */
const apiNames: { readonly [A in Api]: string } = {
  qsv: "Intel Quick Sync",
  vaapi: "VA-API",
  cuda: "NVIDIA NVENC",
  videotoolbox: "Apple VideoToolbox",
};

const apiShort: { readonly [A in Api]: string } = {
  qsv: "Quick Sync",
  vaapi: "VA-API",
  cuda: "NVENC",
  videotoolbox: "VideoToolbox",
};

const classWords: { readonly [C in ConversionsProofClass]: string } = {
  h264_8: "H.264",
  hevc_8: "HEVC",
  hevc_10: "10-bit HEVC",
  av1_8: "AV1",
  av1_10: "10-bit AV1",
  hdr10: "HDR10",
  hlg: "HLG",
  dv5: "Dolby Vision Profile 5",
  dv81: "Dolby Vision Profile 8.1",
  decode_sdr: "MPEG-2",
  decode_hdr10: "10-bit H.264 HDR10",
};

const codecWords: { readonly [C in ConversionsShape["codec"]]: string } = {
  h264: "H.264",
  hevc: "HEVC",
  av1: "AV1",
};

const rangeWords: { readonly [R in ConversionsShape["range"]]: string } = {
  sdr: "SDR ",
  hdr10: "HDR10 ",
  source: "",
};

const toneMapping: Term = {
  word: "tone mapping",
  definition: "Turns HDR video into SDR so it looks right on a screen without HDR.",
};

/** "HDR10 to SDR H.264", or "Dolby Vision Profile 5 to 10-bit HEVC with Dolby Vision". */
function trialWords({ class: from, shape }: ConversionsProof): string {
  const depth = shape.depth === 10 ? "10-bit " : "";
  const vision = shape.keepDolbyVision ? " with Dolby Vision" : "";
  return `${classWords[from]} to ${rangeWords[shape.range]}${depth}${codecWords[shape.codec]}${vision}`;
}

function outcomeWords(proof: ConversionsProof, clock: Clock): string {
  if (proof.lostAt !== null) {
    return `failed after a driver change ${whenWords(proof.lostAt, "at", clock)}.`;
  }
  if (proof.failedAt !== null) {
    return `failed ${whenWords(proof.failedAt, "at", clock)}.`;
  }
  if (proof.provenAt === null) {
    return "not tested yet.";
  }
  return `${proof.speed.toFixed(1)} times real time.`;
}

function ProofLine({ proof }: { proof: ConversionsProof }) {
  const clock = useClock();
  const device = proof.device === "" ? "" : ` on ${proof.device}`;
  const api = proof.api === null ? "the processor" : apiShort[proof.api];
  return (
    <li className="flex flex-col gap-1">
      <span className="text-ink-1">
        {trialWords(proof)}: {api}
        {device}
        {proof.path === "hardware_opencl" && (
          <>
            , OpenCL <Defined term={toneMapping}>{(word) => word}</Defined>
          </>
        )}
        , {outcomeWords(proof, clock)}
      </span>
      {proof.error !== "" && (
        <pre className="max-w-full overflow-x-auto text-meta whitespace-pre-wrap text-fail-dim">
          {proof.error}
        </pre>
      )}
    </li>
  );
}

/** The Node's one sentence on what it converts with, as the spec words it. */
export function convertsWith(name: string, converting: ConversionsNodeConverting): string {
  const { sentence, processor, proofs } = converting;
  if (sentence.api !== null) {
    return `${name} converts with ${apiNames[sentence.api]}.`;
  }
  if (sentence.processor) {
    return `${name} converts with its processor.`;
  }
  if (!processor) {
    return `${name} can't convert anything. Turn on Use the processor when no GPU can, or give the Converting Role to a Node with a GPU.`;
  }
  return proofs.length === 0
    ? `${name} hasn't run its test runs yet.`
    : `${name} can't convert anything. Every test run failed. Rerun the test runs, or check the ffmpeg path under Details.`;
}

/** Each test run's trial, hardware, device and outcome, with ffmpeg's last lines when it failed. */
export function ConvertingProofs({ proofs }: { proofs: readonly ConversionsProof[] }) {
  return (
    proofs.length > 0 && (
      <Details label="Test runs">
        <ul className="mt-1 flex flex-col gap-3">
          {proofs.map((proof) => (
            <ProofLine key={`${proof.class} ${proof.path} ${proof.api ?? ""}`} proof={proof} />
          ))}
        </ul>
      </Details>
    )
  );
}
