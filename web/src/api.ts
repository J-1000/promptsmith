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

// Sync config
export interface SyncConfig {
  team: string;
  remote: string;
  auto_push: boolean;
  status: 'configured' | 'not_configured';
}

export async function getSyncConfig(): Promise<SyncConfig> {
  return fetchApi<SyncConfig>('/api/config/sync');
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

export async function updatePrompt(
  name: string,
  newName: string,
  description: string
): Promise<Prompt> {
  return fetchApi<Prompt>(`/api/prompts/${name}`, {
    method: 'PUT',
    body: JSON.stringify({ name: newName, description }),
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

export async function createTestSuite(
  name: string,
  prompt: string,
  description?: string
): Promise<TestSuite> {
  return fetchApi<TestSuite>('/api/tests', {
    method: 'POST',
    body: JSON.stringify({ name, prompt, description }),
  });
}

export async function getTest(name: string): Promise<TestSuite> {
  return fetchApi<TestSuite>(`/api/tests/${name}`);
}

export async function runTest(name: string): Promise<SuiteResult> {
  return fetchApi<SuiteResult>(`/api/tests/${name}/run`, { method: 'POST' });
}

export interface TestRunEntry {
  id: string;
  suite_id: string;
  status: string;
  results: SuiteResult;
  started_at: string;
  completed_at: string;
}

export async function listTestRuns(name: string): Promise<TestRunEntry[]> {
  return fetchApi<TestRunEntry[]>(`/api/tests/${name}/runs`);
}

export async function getTestRun(name: string, runId: string): Promise<TestRunEntry> {
  return fetchApi<TestRunEntry>(`/api/tests/${name}/runs/${runId}`);
}

// Benchmarks
export async function listBenchmarks(): Promise<BenchmarkSuite[]> {
  return fetchApi<BenchmarkSuite[]>('/api/benchmarks');
}

export async function createBenchmarkSuite(
  name: string,
  prompt: string,
  models?: string[],
  runsPerModel?: number,
  description?: string
): Promise<BenchmarkSuite> {
  return fetchApi<BenchmarkSuite>('/api/benchmarks', {
    method: 'POST',
    body: JSON.stringify({ name, prompt, models, runs_per_model: runsPerModel, description }),
  });
}

export async function getBenchmark(name: string): Promise<BenchmarkSuite> {
  return fetchApi<BenchmarkSuite>(`/api/benchmarks/${name}`);
}

export async function runBenchmark(name: string): Promise<BenchmarkResult> {
  return fetchApi<BenchmarkResult>(`/api/benchmarks/${name}/run`, { method: 'POST' });
}

export interface BenchmarkRunEntry {
  id: string;
  benchmark_id: string;
  results: BenchmarkResult;
  created_at: string;
}

export async function listBenchmarkRuns(name: string): Promise<BenchmarkRunEntry[]> {
  return fetchApi<BenchmarkRunEntry[]>(`/api/benchmarks/${name}/runs`);
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

export async function generateCompression(prompt: string, goal?: string, model?: string): Promise<GenerateResult> {
  return fetchApi<GenerateResult>('/api/generate/compress', {
    method: 'POST',
    body: JSON.stringify({ prompt, goal, model }),
  });
}

// Comments
export interface Comment {
  id: string;
  prompt_id: string;
  version_id: string;
  line_number: number;
  content: string;
  created_at: string;
}

export async function listComments(promptName: string): Promise<Comment[]> {
  return fetchApi<Comment[]>(`/api/prompts/${promptName}/comments`);
}

export async function createComment(
  promptName: string,
  versionId: string,
  lineNumber: number,
  content: string
): Promise<Comment> {
  return fetchApi<Comment>(`/api/prompts/${promptName}/comments`, {
    method: 'POST',
    body: JSON.stringify({ version_id: versionId, line_number: lineNumber, content }),
  });
}

export async function deleteComment(commentId: string): Promise<void> {
  await fetch(`${API_BASE}/api/comments/${commentId}`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
  }).then((res) => {
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
  });
}

export async function generateExpansion(prompt: string, goal?: string, model?: string): Promise<GenerateResult> {
  return fetchApi<GenerateResult>('/api/generate/expand', {
    method: 'POST',
    body: JSON.stringify({ prompt, goal, model }),
  });
}

// Playground

export interface PlaygroundRunRequest {
  prompt_name?: string;
  content?: string;
  version?: string;
  model: string;
  variables?: Record<string, string>;
  max_tokens?: number;
  temperature?: number;
}

export interface PlaygroundRunResponse {
  output: string;
  rendered_prompt: string;
  model: string;
  prompt_tokens: number;
  output_tokens: number;
  latency_ms: number;
  cost: number;
}

export async function runPlayground(req: PlaygroundRunRequest): Promise<PlaygroundRunResponse> {
  return fetchApi<PlaygroundRunResponse>('/api/playground/run', {
    method: 'POST',
    body: JSON.stringify(req),
  });
}

export interface ModelInfo {
  id: string;
  provider: string;
}

export async function getAvailableModels(): Promise<ModelInfo[]> {
  const res = await fetchApi<{ models: ModelInfo[] }>('/api/providers/models');
  return res.models;
}

// Dashboard

export interface ActivityEvent {
  type: string;
  title: string;
  detail: string;
  timestamp: string;
  prompt_name: string;
}

export interface PromptHealth {
  prompt_name: string;
  version_count: number;
  last_test_status: string;
  last_test_at: string;
  test_pass_rate: number;
}

export async function getDashboardActivity(limit?: number): Promise<ActivityEvent[]> {
  const query = limit ? `?limit=${limit}` : '';
  return fetchApi<ActivityEvent[]>(`/api/dashboard/activity${query}`);
}

export async function getDashboardHealth(): Promise<PromptHealth[]> {
  return fetchApi<PromptHealth[]>('/api/dashboard/health');
}

// Chains

export interface Chain {
  id: string;
  name: string;
  description: string;
  step_count: number;
  created_at: string;
  updated_at: string;
}

export interface ChainStep {
  id: string;
  step_order: number;
  prompt_name: string;
  input_mapping: Record<string, string>;
  output_key: string;
}

export interface ChainDetail {
  id: string;
  name: string;
  description: string;
  steps: ChainStep[];
  created_at: string;
  updated_at: string;
}

export interface ChainStepInput {
  step_order: number;
  prompt_name: string;
  input_mapping: Record<string, string>;
  output_key: string;
}

export interface ChainStepRunResult {
  step_order: number;
  prompt_name: string;
  output_key: string;
  rendered_prompt: string;
  output: string;
  duration_ms: number;
}

export interface ChainRunResult {
  id: string;
  status: string;
  inputs: Record<string, string>;
  results: ChainStepRunResult[];
  final_output: string;
  started_at: string;
  completed_at: string;
}

export async function listChains(): Promise<Chain[]> {
  return fetchApi<Chain[]>('/api/chains');
}

export async function getChain(name: string): Promise<ChainDetail> {
  return fetchApi<ChainDetail>(`/api/chains/${name}`);
}

export async function createChain(name: string, description: string): Promise<Chain> {
  return fetchApi<Chain>('/api/chains', {
    method: 'POST',
    body: JSON.stringify({ name, description }),
  });
}

export async function updateChain(name: string, newName: string, description: string): Promise<Chain> {
  return fetchApi<Chain>(`/api/chains/${name}`, {
    method: 'PUT',
    body: JSON.stringify({ name: newName, description }),
  });
}

export async function deleteChain(name: string): Promise<void> {
  await fetch(`${API_BASE}/api/chains/${name}`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
  }).then((res) => {
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
  });
}

export async function saveChainSteps(name: string, steps: ChainStepInput[]): Promise<ChainStep[]> {
  return fetchApi<ChainStep[]>(`/api/chains/${name}/steps`, {
    method: 'PUT',
    body: JSON.stringify({ steps }),
  });
}

export async function runChain(
  name: string,
  inputs: Record<string, string>,
  model: string
): Promise<ChainRunResult> {
  return fetchApi<ChainRunResult>(`/api/chains/${name}/run`, {
    method: 'POST',
    body: JSON.stringify({ inputs, model }),
  });
}

export async function listChainRuns(name: string): Promise<ChainRunResult[]> {
  return fetchApi<ChainRunResult[]>(`/api/chains/${name}/runs`);
}
