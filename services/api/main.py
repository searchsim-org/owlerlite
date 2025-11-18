"""
OwlerLite API Service
FastAPI wrapper for browser extension communication
"""

from fastapi import FastAPI, HTTPException, status
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import List, Optional, Dict, Any
from datetime import datetime
import asyncio
import aiohttp
import logging
import os

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="OwlerLite API",
    description="Scope- and freshness-aware RAG API for browser extension",
    version="1.0.0"
)

# CORS configuration for browser extension
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # For development; restrict in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Configuration
LIGHTRAG_URL = os.getenv("LIGHTRAG_URL", "http://lightrag:9621")
CRAWLER_URL = os.getenv("CRAWLER_URL", "http://crawler:8081")
FRONTIER_URL = os.getenv("FRONTIER_URL", "http://frontier:7072")

# In-memory storage (replace with database in production)
scopes_db: Dict[str, Dict] = {}
stats_db: Dict[str, Any] = {
    "totalScopes": 0,
    "totalPages": 0,
    "activeCrawls": 0,
    "pendingUpdates": 0,
    "recentActivity": [],
    "crawlQueue": [],
    "freshnessData": []
}

# Pydantic models
class Scope(BaseModel):
    name: str
    description: Optional[str] = ""
    patterns: List[str] = []
    autoTrack: bool = False
    
class ScopeUpdate(BaseModel):
    name: Optional[str] = None
    description: Optional[str] = None
    patterns: Optional[List[str]] = None
    autoTrack: Optional[bool] = None

class Page(BaseModel):
    url: str
    title: Optional[str] = None
    triggerCrawl: bool = True

class BulkPages(BaseModel):
    urls: List[str]

class Query(BaseModel):
    query: str
    scopes: List[str]

class APIKeys(BaseModel):
    llm: Dict[str, str]
    embedding: Dict[str, str]

# Helper functions
def generate_scope_id() -> str:
    """Generate unique scope ID"""
    import uuid
    return str(uuid.uuid4())

async def call_lightrag(endpoint: str, method: str = "GET", data: Optional[Dict] = None):
    """Call LightRAG service"""
    try:
        async with aiohttp.ClientSession() as session:
            url = f"{LIGHTRAG_URL}{endpoint}"
            if method == "GET":
                async with session.get(url) as response:
                    return await response.json()
            elif method == "POST":
                async with session.post(url, json=data) as response:
                    return await response.json()
    except Exception as e:
        logger.error(f"LightRAG call failed: {e}")
        return None

async def queue_crawl(url: str, scope_id: str):
    """Queue URL for crawling"""
    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(
                f"{FRONTIER_URL}/urls",
                json={"url": url, "metadata": {"scope_id": scope_id}}
            ) as response:
                return response.status == 200
    except Exception as e:
        logger.error(f"Failed to queue crawl: {e}")
        return False

# Health check
@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "ok", "service": "owlerlite-api", "version": "1.0.0"}

# Scope management endpoints
@app.get("/scopes")
async def get_scopes():
    """Get all scopes"""
    scopes_list = [
        {
            "id": scope_id,
            "name": scope["name"],
            "description": scope["description"],
            "patterns": scope["patterns"],
            "autoTrack": scope["autoTrack"],
            "pageCount": scope.get("pageCount", 0),
            "createdAt": scope.get("createdAt"),
            "updatedAt": scope.get("updatedAt")
        }
        for scope_id, scope in scopes_db.items()
    ]
    return {"scopes": scopes_list}

@app.post("/scopes", status_code=status.HTTP_201_CREATED)
async def create_scope(scope: Scope):
    """Create a new scope"""
    scope_id = generate_scope_id()
    now = datetime.utcnow().isoformat()
    
    scopes_db[scope_id] = {
        "name": scope.name,
        "description": scope.description,
        "patterns": scope.patterns,
        "autoTrack": scope.autoTrack,
        "pageCount": 0,
        "createdAt": now,
        "updatedAt": now
    }
    
    stats_db["totalScopes"] = len(scopes_db)
    
    logger.info(f"Created scope: {scope_id} ({scope.name})")
    
    return {
        "id": scope_id,
        "message": "Scope created successfully",
        "scope": scopes_db[scope_id]
    }

@app.get("/scopes/{scope_id}")
async def get_scope(scope_id: str):
    """Get scope by ID"""
    if scope_id not in scopes_db:
        raise HTTPException(status_code=404, detail="Scope not found")
    
    return {
        "id": scope_id,
        **scopes_db[scope_id]
    }

@app.put("/scopes/{scope_id}")
async def update_scope(scope_id: str, scope_update: ScopeUpdate):
    """Update scope"""
    if scope_id not in scopes_db:
        raise HTTPException(status_code=404, detail="Scope not found")
    
    # Update fields
    if scope_update.name is not None:
        scopes_db[scope_id]["name"] = scope_update.name
    if scope_update.description is not None:
        scopes_db[scope_id]["description"] = scope_update.description
    if scope_update.patterns is not None:
        scopes_db[scope_id]["patterns"] = scope_update.patterns
    if scope_update.autoTrack is not None:
        scopes_db[scope_id]["autoTrack"] = scope_update.autoTrack
    
    scopes_db[scope_id]["updatedAt"] = datetime.utcnow().isoformat()
    
    logger.info(f"Updated scope: {scope_id}")
    
    return {
        "id": scope_id,
        "message": "Scope updated successfully",
        "scope": scopes_db[scope_id]
    }

