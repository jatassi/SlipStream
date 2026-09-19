import { ArrImportWizard } from '@/components/arr-import'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'

export function ArrImportPage() {
  const back = usePushBack()

  return (
    <Screen title="Migrate from *arr" back={back}>
      <div className="px-screen">
        <ArrImportWizard />
      </div>
    </Screen>
  )
}
