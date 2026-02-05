import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  getProject,
  listPrompts,
  getPrompt,
  getPromptVersions,
  getPromptDiff,
  listTests,
  getTest,
  runTest,
  listBenchmarks,
  getBenchmark,
  runBenchmark,
  generateVariations,
} from './api'

// Mock fetch globally
const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

function mockResponse(data: unknown, ok = true, status = 200) {
  return {
    ok,
    status,
    json: () => Promise.resolve(data),
  }
}

function mockErrorResponse(error: string, status = 400) {
  return {
    ok: false,
    status,
    json: () => Promise.resolve({ error }),
  }
}

describe('API Client', () => {
  beforeEach(() => {
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('getProject', () => {
    it('fetches project from /api/project', async () => {
      const project = { id: '1', name: 'test-project' }
      mockFetch.mockResolvedValue(mockResponse(project))

      const result = await getProject()

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/project',
        expect.objectContaining({
          headers: { 'Content-Type': 'application/json' },
        })
      )
      expect(result).toEqual(project)
    })
  })

  describe('listPrompts', () => {
    it('fetches prompts from /api/prompts', async () => {
      const prompts = [
        { id: '1', name: 'greeting', description: 'A greeting', file_path: 'greeting.prompt', created_at: '2024-01-01' },
      ]
      mockFetch.mockResolvedValue(mockResponse(prompts))

      const result = await listPrompts()

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/prompts',
        expect.any(Object)
      )
      expect(result).toEqual(prompts)
    })
  })

  describe('getPrompt', () => {
    it('fetches a specific prompt by name', async () => {
      const prompt = { id: '1', name: 'greeting', description: 'A greeting', file_path: 'greeting.prompt', created_at: '2024-01-01' }
      mockFetch.mockResolvedValue(mockResponse(prompt))

      const result = await getPrompt('greeting')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/prompts/greeting',
        expect.any(Object)
      )
      expect(result).toEqual(prompt)
    })
  })

  describe('getPromptVersions', () => {
    it('fetches versions for a prompt', async () => {
      const versions = [
        { id: '1', version: '1.0.0', content: 'Hello', commit_message: 'Initial', created_at: '2024-01-01', tags: ['prod'] },
      ]
      mockFetch.mockResolvedValue(mockResponse(versions))

      const result = await getPromptVersions('greeting')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/prompts/greeting/versions',
        expect.any(Object)
      )
      expect(result).toEqual(versions)
    })
  })

  describe('getPromptDiff', () => {
    it('fetches diff between two versions', async () => {
      const diff = {
        prompt: 'greeting',
        v1: { version: '1.0.0', content: 'Hello' },
        v2: { version: '1.0.1', content: 'Hello World' },
      }
      mockFetch.mockResolvedValue(mockResponse(diff))

      const result = await getPromptDiff('greeting', '1.0.0', '1.0.1')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/prompts/greeting/diff?v1=1.0.0&v2=1.0.1',
        expect.any(Object)
      )
      expect(result).toEqual(diff)
    })
  })

  describe('listTests', () => {
    it('fetches test suites from /api/tests', async () => {
      const tests = [
        { name: 'greeting-tests', file_path: 'tests/greeting.test.yaml', prompt: 'greeting', test_count: 5 },
      ]
      mockFetch.mockResolvedValue(mockResponse(tests))

      const result = await listTests()

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/tests',
        expect.any(Object)
      )
      expect(result).toEqual(tests)
    })
  })

  describe('getTest', () => {
    it('fetches a specific test suite by name', async () => {
      const test = { name: 'greeting-tests', file_path: 'tests/greeting.test.yaml', prompt: 'greeting', test_count: 5 }
      mockFetch.mockResolvedValue(mockResponse(test))

      const result = await getTest('greeting-tests')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/tests/greeting-tests',
        expect.any(Object)
      )
      expect(result).toEqual(test)
    })
  })

  describe('runTest', () => {
    it('runs a test suite with POST request', async () => {
      const suiteResult = {
        suite_name: 'greeting-tests',
        prompt_name: 'greeting',
        version: '1.0.0',
        passed: 4,
        failed: 1,
        skipped: 0,
        total: 5,
        results: [],
        duration_ms: 100,
      }
      mockFetch.mockResolvedValue(mockResponse(suiteResult))

      const result = await runTest('greeting-tests')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/tests/greeting-tests/run',
        expect.objectContaining({ method: 'POST' })
      )
      expect(result).toEqual(suiteResult)
    })
  })

  describe('listBenchmarks', () => {
    it('fetches benchmark suites from /api/benchmarks', async () => {
      const benchmarks = [
        { name: 'greeting-bench', file_path: 'benchmarks/greeting.bench.yaml', prompt: 'greeting', models: ['gpt-4o'], runs_per_model: 5 },
      ]
      mockFetch.mockResolvedValue(mockResponse(benchmarks))

      const result = await listBenchmarks()

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/benchmarks',
        expect.any(Object)
      )
      expect(result).toEqual(benchmarks)
    })
  })

  describe('getBenchmark', () => {
    it('fetches a specific benchmark suite by name', async () => {
      const benchmark = { name: 'greeting-bench', file_path: 'benchmarks/greeting.bench.yaml', prompt: 'greeting', models: ['gpt-4o'], runs_per_model: 5 }
      mockFetch.mockResolvedValue(mockResponse(benchmark))

      const result = await getBenchmark('greeting-bench')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/benchmarks/greeting-bench',
        expect.any(Object)
      )
      expect(result).toEqual(benchmark)
    })
  })

  describe('runBenchmark', () => {
    it('runs a benchmark suite with POST request', async () => {
      const benchmarkResult = {
        suite_name: 'greeting-bench',
        prompt_name: 'greeting',
        version: '1.0.0',
        models: [],
        duration_ms: 5000,
      }
      mockFetch.mockResolvedValue(mockResponse(benchmarkResult))

      const result = await runBenchmark('greeting-bench')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/benchmarks/greeting-bench/run',
        expect.objectContaining({ method: 'POST' })
      )
      expect(result).toEqual(benchmarkResult)
    })
  })

  describe('generateVariations', () => {
    it('sends generate request with POST and JSON body', async () => {
      const generateResult = {
        original: 'Hello',
        variations: [{ content: 'Hi there', description: 'Casual variation' }],
        model: 'gpt-4o-mini',
        type: 'variations',
      }
      mockFetch.mockResolvedValue(mockResponse(generateResult))

      const request = { type: 'variations' as const, prompt: 'Hello', count: 3 }
      const result = await generateVariations(request)

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/generate',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(request),
        })
      )
      expect(result).toEqual(generateResult)
    })

    it('includes optional goal in request', async () => {
      const generateResult = {
        original: 'Hello',
        variations: [],
        model: 'gpt-4o-mini',
        type: 'compress',
        goal: 'Reduce tokens',
      }
      mockFetch.mockResolvedValue(mockResponse(generateResult))

      const request = { type: 'compress' as const, prompt: 'Hello', goal: 'Reduce tokens' }
      await generateVariations(request)

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/generate',
        expect.objectContaining({
          body: JSON.stringify(request),
        })
      )
    })
  })

  describe('Error handling', () => {
    it('throws error with message from API response', async () => {
      mockFetch.mockResolvedValue(mockErrorResponse('Prompt not found', 404))

      await expect(getPrompt('nonexistent')).rejects.toThrow('Prompt not found')
    })

    it('throws error with HTTP status when no error message', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({}),
      })

      await expect(listPrompts()).rejects.toThrow('HTTP 500')
    })

    it('handles JSON parse failure gracefully', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error('Invalid JSON')),
      })

      await expect(listPrompts()).rejects.toThrow('Unknown error')
    })
  })

  describe('Request headers', () => {
    it('includes Content-Type: application/json header', async () => {
      mockFetch.mockResolvedValue(mockResponse([]))

      await listPrompts()

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
        })
      )
    })
  })
})
