import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import { AuthenticationSection } from '@/components/settings'

export function AuthenticationPage() {
  const back = usePushBack()

  return (
    <Screen title="Authentication" back={back}>
      <div className="px-screen max-w-2xl">
        <AuthenticationSection />
      </div>
    </Screen>
  )
}
