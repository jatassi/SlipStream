import { useRouter } from "@tanstack/react-router";

import type {
  ReachabilityListenView,
  ReachabilityMappingView,
  ReachabilityNodeNetwork,
  ReachabilityRuleViewPolicy,
  SettingField as Field,
} from "../api/gen/model";
import {
  patchNodeNetworkSettings,
  useGetNodeNetworkSettingsSuspense,
  useGetNodeNetworkSuspense,
} from "../api/gen/query/reachability/reachability";
import { useClock } from "../ui/clock";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { whenWords } from "../ui/time-words";
import { familyNames } from "./words";

const policies: { readonly [P in ReachabilityRuleViewPolicy]: string } = {
  allows: "Allows",
  blocks: "Blocks",
  unknown: "SlipStream can't read it",
};

function listenWords(listener: ReachabilityListenView): string {
  const families = (["ipv4", "ipv6"] as const).flatMap((family) => {
    const bound = listener[family];
    if (bound === "no") {
      return [];
    }
    return bound === "yes" ? [familyNames[family]] : [`${familyNames[family]} unknown`];
  });
  return [
    listener.program === "" ? "A program SlipStream can't name" : listener.program,
    families.length === 0 ? "not listening" : families.join(" and "),
    ...(listener.loopback ? ["this machine only"] : []),
  ].join(" · ");
}

function MappingFact({ mapping }: { mapping: ReachabilityMappingView }) {
  const clock = useClock();
  return (
    <Fact label={`Port ${mapping.external}`}>
      {`Opened to port ${mapping.internal}`}
      {mapping.permanent
        ? ". The router took it only with no end date, so it stays until you delete it on the router."
        : mapping.expires === null
          ? ""
          : ` ${whenWords(mapping.expires, "until", clock)}`}
    </Fact>
  );
}

function Addresses({ network }: { network: ReachabilityNodeNetwork }) {
  return (
    <Facts look="well">
      {network.interfaces.map((iface) => (
        <Fact key={iface.name} label={iface.tunnel ? `${iface.name} (tunnel)` : iface.name}>
          <Data>{iface.addresses.join(", ")}</Data>
        </Fact>
      ))}
      {network.gateway !== null && (
        <Fact label="Router">
          <Data>{network.gateway.address}</Data>
          {network.gateway.wan !== undefined && (
            <>
              , internet side <Data>{network.gateway.wan}</Data>
            </>
          )}
        </Fact>
      )}
    </Facts>
  );
}

function Ports({ network }: { network: ReachabilityNodeNetwork }) {
  return (
    <>
      {network.listeners.length > 0 && (
        <Facts look="well">
          {network.listeners.map((listener) => (
            <Fact key={listener.port} label={`Port ${listener.port}`}>
              {listenWords(listener)}
            </Fact>
          ))}
        </Facts>
      )}
      {network.firewall.length > 0 && (
        <Facts look="well">
          {network.firewall.map((rule) => (
            <Fact
              key={`${rule.port}:${rule.family}`}
              label={`Firewall, port ${rule.port} over ${familyNames[rule.family]}`}
            >
              {policies[rule.policy]}
            </Fact>
          ))}
        </Facts>
      )}
      {network.mappings.length > 0 && (
        <Facts look="well">
          {network.mappings.map((mapping) => (
            <MappingFact key={mapping.external} mapping={mapping} />
          ))}
        </Facts>
      )}
    </>
  );
}

function Machinery({ network }: { network: ReachabilityNodeNetwork }) {
  return (
    <Details>
      <div className="mt-2 flex flex-col gap-3">
        <Addresses network={network} />
        <Ports network={network} />
      </div>
    </Details>
  );
}

function Read({ network }: { network: ReachabilityNodeNetwork }) {
  const clock = useClock();
  if (network.now) {
    return null;
  }
  return (
    <p className="text-ink-2">
      {network.observedAt === null
        ? `SlipStream hasn't read ${network.name}'s network yet.`
        : `${network.name} is offline. This is its network ${whenWords(network.observedAt, "at", clock)}.`}
    </p>
  );
}

function useSave(nodeId: string) {
  const router = useRouter();
  return (key: string) => async (value: unknown) => {
    await patchNodeNetworkSettings(nodeId, { values: { [key]: value } });
    await router.invalidate();
  };
}

/**
 * A Node's Network section: how it connects to the internet in no signal
 * colour, its two Remote access settings, and the addresses, listeners,
 * firewall answers, router and mappings under Details.
 */
export function NetworkSection({ nodeId }: { nodeId: string }) {
  const network = useGetNodeNetworkSuspense(nodeId).data;
  const { settings } = useGetNodeNetworkSettingsSuspense(nodeId).data;
  const save = useSave(nodeId);
  return (
    <div className="flex max-w-[640px] flex-col gap-4">
      <p className="text-ink-1">{network.line}</p>
      <Read network={network} />
      <SettingGroup>
        {settings.map((field: Field) => (
          <SettingField key={field.key} field={field} onSave={save(field.key)} />
        ))}
      </SettingGroup>
      <Machinery network={network} />
    </div>
  );
}
