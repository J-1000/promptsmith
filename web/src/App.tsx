import { Routes, Route } from 'react-router-dom'
import { Layout } from './components/Layout'
import { HomePage } from './pages/HomePage'
import { PromptPage } from './pages/PromptPage'
import { TestsPage } from './pages/TestsPage'
import { TestDetailPage } from './pages/TestDetailPage'
import { BenchmarksPage } from './pages/BenchmarksPage'
import { BenchmarkDetailPage } from './pages/BenchmarkDetailPage'
import { SettingsPage } from './pages/SettingsPage'
import { EditorPage } from './pages/EditorPage'
import { PlaygroundPage } from './pages/PlaygroundPage'
import { ChainsPage } from './pages/ChainsPage'
import { ChainDetailPage } from './pages/ChainDetailPage'

export default function App() {
  return (
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
  )
}
