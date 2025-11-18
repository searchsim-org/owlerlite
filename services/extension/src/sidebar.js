// Configuration
let API_BASE_URL = 'http://localhost:7001';

// State
let scopes = [];
let selectedScopes = [];
let conversation = [];
let stats = {};

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
  await loadSettings();
  await checkBackendStatus();
  await loadScopes();
  await loadStats();
  attachEventListeners();
  
  // Auto-resize textarea
  const textarea = document.getElementById('query-input');
  textarea.addEventListener('input', autoResizeTextarea);
});

async function loadSettings() {
  try {
    const stored = await chrome.storage.local.get('settings');
    if (stored.settings) {
      API_BASE_URL = stored.settings.apiEndpoint || 'http://localhost:7001';
    }
  } catch (error) {
    console.error('Failed to load settings:', error);
  }
}

async function checkBackendStatus() {
  const dot = document.getElementById('status-dot');
  const text = document.getElementById('status-text');
  
  try {
    const response = await fetch(`${API_BASE_URL}/health`, {
      method: 'GET',
      signal: AbortSignal.timeout(3000)
    });
    
    if (response.ok) {
      dot.classList.add('online');
      dot.classList.remove('offline');
      text.textContent = 'Connected';
      return true;
    } else {
      throw new Error('Backend error');
    }
  } catch (error) {
    dot.classList.add('offline');
    dot.classList.remove('online');
    text.textContent = 'Offline';
    return false;
  }
}

async function apiRequest(endpoint, options = {}) {
  try {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      }
    });
    
    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('API request failed:', error);
    throw error;
  }
}

// Scopes
async function loadScopes() {
  try {
    const data = await apiRequest('/scopes');
    scopes = data.scopes || [];
    renderScopeChips();
  } catch (error) {
    console.error('Failed to load scopes:', error);
    scopes = [];
    renderScopeChips();
  }
}

function renderScopeChips() {
  const container = document.getElementById('scope-chips');
  
  if (scopes.length === 0) {
    container.innerHTML = '<span class="empty-text">No scopes defined</span>';
    return;
  }
  
  container.innerHTML = scopes.map(scope => `
    <div class="scope-chip ${selectedScopes.includes(scope.id) ? 'active' : ''}" 
         data-id="${scope.id}"
         onclick="toggleScope('${scope.id}')">
      ${escapeHtml(scope.name)}
    </div>
  `).join('');
}

function toggleScope(scopeId) {
  if (selectedScopes.includes(scopeId)) {
    selectedScopes = selectedScopes.filter(id => id !== scopeId);
  } else {
    selectedScopes.push(scopeId);
  }
  renderScopeChips();
}

// Conversation
function addMessage(role, content, results = null) {
  const message = {
    role,
    content,
    results,
    timestamp: Date.now()
  };
  
  conversation.push(message);
  renderConversation();
  
  // Scroll to bottom
  const container = document.getElementById('conversation-container');
  setTimeout(() => {
    container.scrollTop = container.scrollHeight;
  }, 100);
}

function renderConversation() {
  const container = document.getElementById('conversation-container');
  
  if (conversation.length === 0) {
    container.innerHTML = '<div class="empty-state"><p class="empty-text">Select scopes and start a conversation</p></div>';
    return;
  }
  
  container.innerHTML = conversation.map((msg, index) => {
    if (msg.role === 'user') {
      return `
        <div class="message message-user">
          <div class="message-bubble">${escapeHtml(msg.content)}</div>
          <div class="message-meta">${formatTime(msg.timestamp)}</div>
        </div>
      `;
    } else {
      return `
        <div class="message message-assistant">
          <div class="message-bubble">${escapeHtml(msg.content)}</div>
          ${msg.results && msg.results.length > 0 ? `
            <div class="result-items">
              ${msg.results.slice(0, 3).map(result => `
                <div class="result-item" onclick="window.open('${result.url}', '_blank')">
                  <a href="${result.url}" target="_blank" class="result-title" onclick="event.stopPropagation()">
                    ${escapeHtml(result.title || result.url)}
                  </a>
                  <div class="result-snippet">${escapeHtml(result.snippet || result.content || '')}</div>
                  <div class="result-meta">
                    <span>Score: ${result.score ? result.score.toFixed(2) : 'N/A'}</span>
                    ${result.version ? `<span>v${result.version}</span>` : ''}
                  </div>
                </div>
              `).join('')}
              ${msg.results.length > 3 ? `
                <div class="empty-text" style="text-align: center; padding: 8px;">
                  +${msg.results.length - 3} more results
                </div>
              ` : ''}
            </div>
          ` : ''}
          <div class="message-meta">${formatTime(msg.timestamp)}</div>
        </div>
      `;
    }
  }).join('');
}

