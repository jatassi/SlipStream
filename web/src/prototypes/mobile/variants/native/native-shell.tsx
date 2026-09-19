import type { ReactNode } from 'react'

import { ActivityScreen } from './screens/activity-screen'
import { DetailScreen } from './screens/detail-screen'
import { HomeScreen } from './screens/home-screen'
import { LibraryScreen } from './screens/library-screen'
import { MoreScreen } from './screens/more-screen'
import { SearchScreen } from './screens/search-screen'
import { SettingsSectionScreen, SystemScreen } from './screens/settings-screens'
import { TabBar } from './tab-bar'
import type { NativeNav, NativeTab, PushedScreen } from './use-native-nav'
import { useNativeNav } from './use-native-nav'

import './native.css'

const TAB_LABEL: Record<NativeTab, string> = {
  home: 'Dashboard',
  library: 'Library',
  activity: 'Activity',
  search: 'Search',
  more: 'More',
}

function TabRoot({ nav }: { nav: NativeNav }) {
  const screens: Record<NativeTab, ReactNode> = {
    home: <HomeScreen nav={nav} />,
    library: <LibraryScreen nav={nav} />,
    activity: <ActivityScreen />,
    search: <SearchScreen nav={nav} />,
    more: <MoreScreen nav={nav} />,
  }
  return (
    <>
      {(Object.keys(screens) as NativeTab[]).map((id) => (
        <div key={id} className="absolute inset-0" hidden={id !== nav.tab}>
          {screens[id]}
        </div>
      ))}
      <TabBar tab={nav.tab} onChange={nav.setTab} />
    </>
  )
}

function Pushed({ screen, nav, backLabel }: { screen: PushedScreen; nav: NativeNav; backLabel: string }) {
  switch (screen.kind) {
    case 'detail': {
      return <DetailScreen id={screen.id} onBack={nav.pop} backLabel={backLabel} />
    }
    case 'settings': {
      return <SettingsSectionScreen sectionId={screen.sectionId} onBack={nav.pop} />
    }
    case 'system': {
      return <SystemScreen onBack={nav.pop} backLabel={backLabel} />
    }
  }
}

export function NativeShell() {
  const nav = useNativeNav()
  const covered = nav.exitingKey === null ? nav.stack.length : nav.stack.length - 1

  return (
    <div className="relative h-full overflow-hidden bg-background">
      <div className="native-root absolute inset-0" data-behind={covered > 0 ? '' : undefined}>
        <TabRoot nav={nav} />
      </div>
      {nav.stack.map((screen, index) => (
        <div
          key={screen.key}
          className="native-layer"
          data-behind={index < covered - 1 ? '' : undefined}
          data-exiting={screen.key === nav.exitingKey ? '' : undefined}
          onAnimationEnd={(e) => {
            if (screen.key === nav.exitingKey && e.target === e.currentTarget) {
              nav.finishPop()
            }
          }}
        >
          <Pushed screen={screen} nav={nav} backLabel={index === 0 ? TAB_LABEL[nav.tab] : 'Back'} />
        </div>
      ))}
    </div>
  )
}
