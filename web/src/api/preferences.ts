import { apiFetch } from './client'

export type AddFlowPreferences = {
  movieSearchOnAdd: boolean
  seriesSearchOnAdd: SeriesSearchOnAdd
  seriesMonitorOnAdd: SeriesMonitorOnAdd
  seriesIncludeSpecials: boolean
}

export type SeriesSearchOnAdd = 'no' | 'first_episode' | 'first_season' | 'latest_season' | 'all'
export type SeriesMonitorOnAdd = 'none' | 'first_season' | 'latest_season' | 'future' | 'all'

export const preferencesApi = {
  getAddFlowPreferences: () => apiFetch<AddFlowPreferences>('/preferences/addflow'),

  setAddFlowPreferences: (prefs: Partial<AddFlowPreferences>) =>
    apiFetch<AddFlowPreferences>('/preferences/addflow', {
      method: 'PUT',
      body: JSON.stringify(prefs),
    }),
}
