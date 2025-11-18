# OwlerLite Browser Extension Installation Guide

This guide provides detailed instructions for installing and configuring the OwlerLite browser extension.

---

## Prerequisites

Before installing the extension, ensure the following requirements are met:

- **Backend services running**: The OwlerLite backend must be operational
- **Modern web browser**: Firefox (recommended), Chrome, or Edge
- **API access**: OpenAI API key or compatible LLM endpoint

Verify the backend is running:
```bash
cd /path/to/owlerlite
make up

# Test the orchestrator endpoint
curl http://localhost:7001/health
```

Expected response: `{"status":"ok"}`

---

## Installation Instructions

### Firefox (Recommended)

Firefox provides the most stable experience for Manifest V3 extensions with the current implementation.

**Step 1: Open Firefox Developer Tools**
```
about:debugging#/runtime/this-firefox
```

**Step 2: Load Temporary Add-on**
1. Click "Load Temporary Add-on..."
2. Navigate to: `/path/to/owlerlite/services/extension/dist`
3. Select the `manifest.json` file
4. Click "Open"

**Step 3: Verify Installation**
- The OwlerLite icon should appear in your browser toolbar
- Extension will remain loaded until Firefox restarts
- For permanent installation, the extension requires Mozilla signing

**Note**: Temporary add-ons are removed when Firefox closes. You will need to reload the extension after each browser restart.

### Chrome

Chrome supports the extension through its developer mode for unpacked extensions.

**Step 1: Open Extensions Page**
```
chrome://extensions/
```

**Step 2: Enable Developer Mode**
- Toggle "Developer mode" in the top right corner

**Step 3: Load Unpacked Extension**
1. Click "Load unpacked"
2. Navigate to: `/path/to/owlerlite/services/extension/dist`
3. Select the `dist` folder (not manifest.json)
4. Click "Select"

**Step 4: Verify Installation**
- OwlerLite should appear in your extensions list
- Pin the extension to your toolbar for easy access

### Edge

Edge uses the same process as Chrome for loading unpacked extensions.

**Step 1: Open Extensions Page**
```
edge://extensions/
```

**Step 2: Enable Developer Mode**
- Toggle "Developer mode" in the left sidebar

**Step 3: Load Unpacked Extension**
1. Click "Load unpacked"
2. Navigate to: `/path/to/owlerlite/services/extension/dist`
3. Select the `dist` folder
4. Click "Select Folder"

---

## Configuration

### Initial Setup

After installing the extension, configure it to connect to your backend.

**Step 1: Open Configuration**
- Click the OwlerLite icon in your browser toolbar
- The popup will open with the Configuration tab active

**Step 2: Backend Connection**
```
API Endpoint: http://localhost:7001
```
- Enter your backend URL
- Click "Test Connection"
- Verify the status indicator shows "Connected" (green dot)

**Step 3: LLM API Keys**

Configure your language model provider:

```
LLM Provider: OpenAI (or Anthropic/Local)
LLM API Key: sk-...
LLM Model: gpt-4o-mini

Embedding Provider: OpenAI (or Cohere/Local)
Embedding API Key: sk-...
Embedding Model: text-embedding-3-small
```

- Enter your API credentials
- Click "Save API Keys"
- Keys are encrypted and stored locally

**Step 4: Retrieval Settings**

Configure display preferences:
- **Score Breakdown**: Show detailed scoring in results
- **Freshness Indicators**: Display semantic freshness badges
- **Auto-tracking**: Automatically add pages matching scope patterns

### Keyboard Shortcut

The sidebar can be opened using:
```
Ctrl+Shift+O (Windows/Linux)
Cmd+Shift+O (macOS)
```

Or click "Open Sidebar" in the Configuration tab.

---

## Usage

### Creating Your First Scope

**Step 1: Navigate to Scopes Tab**
- Click the OwlerLite icon
- Switch to the "Scopes" tab

**Step 2: Create New Scope**
1. Click "New Scope"
2. Enter scope details:
   - **Name**: e.g., "Python Documentation"
   - **Description**: What this scope contains
   - **URL Patterns**: One per line
     ```
     https://docs.python.org/*
     https://realpython.com/*
     ```
   - **Auto-track**: Enable to automatically add matching pages

**Step 3: Save Scope**
- Click "Save"
- Scope appears in the scopes list

### Querying with Scopes

**Step 1: Open Sidebar**
- Press `Ctrl+Shift+O`
- Or click "Open Sidebar" in popup

**Step 2: Select Scopes**
- Click scope chips at the top to activate/deactivate
- Multiple scopes can be selected simultaneously

**Step 3: Ask Questions**
- Type your question in the input field
- Press `Ctrl+Enter` or click send button
- Results appear inline in the conversation

**Step 4: Review Results**
- Each result shows:
  - Title and URL
  - Relevant snippet
  - Score and metadata
  - Scope badges
  - Freshness indicators

### Adding Pages to Scopes

**Method 1: Current Page**
1. Navigate to a page you want to index
2. Click OwlerLite icon
3. Go to "Scopes" tab
4. Select target scope from list
5. Page is added and queued for crawling

**Method 2: Right-Click Menu**
1. Right-click on any page or link
2. Select "Add to OwlerLite"
3. Sidebar opens for scope selection

**Method 3: URL Patterns**
- Define patterns in scope configuration
- Enable auto-tracking
- Pages matching patterns are automatically added when visited

---

## Updating the Extension

When you make changes to the extension code:

