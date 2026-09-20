import type { ActionItem } from '@/components/presenter'
import { ActionPresenter } from '@/components/presenter'

import type { SessionActions } from './use-session-actions'

function restartLabel(countdown: number | null, isPending: boolean): string {
  if (countdown !== null) {
    return `Restarting (${countdown}s)`
  }
  if (isPending) {
    return 'Restarting...'
  }
  return 'Restart'
}

function restartDescription(countdown: number | null): string {
  if (countdown !== null) {
    return 'Server is restarting. The page will refresh automatically.'
  }
  return 'The application will be briefly unavailable.'
}

export function SessionActionPresenters({ session }: { session: SessionActions }) {
  const locked = session.countdown !== null
  const restartAction: ActionItem = {
    label: restartLabel(session.countdown, session.isRestartPending),
    destructive: true,
    disabled: session.isRestartPending || locked,
    keepOpen: true,
    onClick: () => {
      void session.handleRestart()
    },
  }
  const logoutAction: ActionItem = {
    label: 'Log out',
    destructive: true,
    onClick: session.handleLogout,
  }

  return (
    <>
      <ActionPresenter
        open={session.restartOpen}
        onOpenChange={session.setRestartOpen}
        title="Restart SlipStream"
        description={restartDescription(session.countdown)}
        actions={[restartAction]}
        wide="dialog"
        locked={locked}
      />
      <ActionPresenter
        open={session.logoutOpen}
        onOpenChange={session.setLogoutOpen}
        title="Log out"
        description="You will need to sign in again."
        actions={[logoutAction]}
        wide="dialog"
      />
    </>
  )
}
