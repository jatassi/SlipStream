import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import { RssSyncSection } from '@/components/settings'

export function RssSyncPage() {
  const back = usePushBack()

  return (
    <Screen title="RSS Sync" back={back}>
      <RssSyncSection />
    </Screen>
  )
}
