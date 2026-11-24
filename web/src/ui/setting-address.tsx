import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import { useState } from "react";

import type { SettingListenAt, SettingListenAtHost } from "../api/gen/model";
import { Segmented } from "./segmented";
import {
  AmountInput,
  type ControlProps,
  type Draft,
  type Kind,
  SaveButton,
  controlRow,
  defaultHint,
  inputLook,
  labelLook,
  onSubmit,
  unitLook,
} from "./setting-kind";

type Parts = { readonly address: string; readonly port: string };

/** Splits text such as 0.0.0.0:6881 or [::]:6881 at its last colon. */
function partsOf(text: string): Parts {
  const at = text.lastIndexOf(":");
  const address = text.slice(0, at);
  return {
    address: address.startsWith("[") && address.endsWith("]") ? address.slice(1, -1) : address,
    port: text.slice(at + 1),
  };
}

const textOf = ({ address, port }: Parts): string =>
  address.includes(":") ? `[${address}]:${port}` : `${address}:${port}`;

const highestPort = 65_535;

function parseAddress(parts: Parts, example: string): Draft<"address"> {
  const address = parts.address.trim();
  const port = parts.port.trim();
  if (address === "") {
    return { refused: `An address and port, such as ${example}.` };
  }
  if (!/^\d+$/u.test(port) || Number(port) < 1 || Number(port) > highestPort) {
    return { refused: "A port between 1 and 65535." };
  }
  return { value: textOf({ address, port: String(Number(port)) }) };
}

function PartInput({
  label,
  value,
  invalid,
  disabled,
  inputMode,
  className,
  onChange,
}: {
  label: string;
  value: string;
  invalid: boolean;
  disabled: boolean;
  inputMode: "numeric" | "text";
  className: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field.Root invalid={invalid} disabled={disabled} className="flex items-center gap-2">
      <Field.Label className={unitLook}>{label}</Field.Label>
      <Input
        inputMode={inputMode}
        autoComplete="off"
        spellCheck={false}
        value={value}
        onValueChange={onChange}
        className={`${className} ${inputLook}`}
      />
    </Field.Root>
  );
}

function AddressControl({
  field,
  busy,
  invalid,
  disabled,
  submit,
  onEdit,
}: ControlProps<"address">) {
  const [parts, setParts] = useState(() => partsOf(field.value));
  const edit = (next: Parts) => {
    setParts(next);
    onEdit();
  };
  return (
    <form
      onSubmit={onSubmit(() => {
        submit(parseAddress(parts, field.default));
      })}
    >
      <fieldset>
        <legend className={`mb-1.5 ${labelLook}`}>{field.label}</legend>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
          <PartInput
            label="Address"
            value={parts.address}
            invalid={invalid}
            disabled={disabled}
            inputMode="text"
            className="w-40"
            onChange={(address) => {
              edit({ ...parts, address });
            }}
          />
          <PartInput
            label="Port"
            value={parts.port}
            invalid={invalid}
            disabled={disabled}
            inputMode="numeric"
            className="w-20"
            onChange={(port) => {
              edit({ ...parts, port });
            }}
          />
          <SaveButton shown={textOf(parts) !== field.value} busy={busy} />
        </div>
      </fieldset>
    </form>
  );
}

export const addressKind: Kind<"address"> = {
  words: (field) => field.value,
  hints: (field) => [defaultHint(field, field.default)],
  Control: AddressControl,
};

const portRange = "Between 1 and 65535.";
function parsePort(amount: string): { value: number } | { refused: string } {
  if (!/^\d+$/u.test(amount.trim())) {
    return { refused: portRange };
  }
  const port = Number(amount);
  return port < 1 || port > highestPort ? { refused: portRange } : { value: port };
}

const listenHosts: readonly { value: SettingListenAtHost; label: string }[] = [
  { value: "all", label: "All networks" },
  { value: "local", label: "This machine only" },
];

/** The host part of at's address, which a save keeps. */
const hostOf = (at: SettingListenAt): string => at.address.slice(0, at.address.lastIndexOf(":"));

const hostAddress = (host: SettingListenAtHost, saved: SettingListenAt): string =>
  host === "all" ? "[::]" : host === "local" ? "[::1]" : hostOf(saved);

const listenWords = (at: SettingListenAt): string =>
  at.host === "other"
    ? at.address
    : `${listenHosts.find((option) => option.value === at.host)?.label ?? at.host}, port ${String(at.port)}`;

/** Shown once All networks is chosen in place of this machine only, before the save. */
function AllNetworksWarning() {
  return (
    <p className="text-meta text-ink-2 in-data-[density=touch]:text-body-touch">
      Other devices on your network can open SlipStream. Over IPv6, your router may also let devices
      outside your home in.
    </p>
  );
}

/** The host choice, or a host set by hand in config.yaml, which shows read-only. */
function HostPicker({
  label,
  saved,
  host,
  disabled,
  onChange,
}: {
  label: string;
  saved: SettingListenAt;
  host: SettingListenAtHost;
  disabled: boolean;
  onChange: (host: SettingListenAtHost) => void;
}) {
  return (
    <div className="flex flex-col items-start gap-1.5">
      <span aria-hidden={saved.host !== "other"} className={labelLook}>
        {label}
      </span>
      {saved.host === "other" ? (
        <span className={unitLook}>{hostOf(saved)}</span>
      ) : (
        <Segmented
          label={label}
          options={listenHosts}
          value={host}
          disabled={disabled}
          onValueChange={onChange}
        />
      )}
    </div>
  );
}

/** Where the web server listens and its port. */
function ListenControl({ field, busy, invalid, disabled, submit, onEdit }: ControlProps<"listen">) {
  const saved = field.value;
  const [host, setHost] = useState(saved.host);
  const [amount, setAmount] = useState(String(saved.port));
  const edited =
    <T,>(set: (value: T) => void) =>
    (value: T) => {
      set(value);
      onEdit();
    };
  return (
    <form
      onSubmit={onSubmit(() => {
        const port = parsePort(amount);
        submit(
          "refused" in port ? port : { value: `${hostAddress(host, saved)}:${String(port.value)}` },
        );
      })}
      className="flex flex-col items-start gap-3"
    >
      <HostPicker
        label={field.label}
        saved={saved}
        host={host}
        disabled={disabled || busy}
        onChange={edited(setHost)}
      />
      <div className={controlRow}>
        <AmountInput
          label="Port"
          value={amount}
          invalid={invalid}
          disabled={disabled}
          inputMode="numeric"
          onChange={edited(setAmount)}
        />
        <SaveButton
          shown={host !== saved.host || amount.trim() !== String(saved.port)}
          busy={busy}
        />
      </div>
      {host === "all" && saved.host === "local" && <AllNetworksWarning />}
    </form>
  );
}

export const listenKind: Kind<"listen"> = {
  words: (field) => listenWords(field.value),
  hints: (field) => [defaultHint(field, listenWords(field.default))],
  Control: ListenControl,
};
