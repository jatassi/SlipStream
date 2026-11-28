import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { Details } from "../ui/details";
import { RadioGroup } from "../ui/radio";
import { Segmented } from "../ui/segmented";
import { Select } from "../ui/select";
import { Switch } from "../ui/switch";
import {
  type ConversionDraft,
  depthOf,
  heightOf,
  kbpsProblem,
  keepsDolbyVision,
} from "./conversion-draft";
import { audios, codecs, containers, curves, depths, heights, ranges } from "./conversion-words";
import { DraftField } from "./field";

export type FieldsProps = {
  readonly conversion: ConversionDraft;
  readonly set: (change: Partial<ConversionDraft>) => void;
};

const meta = "text-meta text-ink-3 in-data-[density=touch]:text-body-touch";

const anyNode = "any";

function RunOn({ conversion, set }: FieldsProps) {
  const { nodes } = useListNodesSuspense().data;
  const options = [
    { value: anyNode, label: "Any Node that can" },
    ...nodes
      .filter((node) => node.roles.some((view) => view.role === "converting"))
      .map((node) => ({ value: node.id, label: node.name })),
  ];
  return (
    <Select
      label="Run on"
      options={options}
      value={conversion.runOnNodeId ?? anyNode}
      onValueChange={(value) => {
        set({ runOnNodeId: value === anyNode ? null : value });
      }}
    />
  );
}

function Size({ conversion, set }: FieldsProps) {
  return (
    <>
      <div className="flex flex-col gap-1.5">
        <Segmented
          label="Largest resolution"
          options={heights}
          value={String(conversion.maxHeight)}
          onValueChange={(height) => {
            set({ maxHeight: heightOf(height) });
          }}
        />
        <p className={meta}>A smaller source keeps its own size.</p>
      </div>
      <div className="w-60">
        <DraftField
          label="Video bitrate in kbps"
          value={conversion.kbps}
          inputMode="decimal"
          error={kbpsProblem(conversion.kbps)}
          description="At the largest resolution. A smaller source gets less, and never more than its own."
          onChange={(kbps) => {
            set({ kbps });
          }}
        />
      </div>
    </>
  );
}

function Picture({ conversion, set }: FieldsProps) {
  return (
    <>
      <Segmented
        label="Bit depth"
        options={depths}
        value={String(conversion.depth)}
        disabled={conversion.codec === "h264"}
        onValueChange={(depth) => {
          set({ depth: depthOf(depth) });
        }}
      />
      <RadioGroup
        label="Dynamic range"
        options={ranges}
        value={conversion.range}
        onValueChange={(range) => {
          set({ range });
        }}
      />
      <div className="flex flex-col gap-1.5">
        <Switch
          label="Keep Dolby Vision"
          checked={conversion.keepDolbyVision}
          disabled={!keepsDolbyVision(conversion)}
          onCheckedChange={(keepDolbyVision) => {
            set({ keepDolbyVision });
          }}
        />
        <p className={meta}>
          Only the processor can write Dolby Vision. Every Conversion for this Target runs on the
          processor, which is much slower than a GPU. Profile 7 sources come out as HDR10.
        </p>
      </div>
    </>
  );
}

function Sound({ conversion, set }: FieldsProps) {
  return (
    <>
      <RadioGroup
        label="Audio"
        options={audios}
        value={conversion.audio}
        onValueChange={(audio) => {
          set({ audio });
        }}
      />
      <Switch
        label="Drop commentary tracks"
        checked={conversion.dropCommentary}
        onCheckedChange={(dropCommentary) => {
          set({ dropCommentary });
        }}
      />
      <Segmented
        label="Container"
        options={containers}
        value={conversion.container}
        onValueChange={(container) => {
          set({ container });
        }}
      />
    </>
  );
}

export function ConversionFields(props: FieldsProps) {
  const { conversion, set } = props;
  return (
    <div className="flex flex-col gap-5">
      <Segmented
        label="Video codec"
        options={codecs}
        value={conversion.codec}
        onValueChange={(codec) => {
          set({ codec });
        }}
      />
      <Size {...props} />
      <Picture {...props} />
      <Sound {...props} />
      <Details fields>
        <div className="flex max-w-[400px] flex-col gap-4 pt-2">
          <Select
            label="Tone mapping curve"
            options={curves}
            value={conversion.curve}
            onValueChange={(curve) => {
              set({ curve });
            }}
          />
          <RunOn {...props} />
        </div>
      </Details>
    </div>
  );
}