**Step 1: Rebuild**
```bash
cd services/extension
./build.sh
```

**Step 2: Reload in Browser**

**Firefox:**
- Go to `about:debugging#/runtime/this-firefox`
- Click "Reload" next to OwlerLite

**Chrome/Edge:**
- Go to extensions page
- Click the refresh icon on the OwlerLite card

---

## Troubleshooting

### Backend Connection Failed

**Symptoms**: Red status indicator, "Backend offline" message

**Solutions**:
1. Verify backend services are running:
   ```bash
   docker ps
   ```
2. Check orchestrator is accessible:
   ```bash
   curl http://localhost:7001/health
   ```
3. Verify API endpoint in configuration matches your setup
4. Check firewall settings allow localhost connections

### Extension Won't Load

**Symptoms**: Error messages during installation

**Solutions**:
1. Verify all files exist in `dist/` directory:
   ```bash
   ls -la services/extension/dist/
   ```
2. Rebuild the extension:
   ```bash
   cd services/extension
   ./build.sh
   ```
3. Check browser console for specific errors
4. Ensure you're loading the `dist` folder, not `src`

### No Results Returned

**Symptoms**: Queries complete but return empty results

**Solutions**:
1. Verify scopes have pages added:
   - Check scope statistics in popup
   - Confirm page count > 0
2. Wait for initial crawling to complete:
   - Check "Active Crawls" in Statistics tab
   - Initial indexing may take several minutes
3. Verify selected scopes contain relevant content
4. Check backend logs:
   ```bash
   make logs
   ```

### Sidebar Won't Open

**Symptoms**: Keyboard shortcut or button doesn't open sidebar

**Solutions**:
1. Check browser permissions:
   - Extension has "tabs" permission
   - No conflicting keyboard shortcuts
2. Try opening manually:
   - Click "Open Sidebar" button in popup
3. Check browser console:
   - Right-click extension icon → Inspect popup
   - Look for JavaScript errors

### API Keys Not Saving

**Symptoms**: Keys don't persist after browser restart

**Solutions**:
1. Verify storage permission granted
2. Re-enter keys and click "Save API Keys"
3. Check browser storage:
   - Firefox: Browser Console → Storage → Extension Storage
   - Chrome: Developer Tools → Application → Storage
4. Clear cache and reconfigure:
   - Statistics tab → "Clear Cache"

---

## Advanced Configuration

### Custom Backend URL

For deployments on different hosts or ports:

```javascript
// Configuration tab
API Endpoint: https://your-server.com:8080
```

Test the connection after changing the endpoint.

### Local LLM Integration

To use local language models:

1. Set LLM Provider to "Local"
2. Configure local endpoint in backend
3. Update API endpoint to point to your local LLM server
4. No API key required for local models

### Performance Tuning

For better performance on slower connections:

1. Reduce concurrent crawls in backend configuration
2. Adjust chunk sizes for faster processing
3. Limit number of scopes queried simultaneously
4. Use local embeddings for faster retrieval

---

## Security Considerations

### Data Storage

- **API Keys**: Encrypted using Web Crypto API with AES-GCM
- **Settings**: Stored in browser's local storage
- **Query History**: Stored locally, never transmitted
- **Scope Data**: Metadata only, actual content on backend

### Network Communications

- All requests to `http://localhost:7001` by default
- HTTPS recommended for production deployments
- No telemetry or analytics data collected
- User data never leaves configured infrastructure

### Privacy Settings

Control what data is collected:

1. **Auto-tracking**: Disable to manually control indexed pages
2. **Query History**: Clear regularly in Statistics tab
3. **API Keys**: Use separate keys for different projects
4. **Scope Export**: Review exported data before sharing

---

## Uninstalling

### Remove Extension

**Firefox:**
```
about:addons → Extensions → OwlerLite → Remove
```

**Chrome/Edge:**
```
chrome://extensions/ → OwlerLite → Remove
```

### Clean Up Data

To remove all stored data:

1. Before uninstalling, use "Clear Cache" in Statistics tab
2. Or manually clear extension storage:
   - Firefox: Settings → Privacy & Security → Clear Data → Extension Data
   - Chrome: Settings → Privacy and security → Clear browsing data → Advanced → Hosted app data

### Backend Cleanup

To remove backend data:

```bash
cd /path/to/owlerlite
make down       # Stop services
make clean      # Remove volumes and data
```

---

## Getting Help

### Documentation

- **Main README**: `/path/to/owlerlite/README.md`
- **Extension README**: `services/extension/README.md` (if exists)
- **Backend Documentation**: Service-specific READMEs in `services/`

### Debugging

Enable debug logging:

1. Open browser developer console
2. Monitor extension activity:
   - Background script logs
   - Sidebar page logs
   - Popup logs
3. Check network requests in Network tab

### Common Error Messages

**"Sidebar not supported in this browser"**
- This is expected; sidebar opens in a new tab instead

**"Backend unavailable"**
- Backend services not running or endpoint misconfigured

**"Failed to add page"**
- URL may be blocked by robots.txt
- Backend crawl queue may be full

---

## Version Information

Current version: **1.0.0**

**Compatibility:**
- Firefox 109+ (Manifest V3 support)
- Chrome 88+
- Edge 88+

**Backend Requirements:**
- OwlerLite orchestrator v1.0+
- LightRAG integration
- URLFrontier protocol support

---

## Feedback and Contributions

This extension is part of the OwlerLite research project. For issues, suggestions, or contributions, please refer to the main project repository.
