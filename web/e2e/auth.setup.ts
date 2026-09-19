import { test as setup } from '@playwright/test'
import fs from 'node:fs/promises'
import path from 'node:path'

import { createAdminSession, storageStateFromAuth } from './helpers/auth'
import { authStatePath } from './helpers/paths'

setup('authenticate as admin', async ({ request }) => {
  const payload = await createAdminSession(request)
  await fs.mkdir(path.dirname(authStatePath), { recursive: true })
  await fs.writeFile(authStatePath, JSON.stringify(storageStateFromAuth(payload), null, 2))
})
