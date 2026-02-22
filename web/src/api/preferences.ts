import { apiFetch } from './client'

type AddFlowPreferences = {
  movieSearchOnAdd: boolean
  seriesSearchOnAdd: SeriesSearchOnAdd
  seriesMonitorOnAdd: SeriesMonitorOnAdd
  seriesIncludeSpecials: boolean
}

type SeriesSearchOnAdd = 'no' | 'first_episode' | 'first_season' | 'latest_season' | 'all'
type SeriesMonitorOnAdd = 'none' | 'first_season' | 'latest_season' | 'future' | 'all'

export const preferencesApi = {
  getAddFlowPreferences: () => apiFetch<AddFlowPreferences>('/preferences/addflow'),

  setAddFlowPreferences: (prefs: Partial<AddFlowPreferences>) =>
    apiFetch<AddFlowPreferences>('/preferences/addflow', {
      method: 'PUT',
      body: JSON.stringify(prefs),
    }),
}
