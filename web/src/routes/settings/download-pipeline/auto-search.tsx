import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import { AutoSearchSection } from '@/components/settings'

export function AutoSearchPage() {
  const back = usePushBack()

  return (
    <Screen title="Auto Search" back={back}>
      <AutoSearchSection />
    </Screen>
  )
}
