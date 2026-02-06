// API client for PromptSmith backend

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export interface Project {
  id: string;
  name: string;
}

export interface Prompt {
  id: string;
  name: string;
  description: string;
  file_path: string;
  version?: string;
  created_at: string;
}

export interface Version {
  id: string;
  version: string;
  content: string;
  commit_message: string;
  created_at: string;
  tags?: string[];
}

export interface TestSuite {
  name: string;
  file_path: string;
  prompt: string;
  description?: string;
  test_count: number;
}

export interface TestResult {
  test_name: string;
  passed: boolean;
  skipped: boolean;
  output?: string;
  failures?: Array<{
    type: string;
    passed: boolean;
    expected: string;
    actual: string;
    message?: string;
  }>;
  error?: string;
  duration_ms: number;
}

export interface SuiteResult {
  suite_name: string;
  prompt_name: string;
  version: string;
  passed: number;
  failed: number;
  skipped: number;
  total: number;
  results: TestResult[];
  duration_ms: number;
}

export interface BenchmarkSuite {
  name: string;
  file_path: string;
  prompt: string;
  description?: string;
  models: string[];
  runs_per_model: number;
}

export interface ModelResult {
  model: string;
  runs: number;
  errors: number;
  error_rate: number;
  latency_p50_ms: number;
  latency_p99_ms: number;
  total_tokens_avg: number;
  cost_per_request: number;
}

export interface BenchmarkResult {
  suite_name: string;
  prompt_name: string;
  version: string;
  models: ModelResult[];
  duration_ms: number;
}

async function fetchApi<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

// Project
export async function getProject(): Promise<Project> {
  return fetchApi<Project>('/api/project');
}

// Prompts
export async function listPrompts(): Promise<Prompt[]> {
  return fetchApi<Prompt[]>('/api/prompts');
}

export async function getPrompt(name: string): Promise<Prompt> {
  return fetchApi<Prompt>(`/api/prompts/${name}`);
}

export async function getPromptVersions(name: string): Promise<Version[]> {
  return fetchApi<Version[]>(`/api/prompts/${name}/versions`);
}

export async function getPromptDiff(
  name: string,
  v1: string,
  v2: string
): Promise<{ prompt: string; v1: { version: string; content: string }; v2: { version: string; content: string } }> {
  return fetchApi(`/api/prompts/${name}/diff?v1=${v1}&v2=${v2}`);
}

export async function createVersion(
  name: string,
  content: string,
  commitMessage: string
): Promise<Version> {
  return fetchApi<Version>(`/api/prompts/${name}/versions`, {
    method: 'POST',
    body: JSON.stringify({ content, commit_message: commitMessage }),
  });
}

export async function createPrompt(
  name: string,
  description: string,
  content?: string
): Promise<Prompt> {
  return fetchApi<Prompt>('/api/prompts', {
    method: 'POST',
    body: JSON.stringify({ name, description, content }),
  });
}

export async function deletePrompt(name: string): Promise<void> {
  await fetch(`${API_BASE}/api/prompts/${name}`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
  }).then((res) => {
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
  });
}

export async function createTag(
  promptName: string,
  tagName: string,
  versionId: string
): Promise<{ id: string; name: string; version_id: string }> {
  return fetchApi(`/api/prompts/${promptName}/tags`, {
    method: 'POST',
    body: JSON.stringify({ name: tagName, version_id: versionId }),
  });
}

export async function deleteTag(promptName: string, tagName: string): Promise<void> {
  await fetch(`${API_BASE}/api/prompts/${promptName}/tags/${tagName}`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
  }).then((res) => {
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
  });
}

// Tests
export async function listTests(): Promise<TestSuite[]> {
  return fetchApi<TestSuite[]>('/api/tests');
}

export async function getTest(name: string): Promise<TestSuite> {
  return fetchApi<TestSuite>(`/api/tests/${name}`);
}

export async function runTest(name: string): Promise<SuiteResult> {
  return fetchApi<SuiteResult>(`/api/tests/${name}/run`, { method: 'POST' });
}

// Benchmarks
export async function listBenchmarks(): Promise<BenchmarkSuite[]> {
  return fetchApi<BenchmarkSuite[]>('/api/benchmarks');
}

export async function getBenchmark(name: string): Promise<BenchmarkSuite> {
  return fetchApi<BenchmarkSuite>(`/api/benchmarks/${name}`);
}

export async function runBenchmark(name: string): Promise<BenchmarkResult> {
  return fetchApi<BenchmarkResult>(`/api/benchmarks/${name}/run`, { method: 'POST' });
}

// Generate

export interface GenerateVariation {
  content: string;
  description: string;
  token_delta?: number;
}

export interface GenerateResult {
  original: string;
  variations: GenerateVariation[];
  model: string;
  type: string;
  goal?: string;
}

export interface GenerateRequest {
  type: 'variations' | 'compress' | 'expand' | 'rephrase';
  prompt: string;
  count?: number;
  goal?: string;
  model?: string;
}

export async function generateVariations(request: GenerateRequest): Promise<GenerateResult> {
  return fetchApi<GenerateResult>('/api/generate', {
    method: 'POST',
    body: JSON.stringify(request),
  });
}
