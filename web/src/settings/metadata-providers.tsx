import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type {
  MetadataAttribution,
  MetadataAttributionLogo,
  MetadataProvider,
  MetadataProviderKey,
} from "../api/gen/model";
import {
  clearMetadataKey,
  testMetadataProvider,
  useListMetadataProvidersSuspense,
} from "../api/gen/query/metadata/metadata";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Section } from "../ui/section";
import { toast } from "../ui/toast";
import { KeyDialog } from "./metadata-key-dialog";
import tmdbLogo from "./tmdb-logo.svg";

const logos: Record<MetadataAttributionLogo, { src: string; name: string }> = {
  tmdb: { src: tmdbLogo, name: "TMDB" },
};

const setOn = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "long" });

const keyWords: Record<MetadataProviderKey, (provider: MetadataProvider) => string> = {
  none: () => "No key",
  bundled: () => "SlipStream's key",
  yours: ({ keySetAt }) =>
    keySetAt === null ? "Your key" : `Your key, set ${setOn.format(new Date(keySetAt))}`,
};

function Attribution({ attribution }: { attribution: MetadataAttribution }) {
  const logo = logos[attribution.logo];
  return (
    <p className="flex flex-wrap items-center gap-3 text-meta text-ink-3">
      <img src={logo.src} alt={logo.name} className="h-3" />
      {attribution.text}
    </p>
  );
}

function RemoveKey({ provider }: { provider: MetadataProvider }) {
  const [asking, setAsking] = useState(false);
  return (
    <>
      <Button
        size="small"
        onClick={() => {
          setAsking(true);
        }}
      >
        Remove key
      </Button>
      <ConfirmDialog
        title={`Remove your ${provider.label} key?`}
        confirmLabel="Remove key"
        look="fail"
        open={asking}
        onOpenChange={setAsking}
        onConfirm={() => clearMetadataKey(provider.id)}
      >
        <p>
          Calls to {provider.label} go back to SlipStream&apos;s key, or stop if this build has
          none.
        </p>
      </ConfirmDialog>
    </>
  );
}

function Test({ provider }: { provider: MetadataProvider }) {
  const [testing, setTesting] = useState(false);
  const test = async () => {
    setTesting(true);
    try {
      toast((await testMetadataProvider(provider.id)).summary);
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setTesting(false);
    }
  };
  return (
    <Button size="small" busy={testing} onClick={() => void test()}>
      Test
    </Button>
  );
}

function ProviderActions({ provider }: { provider: MetadataProvider }) {
  const [setting, setSetting] = useState(false);
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button
        size="small"
        onClick={() => {
          setSetting(true);
        }}
      >
        {provider.key === "yours" ? "Change key" : "Set key"}
      </Button>
      {provider.key === "yours" && <RemoveKey provider={provider} />}
      <Test provider={provider} />
      <a
        href={provider.keyPage}
        target="_blank"
        rel="noreferrer"
        className="ml-1 text-meta text-ink-2 underline hover:text-ink-1"
      >
        Get a {provider.label} key
      </a>
      <KeyDialog provider={provider} open={setting} onOpenChange={setSetting} />
    </div>
  );
}

function ProviderCard({ provider }: { provider: MetadataProvider }) {
  return (
    <article
      aria-label={provider.label}
      className="flex flex-col gap-3 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset"
    >
      <header className="flex flex-wrap items-baseline justify-between gap-2">
        <h3 className="font-semibold text-ink-1">{provider.label}</h3>
        <span className="text-meta text-ink-3">{keyWords[provider.key](provider)}</span>
      </header>
      <p className="text-ink-2">{provider.summary}</p>
      <ProviderActions provider={provider} />
      {provider.attribution !== null && <Attribution attribution={provider.attribution} />}
    </article>
  );
}

/** Each metadata provider's status, the key it is called with, and the attribution its terms require. */
export function MetadataProviders() {
  const { providers } = useListMetadataProvidersSuspense().data;
  return (
    <Section title="Providers">
      <div className="flex flex-col gap-3">
        {providers.map((provider) => (
          <ProviderCard key={provider.id} provider={provider} />
        ))}
      </div>
    </Section>
  );
}