async function sendQuery() {
  const input = document.getElementById('query-input');
  const query = input.value.trim();
  
  if (!query) return;
  
  if (selectedScopes.length === 0) {
    showNotification('Please select at least one scope');
    return;
  }
  
  // Add user message
  addMessage('user', query);
  input.value = '';
  input.style.height = 'auto';
  
  // Show loading
  const sendBtn = document.getElementById('send-btn');
  sendBtn.disabled = true;
  sendBtn.innerHTML = '<div class="loading"></div>';
  
  try {
    const data = await apiRequest('/query', {
      method: 'POST',
      body: JSON.stringify({
        query,
        scopes: selectedScopes
      })
    });
    
    // Add assistant response
    const answer = data.answer || 'I found the following results:';
    addMessage('assistant', answer, data.results || []);
    
  } catch (error) {
    // Fallback to placeholder
    addMessage('assistant', 'Backend unavailable. Here are example results:', [
      {
        title: 'Example Result',
        url: 'https://example.com',
        snippet: 'This is a placeholder result showing the conversation interface.',
        score: 0.85,
        version: '1'
      }
    ]);
  } finally {
    sendBtn.disabled = false;
    sendBtn.innerHTML = `
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="22" y1="2" x2="11" y2="13"></line>
        <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
      </svg>
    `;
  }
}

// Stats
async function loadStats() {
  try {
    const data = await apiRequest('/stats');
    stats = data;
    updateStatsModal();
  } catch (error) {
    console.error('Failed to load stats:', error);
    stats = {
      totalScopes: scopes.length,
      totalPages: 0,
      activeCrawls: 0,
      pendingUpdates: 0,
      recentActivity: [],
      crawlQueue: [],
      freshnessData: []
    };
  }
}

function updateStatsModal() {
  document.getElementById('stat-scopes').textContent = stats.totalScopes || scopes.length;
  document.getElementById('stat-pages').textContent = stats.totalPages || 0;
  document.getElementById('stat-crawls').textContent = stats.activeCrawls || 0;
  document.getElementById('stat-pending').textContent = stats.pendingUpdates || 0;
  
  // Recent activity
  const activityContainer = document.getElementById('recent-activity');
  if (stats.recentActivity && stats.recentActivity.length > 0) {
    activityContainer.innerHTML = stats.recentActivity.map(activity => `
      <div class="activity-item">
        <div class="activity-item-title">${escapeHtml(activity.description)}</div>
        <div class="activity-item-time">${formatTime(activity.timestamp)}</div>
      </div>
    `).join('');
  } else {
    activityContainer.innerHTML = '<p class="empty-text">No recent activity</p>';
  }
  
  // Crawl queue
  const queueContainer = document.getElementById('crawl-queue');
  if (stats.crawlQueue && stats.crawlQueue.length > 0) {
    queueContainer.innerHTML = stats.crawlQueue.map(item => `
      <div class="queue-item">
        <div class="activity-item-title">${escapeHtml(item.url)}</div>
        <div class="activity-item-time">Scope: ${escapeHtml(item.scope)}</div>
      </div>
    `).join('');
  } else {
    queueContainer.innerHTML = '<p class="empty-text">No pending crawls</p>';
  }
  
  // Freshness
  const freshnessContainer = document.getElementById('freshness-overview');
  if (stats.freshnessData && stats.freshnessData.length > 0) {
    freshnessContainer.innerHTML = stats.freshnessData.map(item => `
      <div class="activity-item">
        <div class="activity-item-title">${escapeHtml(item.scope)}</div>
        <div class="activity-item-time">
          Fresh: ${item.fresh || 0} | Stale: ${item.stale || 0}
        </div>
      </div>
    `).join('');
  } else {
    freshnessContainer.innerHTML = '<p class="empty-text">No freshness data available</p>';
  }
}

