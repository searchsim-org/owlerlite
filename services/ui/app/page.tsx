
"use client"
import { useState, useEffect } from "react"

const API = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:7000"

interface Scope {
  id: number
  name: string
  patterns: string[]
}

interface ScoreBreakdown {
  sim_vec: number
  score_graph: number
  scope_prior: number
  freshness: number
}

interface Result {
  doc_id: string
  url: string
  snippet: string
  chunk_id: number
  version_ts: number
  scope: string
  score: number
  breakdown: ScoreBreakdown
}

interface QueryResponse {
  items: Result[]
  rationale?: string
}

interface DiffLine {
  type: "added" | "removed"
  line: string
}

interface CrawlerStats {
  pages: number
  chunks: number
  versions: number
}

export default function Home() {
  const [scopes, setScopes] = useState<Scope[]>([])
  const [selectedScopes, setSelectedScopes] = useState<number[]>([])
  const [scopeName, setScopeName] = useState("")
  const [scopePatterns, setScopePatterns] = useState("")
  const [seedUrls, setSeedUrls] = useState("")
  const [query, setQuery] = useState("")
  const [generateNL, setGenerateNL] = useState(true)
  const [results, setResults] = useState<Result[]>([])
  const [rationale, setRationale] = useState("")
  const [loading, setLoading] = useState(false)
  const [diffData, setDiffData] = useState<{ url: string; chunk_id: number; diff: DiffLine[] | null; old_ts: number; new_ts: number } | null>(null)
  const [crawlerStats, setCrawlerStats] = useState<CrawlerStats | null>(null)
  const [activeTab, setActiveTab] = useState<"query" | "scopes" | "stats">("query")

  useEffect(() => {
    loadScopes()
    loadCrawlerStats()
  }, [])

  const loadScopes = async () => {
    try {
      const res = await fetch(`${API}/scopes`)
      const data = await res.json()
      setScopes(data.scopes || [])
    } catch (e) {
      console.error("Failed to load scopes:", e)
    }
  }

  const loadCrawlerStats = async () => {
    try {
      const res = await fetch(`${API.replace(":7000", ":8081")}/stats`)
      const data = await res.json()
      setCrawlerStats(data)
    } catch (e) {
      console.error("Failed to load crawler stats:", e)
    }
  }

  const createScope = async () => {
    if (!scopeName.trim()) return
    try {
      const res = await fetch(`${API}/scopes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: scopeName,
          patterns: scopePatterns.split("\n").filter(Boolean)
        })
      })
      const data = await res.json()
      setScopeName("")
      setScopePatterns("")
      loadScopes()
      alert(`Created scope "${data.Name}" with ID ${data.ID}`)
    } catch (e) {
      alert("Failed to create scope")
    }
  }

  const seedScope = async (scopeId: number) => {
    if (!seedUrls.trim()) return
    try {
      await fetch(`${API}/scopes/${scopeId}/seed`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          urls: seedUrls.split("\n").filter(Boolean)
        })
      })
      setSeedUrls("")
      alert("URLs added to crawl queue")
    } catch (e) {
      alert("Failed to seed scope")
    }
  }

  const executeQuery = async () => {
    if (!query.trim()) return
    setLoading(true)
    setResults([])
    setRationale("")
    try {
      const res = await fetch(`${API}/query`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          query,
          scope_ids: selectedScopes,
          mode: "hybrid",
          generate_nl: generateNL
        })
      })
      const data: QueryResponse = await res.json()
      setResults(data.items || [])
      setRationale(data.rationale || "")
    } catch (e) {
      alert("Query failed")
    } finally {
      setLoading(false)
    }
  }

  const loadDiff = async (url: string, chunkId: number) => {
    try {
      const res = await fetch(`${API}/diff?url=${encodeURIComponent(url)}&chunk_id=${chunkId}`)
      const data = await res.json()
      setDiffData({
        url,
        chunk_id: chunkId,
        diff: data.diff,
        old_ts: data.old_ts,
        new_ts: data.new_ts
      })
    } catch (e) {
      console.error("Failed to load diff:", e)
    }
  }

  const toggleScope = (id: number) => {
    setSelectedScopes(prev =>
      prev.includes(id) ? prev.filter(s => s !== id) : [...prev, id]
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
      <div className="max-w-6xl mx-auto p-6">
        {/* Header */}
        <header className="mb-8">
          <h1 className="text-3xl font-bold text-white mb-2">
            🦉 OwlerLite
          </h1>
          <p className="text-purple-200">
            Scope- and Freshness-Aware Web Retrieval
          </p>
        </header>

        {/* Tab Navigation */}
        <div className="flex gap-2 mb-6">
          {["query", "scopes", "stats"].map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab as typeof activeTab)}
              className={`px-4 py-2 rounded-lg font-medium transition-all ${activeTab === tab
                  ? "bg-purple-500 text-white"
                  : "bg-white/10 text-purple-200 hover:bg-white/20"
                }`}
            >
              {tab === "query" && "🔍 Query"}
              {tab === "scopes" && "📁 Scopes"}
              {tab === "stats" && "📊 Stats"}
            </button>
          ))}
        </div>

        {/* Query Tab */}
        {activeTab === "query" && (
          <div className="space-y-6">
            {/* Scope Selection */}
            <div className="bg-white/10 backdrop-blur rounded-2xl p-4">
              <h2 className="text-white font-medium mb-3">Select Scopes</h2>
              {scopes.length === 0 ? (
                <p className="text-purple-200 text-sm">No scopes defined. Create one in the Scopes tab.</p>
              ) : (
                <div className="flex flex-wrap gap-2">
                  {scopes.map(scope => (
                    <button
                      key={scope.id}
                      onClick={() => toggleScope(scope.id)}
                      className={`px-3 py-1.5 rounded-full text-sm font-medium transition-all ${selectedScopes.includes(scope.id)
                          ? "bg-purple-500 text-white"
                          : "bg-white/20 text-purple-200 hover:bg-white/30"
                        }`}
                    >
                      {scope.name}
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Query Input */}
            <div className="bg-white/10 backdrop-blur rounded-2xl p-4">
              <div className="flex gap-3">
                <input
                  type="text"
                  value={query}
                  onChange={e => setQuery(e.target.value)}
                  onKeyDown={e => e.key === "Enter" && executeQuery()}
                  placeholder="Ask a question..."
                  className="flex-1 bg-white/20 text-white placeholder-purple-300 rounded-xl px-4 py-3 focus:outline-none focus:ring-2 focus:ring-purple-400"
                />
                <button
                  onClick={executeQuery}
                  disabled={loading}
                  className="px-6 py-3 bg-purple-500 text-white font-medium rounded-xl hover:bg-purple-400 transition-all disabled:opacity-50"
                >
                  {loading ? "..." : "Search"}
                </button>
              </div>
              <label className="flex items-center gap-2 mt-3 text-purple-200 text-sm cursor-pointer">
                <input
                  type="checkbox"
                  checked={generateNL}
                  onChange={e => setGenerateNL(e.target.checked)}
                  className="rounded"
                />
                Generate NL rationale for results
              </label>
            </div>

            {/* Rationale */}
            {rationale && (
              <div className="bg-emerald-500/20 backdrop-blur rounded-2xl p-4">
                <h3 className="text-emerald-300 font-medium mb-2">💡 Rationale</h3>
                <p className="text-white text-sm">{rationale}</p>
              </div>
            )}

            {/* Results */}
            {results.length > 0 && (
              <div className="space-y-4">
                <h3 className="text-white font-medium">Results ({results.length})</h3>
                {results.map((r, i) => (
                  <div key={i} className="bg-white/10 backdrop-blur rounded-2xl p-4">
                    <div className="flex justify-between items-start mb-2">
                      <a
                        href={r.url}
                        target="_blank"
                        rel="noopener"
                        className="text-purple-300 hover:text-purple-200 text-sm truncate max-w-[70%]"
                      >
                        {r.url}
                      </a>
                      <span className="text-white font-mono text-sm bg-purple-500/50 px-2 py-0.5 rounded">
                        {r.score.toFixed(3)}
                      </span>
                    </div>
                    <p className="text-white text-sm mb-3 line-clamp-3">{r.snippet}</p>

                    {/* Score Breakdown */}
                    <div className="grid grid-cols-4 gap-2 mb-3">
                      {[
                        { label: "Semantic", value: r.breakdown.sim_vec, color: "blue" },
                        { label: "Graph", value: r.breakdown.score_graph, color: "green" },
                        { label: "Scope", value: r.breakdown.scope_prior, color: "purple" },
                        { label: "Fresh", value: r.breakdown.freshness, color: "orange" }
                      ].map(({ label, value, color }) => (
                        <div key={label} className="text-center">
                          <div className={`text-xs text-${color}-300`}>{label}</div>
                          <div className="text-white font-mono text-sm">{(value * 100).toFixed(0)}%</div>
                          <div className="h-1 bg-white/20 rounded-full mt-1">
                            <div
                              className={`h-full bg-${color}-400 rounded-full`}
                              style={{ width: `${Math.min(value * 100, 100)}%` }}
                            />
                          </div>
                        </div>
                      ))}
                    </div>

                    {/* Actions */}
                    <div className="flex gap-2">
                      <button
                        onClick={() => loadDiff(r.url, r.chunk_id)}
                        className="px-3 py-1 bg-indigo-500/50 text-white text-xs rounded-lg hover:bg-indigo-500/70"
                      >
                        View Diff
                      </button>
                      <button
                        onClick={async () => {
                          await fetch(`${API}/feedback/click`, {
                            method: "POST",
                            headers: { "Content-Type": "application/json" },
                            body: JSON.stringify({ scope_id: selectedScopes[0], doc_id: r.doc_id })
                          })
                        }}
                        className="px-3 py-1 bg-emerald-500/50 text-white text-xs rounded-lg hover:bg-emerald-500/70"
                      >
                        Mark as used
                      </button>
                      <span className="text-purple-300 text-xs ml-auto">
                        Scope: {r.scope || "—"} | v{r.version_ts}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* Diff Modal */}
            {diffData && (
              <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
                <div className="bg-slate-800 rounded-2xl p-6 max-w-2xl w-full max-h-[80vh] overflow-auto">
                  <div className="flex justify-between items-center mb-4">
                    <h3 className="text-white font-medium">
                      Version Diff: Chunk #{diffData.chunk_id}
                    </h3>
                    <button
                      onClick={() => setDiffData(null)}
                      className="text-white/60 hover:text-white"
                    >
                      ✕
                    </button>
                  </div>
                  <p className="text-purple-300 text-sm mb-4 truncate">{diffData.url}</p>
                  {diffData.diff ? (
                    <div className="space-y-1 font-mono text-sm">
                      {diffData.diff.map((line, i) => (
                        <div
                          key={i}
                          className={`px-2 py-1 rounded ${line.type === "added"
                              ? "bg-emerald-500/20 text-emerald-300"
                              : "bg-red-500/20 text-red-300"
                            }`}
                        >
                          {line.type === "added" ? "+ " : "- "}{line.line}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-purple-200">No diff available (only one version exists)</p>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Scopes Tab */}
        {activeTab === "scopes" && (
          <div className="grid md:grid-cols-2 gap-6">
            {/* Create Scope */}
            <div className="bg-white/10 backdrop-blur rounded-2xl p-4">
              <h2 className="text-white font-medium mb-3">Create New Scope</h2>
              <input
                type="text"
                value={scopeName}
                onChange={e => setScopeName(e.target.value)}
                placeholder="Scope name"
                className="w-full bg-white/20 text-white placeholder-purple-300 rounded-xl px-4 py-2 mb-3 focus:outline-none"
              />
              <textarea
                value={scopePatterns}
                onChange={e => setScopePatterns(e.target.value)}
                placeholder="URL patterns (one per line)"
                className="w-full bg-white/20 text-white placeholder-purple-300 rounded-xl px-4 py-2 h-24 mb-3 focus:outline-none"
              />
              <button
                onClick={createScope}
                className="w-full py-2 bg-purple-500 text-white font-medium rounded-xl hover:bg-purple-400"
              >
                Create Scope
              </button>
            </div>

            {/* Seed URLs */}
            <div className="bg-white/10 backdrop-blur rounded-2xl p-4">
              <h2 className="text-white font-medium mb-3">Add URLs to Scope</h2>
              <select
                className="w-full bg-white/20 text-white rounded-xl px-4 py-2 mb-3 focus:outline-none"
                onChange={e => {
                  const id = parseInt(e.target.value)
                  if (id && seedUrls.trim()) seedScope(id)
                }}
              >
                <option value="">Select scope...</option>
                {scopes.map(s => (
                  <option key={s.id} value={s.id}>{s.name}</option>
                ))}
              </select>
              <textarea
                value={seedUrls}
                onChange={e => setSeedUrls(e.target.value)}
                placeholder="URLs to crawl (one per line)"
                className="w-full bg-white/20 text-white placeholder-purple-300 rounded-xl px-4 py-2 h-24 focus:outline-none"
              />
            </div>

            {/* Existing Scopes */}
            <div className="md:col-span-2 bg-white/10 backdrop-blur rounded-2xl p-4">
              <h2 className="text-white font-medium mb-3">Existing Scopes</h2>
              {scopes.length === 0 ? (
                <p className="text-purple-200">No scopes created yet.</p>
              ) : (
                <div className="grid md:grid-cols-3 gap-3">
                  {scopes.map(scope => (
                    <div key={scope.id} className="bg-white/10 rounded-xl p-3">
                      <h3 className="text-white font-medium">{scope.name}</h3>
                      <p className="text-purple-300 text-xs">ID: {scope.id}</p>
                      {scope.patterns?.length > 0 && (
                        <p className="text-purple-300 text-xs mt-1">
                          {scope.patterns.length} patterns
                        </p>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Stats Tab */}
        {activeTab === "stats" && (
          <div className="grid md:grid-cols-3 gap-6">
            <div className="bg-white/10 backdrop-blur rounded-2xl p-6 text-center">
              <div className="text-4xl font-bold text-white mb-2">
                {crawlerStats?.pages || 0}
              </div>
              <div className="text-purple-200">Pages Crawled</div>
            </div>
            <div className="bg-white/10 backdrop-blur rounded-2xl p-6 text-center">
              <div className="text-4xl font-bold text-white mb-2">
                {crawlerStats?.chunks || 0}
              </div>
              <div className="text-purple-200">Indexed Chunks</div>
            </div>
            <div className="bg-white/10 backdrop-blur rounded-2xl p-6 text-center">
              <div className="text-4xl font-bold text-white mb-2">
                {crawlerStats?.versions || 0}
              </div>
              <div className="text-purple-200">Version Snapshots</div>
            </div>
            <div className="md:col-span-3 bg-white/10 backdrop-blur rounded-2xl p-4">
              <h3 className="text-white font-medium mb-3">System Status</h3>
              <div className="grid md:grid-cols-2 gap-4 text-sm">
                <div className="flex justify-between text-purple-200">
                  <span>API Endpoint</span>
                  <span className="text-white font-mono">{API}</span>
                </div>
                <div className="flex justify-between text-purple-200">
                  <span>Active Scopes</span>
                  <span className="text-white">{scopes.length}</span>
                </div>
              </div>
              <button
                onClick={() => { loadScopes(); loadCrawlerStats() }}
                className="mt-4 px-4 py-2 bg-purple-500/50 text-white text-sm rounded-lg hover:bg-purple-500/70"
              >
                Refresh Stats
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

