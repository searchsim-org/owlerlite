
"use client"
import {useState} from "react"
const API = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:7000";
export default function Home(){
  const [scopeName, setScopeName] = useState("default");
  const [scopeId, setScopeId] = useState<string>("");
  const [urls, setUrls] = useState<string>("https://example.org/");
  const [entities, setEntities] = useState<string>("OpenAI\nLightRAG");
  const [q, setQ] = useState<string>("What is this site about?");
  const [results, setResults] = useState<any[]>([]);
  const [lineage, setLineage] = useState<any[]>([]);
  const [lineageFor, setLineageFor] = useState<{url:string, chunk_id:number}|null>(null);
  return (
    <div className="max-w-5xl mx-auto p-6 space-y-6">
      <h1 className="text-2xl font-semibold">OwlerLite v2 (LightRAG)</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white p-4 rounded-2xl shadow space-y-2">
          <h2 className="font-medium">Create Scope</h2>
          <input className="border p-2 w-full" placeholder="scope name" value={scopeName} onChange={e=>setScopeName(e.target.value)} />
          <button className="px-3 py-2 rounded-lg bg-black text-white" onClick={async()=>{
            const r = await fetch(API+"/scopes",{method:"POST", headers:{"Content-Type":"application/json"}, body: JSON.stringify({name:scopeName, patterns:[]})})
            const j = await r.json(); setScopeId(String(j.id)); alert("scope id "+j.id);
          }}>Create</button>
          <div className="text-xs text-neutral-500">Scope ID: {scopeId||"—"}</div>
        </div>
        <div className="bg-white p-4 rounded-2xl shadow space-y-2">
          <h2 className="font-medium">Seed Frontier</h2>
          <input className="border p-2 w-full" placeholder="scope id" value={scopeId} onChange={e=>setScopeId(e.target.value)} />
          <textarea className="border p-2 w-full h-24" value={urls} onChange={e=>setUrls(e.target.value)} />
          <button className="px-3 py-2 rounded-lg bg-black text-white" onClick={async()=>{
            await fetch(API+`/scopes/${scopeId}/seed`,{method:"POST", headers:{"Content-Type":"application/json"}, body: JSON.stringify({urls: urls.split(/\n+/).filter(Boolean)})})
            alert("seeded")
          }}>Add & Seed</button>
        </div>
        <div className="bg-white p-4 rounded-2xl shadow space-y-2">
          <h2 className="font-medium">Scope KG Entities</h2>
          <textarea className="border p-2 w-full h-24" value={entities} onChange={e=>setEntities(e.target.value)} />
          <button className="px-3 py-2 rounded-lg bg-black text-white" onClick={async()=>{
            await fetch(API+`/scopes/${scopeId}/entities`,{method:"POST", headers:{"Content-Type":"application/json"}, body: JSON.stringify({entities: entities.split(/\n+/).map(s=>s.trim()).filter(Boolean)})})
            alert("saved entities")
          }}>Save Entities</button>
        </div>
      </div>
      <div className="bg-white p-4 rounded-2xl shadow space-y-2">
        <h2 className="font-medium">Query</h2>
        <div className="flex gap-2">
          <input className="border p-2 flex-1" placeholder="Ask..." value={q} onChange={e=>setQ(e.target.value)} />
          <button className="px-3 py-2 rounded-lg bg-black text-white" onClick={async()=>{
            const r = await fetch(API+"/query",{method:"POST", headers:{"Content-Type":"application/json"}, body: JSON.stringify({query:q, scope_ids: [Number(scopeId)].filter(Boolean)})})
            const j = await r.json(); setResults(j.items||[])
          }}>Search</button>
        </div>
      </div>
      {results.length>0 && (
        <div className="bg-white p-4 rounded-2xl shadow space-y-3">
          <h3 className="font-medium">Results</h3>
          {results.map((r,i)=> (
            <div key={i} className="border rounded-xl p-3 space-y-2">
              <div className="text-sm text-neutral-700"><a href={r.url} target="_blank">{r.url}</a></div>
              <div className="text-sm">{r.snippet}</div>
              <div className="text-xs text-neutral-500">score: {r.final_score?.toFixed(3)} (base {r.base_score?.toFixed(3)}) • scope: {r.scope||"?"} • vts: {r.version_ts}</div>
              <div className="flex gap-2">
                <button className="px-2 py-1 rounded bg-emerald-600 text-white text-xs" onClick={async()=>{
                  await fetch(API+"/feedback/click",{method:"POST", headers:{"Content-Type":"application/json"}, body: JSON.stringify({scope_id:Number(scopeId), doc_id:r.doc_id})})
                  alert("Thanks! Learned your preference.")
                }}>I used this</button>
                <button className="px-2 py-1 rounded bg-indigo-600 text-white text-xs" onClick={async()=>{
                  const u = new URL(API+"/versions"); u.searchParams.set("url", r.url); u.searchParams.set("chunk_id", String(r.chunk_id||""))
                  const res = await fetch(u.toString()); const js = await res.json()
                  setLineage(js.versions||[]); setLineageFor({url:r.url, chunk_id:r.chunk_id})
                }}>Lineage</button>
              </div>
            </div>
          ))}
        </div>
      )}
      {lineageFor && (
        <div className="bg-white p-4 rounded-2xl shadow space-y-2">
          <h3 className="font-medium">Version lineage — {lineageFor.url} #{lineageFor.chunk_id}</h3>
          <div className="grid md:grid-cols-2 gap-3">
            {lineage.slice(0,5).map((v:any, idx:number)=> (
              <div key={idx} className="border rounded-lg p-2 text-xs whitespace-pre-wrap">
                <div className="text-[10px] text-neutral-500 mb-1">ts: {v.version_ts} • chunk: {v.chunk_id}</div>
                {v.text?.slice(0,800)}
              </div>
            ))}
          </div>
          {lineage.length>=2 && (
            <div className="mt-3">
              <h4 className="text-sm font-medium">Diff (last vs previous)</h4>
              <pre className="text-xs bg-neutral-100 p-2 rounded">{diff(lineage[0]?.text||"", lineage[1]?.text||"")}</pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
function diff(a:string,b:string){
  const A=a.split(/\n+/), B=b.split(/\n+/); const max=Math.max(A.length,B.length); const out:string[]=[]
  for(let i=0;i<max;i++){ if(A[i]===B[i]) out.push("  "+(A[i]||"")); else{ if(B[i]!==undefined) out.push("- "+B[i]); if(A[i]!==undefined) out.push("+ "+A[i]) } }
  return out.join("\n")
}