function showStats() {
  loadStats();
  document.getElementById('stats-modal').classList.remove('hidden');
}

function hideStats() {
  document.getElementById('stats-modal').classList.add('hidden');
}

// Scope Manager
function showScopeManager() {
  renderScopesList();
  document.getElementById('scope-manager-modal').classList.remove('hidden');
}

function hideScopeManager() {
  document.getElementById('scope-manager-modal').classList.add('hidden');
}

function renderScopesList() {
  const container = document.getElementById('scopes-list');
  
  if (scopes.length === 0) {
    container.innerHTML = '<p class="empty-text">No scopes defined</p>';
    return;
  }
  
  container.innerHTML = scopes.map(scope => `
    <label class="scope-list-item">
      <div>
        <div class="scope-list-item-name">${escapeHtml(scope.name)}</div>
        <div class="scope-list-item-meta">
          ${scope.patterns ? scope.patterns.length : 0} patterns | 
          ${scope.pageCount || 0} pages
        </div>
      </div>
      <input 
        type="checkbox" 
        ${selectedScopes.includes(scope.id) ? 'checked' : ''}
        onchange="toggleScopeFromList('${scope.id}')"
      >
    </label>
  `).join('');
}

function toggleScopeFromList(scopeId) {
  toggleScope(scopeId);
  renderScopesList();
}

function openPopup() {
  chrome.runtime.sendMessage({ type: 'OPEN_POPUP' });
}

// Utilities
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

function formatTime(timestamp) {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now - date;
  
  if (diff < 60000) return 'Just now';
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return date.toLocaleDateString();
}

function autoResizeTextarea() {
  const textarea = document.getElementById('query-input');
  textarea.style.height = 'auto';
  textarea.style.height = textarea.scrollHeight + 'px';
}

function showNotification(message) {
  // Simple notification in status bar
  const statusText = document.getElementById('status-text');
  const originalText = statusText.textContent;
  statusText.textContent = message;
  setTimeout(() => {
    statusText.textContent = originalText;
  }, 3000);
}

// Make functions globally accessible
window.toggleScope = toggleScope;
window.toggleScopeFromList = toggleScopeFromList;

// Event Listeners
function attachEventListeners() {
  document.getElementById('send-btn').addEventListener('click', sendQuery);
  document.getElementById('query-input').addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && e.ctrlKey) {
      e.preventDefault();
      sendQuery();
    }
  });
  
  document.getElementById('refresh-btn').addEventListener('click', async () => {
    await checkBackendStatus();
    await loadScopes();
    await loadStats();
    showNotification('Refreshed');
  });
  
  document.getElementById('stats-btn').addEventListener('click', showStats);
  document.getElementById('close-stats-btn').addEventListener('click', hideStats);
  
  document.getElementById('manage-scopes-btn').addEventListener('click', showScopeManager);
  document.getElementById('close-scope-manager-btn').addEventListener('click', hideScopeManager);
  
  document.getElementById('new-scope-btn').addEventListener('click', () => {
    hideScopeManager();
    openPopup();
  });
  
  // Close modals on background click
  document.getElementById('stats-modal').addEventListener('click', (e) => {
    if (e.target.id === 'stats-modal') hideStats();
  });
  
  document.getElementById('scope-manager-modal').addEventListener('click', (e) => {
    if (e.target.id === 'scope-manager-modal') hideScopeManager();
  });
}

