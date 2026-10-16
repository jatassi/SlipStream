import { useRouter } from "@tanstack/react-router";

import type { IdentitySettings, IdentitySettingsPatch } from "../api/gen/model";
import {
  updateIdentitySettings,
  useGetIdentitySettingsSuspense,
} from "../api/gen/query/identity/identity";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

type Key = keyof IdentitySettings & keyof IdentitySettingsPatch;

const groups: readonly { readonly title: string; readonly keys: readonly Key[] }[] = [
  { title: "Wrong PINs", keys: ["pinWaitAfter", "pinWait", "pinOffAfter"] },
  { title: "Failed attempts from one address", keys: ["addressFailures", "addressWindow"] },
  { title: "Sessions and re-confirm", keys: ["sessionLifetime", "reconfirmWindow"] },
  {
    title: "Link lifetimes",
    keys: ["signupLinkLifetime", "resetLinkLifetime", "emailLinkLifetime"],
  },
];

export function SignInSection() {
  const settings = useGetIdentitySettingsSuspense().data;
  const router = useRouter();
  const save = async (key: Key, value: number | null) => {
    await updateIdentitySettings({ [key]: value });
    await router.invalidate();
  };
  return groups.map(({ title, keys }) => (
    <Section key={title} title={title}>
      <SettingGroup>
        {keys.map((key) => (
          <SettingField key={key} field={settings[key]} onSave={(value) => save(key, value)} />
        ))}
      </SettingGroup>
    </Section>
  ));
}
