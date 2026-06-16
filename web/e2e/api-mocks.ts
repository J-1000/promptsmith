import { Page, Route } from '@playwright/test'

const corsHeaders = {
  'access-control-allow-origin': 'http://localhost:8081',
  'access-control-allow-methods': 'GET,POST,PUT,DELETE,OPTIONS',
  'access-control-allow-headers': 'content-type',
}

const prompts = [
  {
    id: 'prompt-greeting',
    name: 'greeting',
    description: 'Greets users with a configurable tone',
    file_path: 'prompts/greeting.prompt',
    version: '1.0.2',
    created_at: '2026-06-15T10:00:00Z',
  },
  {
    id: 'prompt-summarize',
    name: 'summarize',
    description: 'Summarizes long-form text',
    file_path: 'prompts/summarize.prompt',
    version: '0.4.0',
    created_at: '2026-06-14T10:00:00Z',
  },
  {
    id: 'prompt-code-review',
    name: 'code-review',
    description: 'Reviews code changes for risk',
    file_path: 'prompts/code-review.prompt',
    version: '2.1.0',
    created_at: '2026-06-13T10:00:00Z',
  },
]

const greetingVersions = [
  {
    id: 'version-greeting-102',
    version: '1.0.2',
    content: 'You are a helpful assistant. Greet {{name}} with a {{tone}} tone.',
    commit_message: 'Add tone parameter for flexibility',
    created_at: '2026-06-15T10:00:00Z',
    tags: ['prod'],
  },
  {
    id: 'version-greeting-101',
    version: '1.0.1',
    content: 'You are a helpful assistant. Greet {{name}} warmly and handle missing names.',
    commit_message: 'Fix greeting for edge cases',
    created_at: '2026-06-14T10:00:00Z',
    tags: [],
  },
  {
    id: 'version-greeting-100',
    version: '1.0.0',
    content: 'You are a helpful assistant. Greet {{name}}.',
    commit_message: 'Initial greeting prompt',
    created_at: '2026-06-13T10:00:00Z',
    tags: [],
  },
]

const tests = [
  {
    name: 'greeting-quality',
    file_path: 'tests/greeting-quality.yaml',
    prompt: 'greeting',
    description: 'Checks greeting quality and length',
    test_count: 4,
  },
]

const benchmarks = [
  {
    name: 'greeting-latency',
    file_path: 'benchmarks/greeting-latency.yaml',
    prompt: 'greeting',
    description: 'Compares greeting prompt latency and cost',
    models: ['gpt-4o', 'gpt-4o-mini', 'claude-sonnet'],
    runs_per_model: 3,
  },
]

const testResult = {
  suite_name: 'greeting-quality',
  prompt_name: 'greeting',
  version: '1.0.2',
  passed: 3,
  failed: 1,
  skipped: 0,
  total: 4,
  duration_ms: 184,
  results: [
    {
      test_name: 'friendly-tone',
      passed: true,
      skipped: false,
      output: 'Hello Ada, welcome back.',
      duration_ms: 31,
    },
    {
      test_name: 'includes-name',
      passed: true,
      skipped: false,
      output: 'Hello Grace, welcome back.',
      duration_ms: 27,
    },
    {
      test_name: 'handles-empty-name',
      passed: true,
      skipped: false,
      output: 'Hello there.',
      duration_ms: 25,
    },
    {
      test_name: 'max-length-check',
      passed: false,
      skipped: false,
      output: 'Hello Ada, it is wonderful to have you back with us today.',
      failures: [
        {
          type: 'length',
          passed: false,
          expected: 'at most 50 characters',
          actual: '64 characters',
          message: 'expected at most 50 characters',
        },
      ],
      duration_ms: 38,
    },
  ],
}

const benchmarkResult = {
  suite_name: 'greeting-latency',
  prompt_name: 'greeting',
  version: '1.0.2',
  duration_ms: 940,
  models: [
    {
      model: 'gpt-4o',
      runs: 3,
      errors: 0,
      error_rate: 0,
      latency_p50_ms: 220,
      latency_p99_ms: 300,
      total_tokens_avg: 180,
      cost_per_request: 0.0012,
    },
    {
      model: 'gpt-4o-mini',
      runs: 3,
      errors: 0,
      error_rate: 0,
      latency_p50_ms: 120,
      latency_p99_ms: 160,
      total_tokens_avg: 174,
      cost_per_request: 0.0002,
    },
    {
      model: 'claude-sonnet',
      runs: 3,
      errors: 0,
      error_rate: 0,
      latency_p50_ms: 180,
      latency_p99_ms: 250,
      total_tokens_avg: 176,
      cost_per_request: 0.0008,
    },
  ],
}

export async function installApiMocks(page: Page) {
  await page.route('http://localhost:8080/api/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (request.method() === 'OPTIONS') {
      await route.fulfill({ status: 204, headers: corsHeaders })
      return
    }

    const body = responseFor(url.pathname, request.method(), url)
    if (body === undefined) {
      await json(route, { error: 'Not found' }, 404)
      return
    }

    await json(route, body)
  })
}

function responseFor(pathname: string, method: string, url: URL): unknown {
  if (method === 'GET' && pathname === '/api/project') {
    return { id: 'project-demo', name: 'promptsmith-demo' }
  }
  if (method === 'GET' && pathname === '/api/prompts') {
    return prompts
  }
  if (method === 'GET' && pathname === '/api/prompts/greeting') {
    return prompts[0]
  }
  if (method === 'GET' && pathname === '/api/prompts/greeting/versions') {
    return greetingVersions
  }
  if (method === 'GET' && pathname === '/api/prompts/greeting/diff') {
    const v1 = versionByNumber(url.searchParams.get('v1'))
    const v2 = versionByNumber(url.searchParams.get('v2'))
    return {
      prompt: 'greeting',
      v1: { version: v1.version, content: v1.content },
      v2: { version: v2.version, content: v2.content },
    }
  }
  if (method === 'GET' && pathname === '/api/tests') {
    return tests
  }
  if (method === 'POST' && pathname === '/api/tests/greeting-quality/run') {
    return testResult
  }
  if (method === 'GET' && pathname === '/api/benchmarks') {
    return benchmarks
  }
  if (method === 'POST' && pathname === '/api/benchmarks/greeting-latency/run') {
    return benchmarkResult
  }
  if (method === 'GET' && pathname === '/api/dashboard/activity') {
    return [
      {
        type: 'version',
        title: 'v1.0.2',
        detail: 'Add tone parameter for flexibility',
        timestamp: '2026-06-15T10:00:00Z',
        prompt_name: 'greeting',
      },
    ]
  }
  if (method === 'GET' && pathname === '/api/dashboard/health') {
    return [
      {
        prompt_name: 'greeting',
        version_count: 3,
        last_test_status: 'failed',
        last_test_at: '2026-06-15T10:05:00Z',
        test_pass_rate: 0.75,
      },
    ]
  }

  return undefined
}

function versionByNumber(version: string | null) {
  return greetingVersions.find((candidate) => candidate.version === version) || greetingVersions[0]
}

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({
    status,
    headers: {
      ...corsHeaders,
      'content-type': 'application/json',
    },
    body: JSON.stringify(body),
  })
}
