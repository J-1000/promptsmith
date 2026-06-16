import { describe, it, expect, vi } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useApi } from './useApi'

describe('useApi', () => {
  it('exposes data on success', async () => {
    const { result } = renderHook(() => useApi(() => Promise.resolve('hello'), []))
    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.data).toBe('hello')
    expect(result.current.error).toBeNull()
  })

  it('exposes the message on failure', async () => {
    const { result } = renderHook(() =>
      useApi(() => Promise.reject(new Error('boom')), []),
    )
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.error).toBe('boom')
    expect(result.current.data).toBeNull()
  })

  it('does not update state after unmount', async () => {
    let resolve!: (value: string) => void
    const fetcher = () => new Promise<string>((r) => { resolve = r })
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    const { result, unmount } = renderHook(() => useApi(fetcher, []))
    unmount()
    await act(async () => {
      resolve('late')
      await Promise.resolve()
    })

    expect(result.current.data).toBeNull()
    expect(errorSpy).not.toHaveBeenCalled()
    errorSpy.mockRestore()
  })

  it('refetches on demand', async () => {
    const fetcher = vi.fn().mockResolvedValue('value')
    const { result } = renderHook(() => useApi(fetcher, []))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(fetcher).toHaveBeenCalledTimes(1)

    act(() => result.current.refetch())
    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(2))
  })
})
