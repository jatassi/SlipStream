import { format, startOfMonth } from 'date-fns'

import { CalendarListView, CalendarMonthView } from '@/components/calendar'
import { ErrorState } from '@/components/data/error-state'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import { Segmented, type SegmentedOption } from '@/components/ui/segmented'
import type { CalendarView } from '@/types/calendar'

import { useCalendarPage } from './use-calendar-page'

const VIEW_OPTIONS: SegmentedOption<CalendarView>[] = [
  { value: 'month', label: 'Month' },
  { value: 'list', label: 'List' },
]

function TodayAction({ onToday }: { onToday: () => void }) {
  return (
    <button
      type="button"
      onClick={onToday}
      className="press-dim min-h-tap text-title focus-visible:ring-ring px-2 outline-none focus-visible:ring-[3px]"
    >
      Today
    </button>
  )
}

export function CalendarPage() {
  const state = useCalendarPage()
  const back = usePushBack()

  const onToday = () => {
    const today = new Date()
    state.changeMonth(startOfMonth(today))
    state.setSelected(format(today, 'yyyy-MM-dd'))
  }

  if (state.isError) {
    return (
      <Screen title="Calendar" back={back}>
        <ErrorState onRetry={state.refetch} />
      </Screen>
    )
  }

  return (
    <Screen
      title="Calendar"
      back={back}
      trailing={state.view === 'month' && <TodayAction onToday={onToday} />}
    >
      <div className="px-screen pb-5">
        <Segmented
          label="Calendar view"
          value={state.view}
          onChange={state.setView}
          options={VIEW_OPTIONS}
        />
      </div>
      {state.view === 'month' ? (
        <CalendarMonthView
          events={state.events}
          month={state.month}
          onMonthChange={state.changeMonth}
          selected={state.selected}
          onSelect={state.setSelected}
        />
      ) : (
        <CalendarListView events={state.events} loading={state.isLoading} />
      )}
    </Screen>
  )
}