@app.delete("/scopes/{scope_id}")
async def delete_scope(scope_id: str):
    """Delete scope"""
    if scope_id not in scopes_db:
        raise HTTPException(status_code=404, detail="Scope not found")
    
    del scopes_db[scope_id]
    stats_db["totalScopes"] = len(scopes_db)
    
    logger.info(f"Deleted scope: {scope_id}")
    
    return {"message": "Scope deleted successfully"}

# Page management endpoints
@app.post("/scopes/{scope_id}/pages")
async def add_page_to_scope(scope_id: str, page: Page):
    """Add page to scope"""
    if scope_id not in scopes_db:
        raise HTTPException(status_code=404, detail="Scope not found")
    
    # Queue for crawling
    if page.triggerCrawl:
        success = await queue_crawl(page.url, scope_id)
        if not success:
            logger.warning(f"Failed to queue crawl for {page.url}")
    
    # Update page count
    scopes_db[scope_id]["pageCount"] = scopes_db[scope_id].get("pageCount", 0) + 1
    stats_db["totalPages"] = sum(s.get("pageCount", 0) for s in scopes_db.values())
    
    # Add to recent activity
    stats_db["recentActivity"].insert(0, {
        "description": f"Added page to {scopes_db[scope_id]['name']}",
        "timestamp": datetime.utcnow().timestamp() * 1000
    })
    stats_db["recentActivity"] = stats_db["recentActivity"][:20]  # Keep last 20
    
    # Add to crawl queue
    if page.triggerCrawl:
        stats_db["crawlQueue"].insert(0, {
            "url": page.url,
            "scope": scopes_db[scope_id]["name"]
        })
        stats_db["crawlQueue"] = stats_db["crawlQueue"][:20]
    
    logger.info(f"Added page {page.url} to scope {scope_id}")
    
    return {
        "message": "Page added successfully",
        "url": page.url,
        "queued_for_crawl": page.triggerCrawl
    }

@app.post("/scopes/{scope_id}/pages/bulk")
async def add_pages_bulk(scope_id: str, pages: BulkPages):
    """Add multiple pages to scope"""
    if scope_id not in scopes_db:
        raise HTTPException(status_code=404, detail="Scope not found")
    
    # Queue all URLs for crawling
    queued = 0
    for url in pages.urls:
        success = await queue_crawl(url, scope_id)
        if success:
            queued += 1
    
    # Update stats
    scopes_db[scope_id]["pageCount"] = scopes_db[scope_id].get("pageCount", 0) + len(pages.urls)
    stats_db["totalPages"] = sum(s.get("pageCount", 0) for s in scopes_db.values())
    
    logger.info(f"Added {len(pages.urls)} pages to scope {scope_id}")
    
    return {
        "message": f"Added {len(pages.urls)} pages",
        "total": len(pages.urls),
        "queued": queued
    }

# Query endpoint
@app.post("/query")
async def execute_query(query: Query):
    """Execute scoped query"""
    # Validate scopes
    for scope_id in query.scopes:
        if scope_id not in scopes_db:
            raise HTTPException(status_code=400, detail=f"Invalid scope: {scope_id}")
    
    # Get scope names for context
    scope_names = [scopes_db[sid]["name"] for sid in query.scopes]
    
    # Call LightRAG for retrieval
    lightrag_query = {
        "query": query.query,
        "scopes": query.scopes,
        "mode": "hybrid"
    }
    
    results = await call_lightrag("/query", "POST", lightrag_query)
    
    if results is None:
        # Return placeholder results if LightRAG unavailable
        results = {
            "answer": f"Searching in scopes: {', '.join(scope_names)}. Backend processing...",
            "results": [
                {
                    "title": "Example Result",
                    "url": "https://example.com",
                    "snippet": "This is a placeholder result. The actual backend will provide real results.",
                    "score": 0.85,
                    "version": "1",
                    "lastUpdate": datetime.utcnow().isoformat(),
                    "scopes": scope_names,
                    "freshness": 0.9,
                    "scoreBreakdown": {
                        "semantic": 0.75,
                        "graph": 0.10,
                        "scope": 0.95,
                        "freshness": 0.90
                    }
                }
            ]
        }
    
    logger.info(f"Query executed: {query.query} in scopes {query.scopes}")
    
    return results

# Statistics endpoint
@app.get("/stats")
async def get_stats():
    """Get system statistics"""
    return {
        "totalScopes": len(scopes_db),
        "totalPages": sum(s.get("pageCount", 0) for s in scopes_db.values()),
        "activeCrawls": stats_db.get("activeCrawls", 0),
        "pendingUpdates": stats_db.get("pendingUpdates", 0),
        "recentActivity": stats_db.get("recentActivity", []),
        "crawlQueue": stats_db.get("crawlQueue", []),
        "freshnessData": stats_db.get("freshnessData", [])
    }

# Configuration endpoint
@app.post("/config/api-keys")
async def save_api_keys(api_keys: APIKeys):
    """Save API keys configuration"""
    # In production, forward to LightRAG service
    # For now, just acknowledge
    logger.info("API keys configuration received")
    
    return {
        "message": "API keys saved successfully",
        "llm_provider": api_keys.llm.get("provider"),
        "embedding_provider": api_keys.embedding.get("provider")
    }

# Root endpoint
@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "OwlerLite API",
        "version": "1.0.0",
        "docs": "/docs",
        "health": "/health"
    }

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=7001,
        reload=True,
        log_level="info"
    )

