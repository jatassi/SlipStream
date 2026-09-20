import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import { FileNamingSection } from '@/components/settings'

export function FileNamingPage() {
  const back = usePushBack()

  return (
    <Screen title="Import & Naming" back={back}>
      <FileNamingSection />
    </Screen>
  )
}
