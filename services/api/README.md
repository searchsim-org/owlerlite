# OwlerLite API Service

FastAPI-based REST API wrapper for the OwlerLite browser extension.

## Overview

The API service provides a RESTful interface between the browser extension and the backend services (LightRAG, Crawler, Frontier, Orchestrator). It handles:

- Scope management (CRUD operations)
- Page indexing and crawl queue management
- Query execution with scope-aware retrieval
- Statistics and monitoring
- API key configuration

## Architecture

```
Browser Extension → API Service (FastAPI) → Backend Services
                                           ├── LightRAG (vector + graph)
                                           ├── Crawler (content fetching)
                                           ├── Frontier (URL queue)
                                           └── Orchestrator (coordination)
```

## Endpoints

### Health Check
```
GET /health
```

Returns service status.

### Scope Management

**List all scopes**
```
GET /scopes
```

**Create scope**
```
POST /scopes
Content-Type: application/json

{
  "name": "Python Documentation",
  "description": "Official Python docs and tutorials",
  "patterns": ["https://docs.python.org/*"],
  "autoTrack": false
}
```

**Get scope**
```
GET /scopes/{scope_id}
```

**Update scope**
```
PUT /scopes/{scope_id}
Content-Type: application/json

{
  "name": "Updated Name",
  "description": "Updated description",
  "patterns": ["https://new-pattern.com/*"],
  "autoTrack": true
}
```

**Delete scope**
```
DELETE /scopes/{scope_id}
```

### Page Management

**Add page to scope**
```
POST /scopes/{scope_id}/pages
Content-Type: application/json

{
  "url": "https://example.com/page",
  "title": "Page Title",
  "triggerCrawl": true
}
```

**Bulk add pages**
```
POST /scopes/{scope_id}/pages/bulk
Content-Type: application/json

{
  "urls": [
    "https://example.com/page1",
    "https://example.com/page2"
  ]
}
```

### Query Execution

**Execute scoped query**
```
POST /query
Content-Type: application/json

{
  "query": "How do I use async/await in Python?",
  "scopes": ["scope-id-1", "scope-id-2"]
}
```

Response:
```json
{
  "answer": "Generated answer...",
  "results": [
    {
      "title": "Result Title",
      "url": "https://example.com",
      "snippet": "Relevant text snippet...",
      "score": 0.85,
      "version": "1",
      "lastUpdate": "2024-01-01T00:00:00Z",
      "scopes": ["Python Documentation"],
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
```

### Statistics

**Get system statistics**
```
GET /stats
```

Response:
```json
{
  "totalScopes": 5,
  "totalPages": 150,
  "activeCrawls": 2,
  "pendingUpdates": 10,
  "recentActivity": [...],
  "crawlQueue": [...],
  "freshnessData": [...]
}
```

### Configuration

**Save API keys**
```
POST /config/api-keys
Content-Type: application/json

{
  "llm": {
    "provider": "openai",
    "apiKey": "sk-...",
    "model": "gpt-4o-mini"
  },
  "embedding": {
    "provider": "openai",
    "apiKey": "sk-...",
    "model": "text-embedding-3-small"
  }
}
```

## Development

### Running Locally

```bash
cd services/api

# Install dependencies
pip install -r requirements.txt

# Run server
python main.py
```

Server will start on `http://localhost:7001`

### Running with Docker

```bash
# Build image
docker build -t owlerlite-api .

# Run container
docker run -p 7001:7001 \
  -e LIGHTRAG_URL=http://lightrag:9621 \
  -e CRAWLER_URL=http://crawler:8081 \
  -e FRONTIER_URL=http://frontier:7072 \
  owlerlite-api
```

### Environment Variables

- `LIGHTRAG_URL`: LightRAG service endpoint (default: `http://lightrag:9621`)
- `CRAWLER_URL`: Crawler service endpoint (default: `http://crawler:8081`)
- `FRONTIER_URL`: Frontier service endpoint (default: `http://frontier:7072`)
- `ORCHESTRATOR_URL`: Orchestrator service endpoint (default: `http://orchestrator:7000`)

## API Documentation

Interactive API documentation is available at:
- Swagger UI: `http://localhost:7001/docs`
- ReDoc: `http://localhost:7001/redoc`

## Data Storage

Currently uses in-memory storage for scopes and statistics. For production:

- Replace with PostgreSQL or MongoDB for persistence
- Add Redis for caching
- Implement proper session management
- Add authentication and authorization

## CORS Configuration

CORS is currently configured to allow all origins for development. In production:

```python
app.add_middleware(
    CORSMiddleware,
    allow_origins=["chrome-extension://*", "moz-extension://*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)
```

## Error Handling

Standard HTTP status codes:
- `200 OK`: Successful request
- `201 Created`: Resource created
- `400 Bad Request`: Invalid request data
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

Error response format:
```json
{
  "detail": "Error message"
}
```

## Security Considerations

### Current Implementation
- No authentication (suitable for local development)
- CORS allows all origins
- In-memory storage (no persistence)

### Production Requirements
- Add JWT authentication
- Implement rate limiting
- Use proper database with encryption
- Restrict CORS to extension origins only
- Add input validation and sanitization
- Implement audit logging

## Testing

```bash
# Install test dependencies
pip install pytest pytest-asyncio httpx

# Run tests
pytest tests/
```

## Dependencies

- **FastAPI**: Modern web framework
- **Uvicorn**: ASGI server
- **Pydantic**: Data validation
- **aiohttp**: Async HTTP client
- **python-multipart**: File upload support

## Performance

- Async/await for non-blocking I/O
- Connection pooling for backend services
- In-memory caching for frequent requests
- Automatic request/response compression

## Monitoring

Health check endpoint for container orchestration:
```bash
curl http://localhost:7001/health
```

## License

Part of the OwlerLite project. See main project for license information.

