import { useCallback, useEffect, useRef, useState } from 'react'

export interface UseApiResult<T> {
  data: T | null
  loading: boolean
  error: string | null
  /** Re-run the fetcher (e.g. for a refresh button). */
  refetch: () => void
}

/**
 * Consolidates the loading/error/data lifecycle shared by every page that reads
 * from the API. A cancellation guard ensures responses that resolve after the
 * component unmounts — or after a newer request supersedes them — are dropped
 * instead of triggering state updates on a stale render.
 */
export function useApi<T>(fetcher: () => Promise<T>, deps: unknown[] = []): UseApiResult<T> {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [nonce, setNonce] = useState(0)

  // Keep the latest fetcher without making it a dependency, so callers can pass
  // an inline closure without forcing a refetch on every render.
  const fetcherRef = useRef(fetcher)
  fetcherRef.current = fetcher

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    fetcherRef
      .current()
      .then((result) => {
        if (!cancelled) setData(result)
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, nonce])

  const refetch = useCallback(() => setNonce((n) => n + 1), [])

  return { data, loading, error, refetch }
}
