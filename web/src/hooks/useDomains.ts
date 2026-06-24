import { useEffect, useState } from 'react'
import { getJson } from '../lib/api'
import type { Domain } from '../types'

export function useDomains() {
  const [domains, setDomains] = useState<Domain[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getJson<Domain[]>('domains')
      .then(setDomains)
      .catch((err) => setError(err instanceof Error ? err.message : '加载域名失败'))
      .finally(() => setLoading(false))
  }, [])

  return { domains, loading, error }
}
