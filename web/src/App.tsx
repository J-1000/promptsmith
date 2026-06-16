import { Routes, Route } from 'react-router-dom'
import { Suspense, lazy } from 'react'
import { Layout } from './components/Layout'

const HomePage = lazy(() => import('./pages/HomePage').then((module) => ({ default: module.HomePage })))
const PromptPage = lazy(() => import('./pages/PromptPage').then((module) => ({ default: module.PromptPage })))
const TestsPage = lazy(() => import('./pages/TestsPage').then((module) => ({ default: module.TestsPage })))
const TestDetailPage = lazy(() => import('./pages/TestDetailPage').then((module) => ({ default: module.TestDetailPage })))
const BenchmarksPage = lazy(() => import('./pages/BenchmarksPage').then((module) => ({ default: module.BenchmarksPage })))
const BenchmarkDetailPage = lazy(() => import('./pages/BenchmarkDetailPage').then((module) => ({ default: module.BenchmarkDetailPage })))
const SettingsPage = lazy(() => import('./pages/SettingsPage').then((module) => ({ default: module.SettingsPage })))
const EditorPage = lazy(() => import('./pages/EditorPage').then((module) => ({ default: module.EditorPage })))
const PlaygroundPage = lazy(() => import('./pages/PlaygroundPage').then((module) => ({ default: module.PlaygroundPage })))
const ChainsPage = lazy(() => import('./pages/ChainsPage').then((module) => ({ default: module.ChainsPage })))
const ChainDetailPage = lazy(() => import('./pages/ChainDetailPage').then((module) => ({ default: module.ChainDetailPage })))

export default function App() {
  return (
    <Suspense fallback={<div />}>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<HomePage />} />
          <Route path="prompt/:name" element={<PromptPage />} />
          <Route path="prompt/:name/edit" element={<EditorPage />} />
          <Route path="tests" element={<TestsPage />} />
          <Route path="tests/:name" element={<TestDetailPage />} />
          <Route path="benchmarks" element={<BenchmarksPage />} />
          <Route path="benchmarks/:name" element={<BenchmarkDetailPage />} />
          <Route path="chains" element={<ChainsPage />} />
          <Route path="chains/:name" element={<ChainDetailPage />} />
          <Route path="playground" element={<PlaygroundPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </Suspense>
  )
}
