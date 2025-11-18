// Configuration
let API_BASE_URL = 'http://localhost:7001';
let scopes = [];
let settings = {
  apiEndpoint: 'http://localhost:7001',
  llmProvider: 'openai',
  llmApiKey: '',
  llmModel: 'gpt-4o-mini',
  embeddingProvider: 'openai',
  embeddingApiKey: '',
  embeddingModel: 'text-embedding-3-small',
  showScores: true,
  showFreshness: true,
  enableAutoTrack: false
};

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
  await loadSettings();
  await checkConnection();
  await loadScopes();
  attachEventListeners();
});

// Settings
async function loadSettings() {
  try {
    const stored = await chrome.storage.local.get('settings');
    if (stored.settings) {
      settings = { ...settings, ...stored.settings };
      API_BASE_URL = settings.apiEndpoint;
      
      // Apply to UI
      document.getElementById('api-endpoint').value = settings.apiEndpoint;
      document.getElementById('llm-provider').value = settings.llmProvider;
      document.getElementById('llm-api-key').value = settings.llmApiKey;
      document.getElementById('llm-model').value = settings.llmModel;
      document.getElementById('embedding-provider').value = settings.embeddingProvider;
      document.getElementById('embedding-api-key').value = settings.embeddingApiKey;
      document.getElementById('embedding-model').value = settings.embeddingModel;
      document.getElementById('show-scores').checked = settings.showScores;
      document.getElementById('show-freshness').checked = settings.showFreshness;
      document.getElementById('enable-auto-track').checked = settings.enableAutoTrack;
    }
  } catch (error) {
    console.error('Failed to load settings:', error);
  }
}

async function saveSettings() {
  settings.apiEndpoint = document.getElementById('api-endpoint').value;
  settings.llmProvider = document.getElementById('llm-provider').value;
  settings.llmApiKey = document.getElementById('llm-api-key').value;
  settings.llmModel = document.getElementById('llm-model').value;
  settings.embeddingProvider = document.getElementById('embedding-provider').value;
  settings.embeddingApiKey = document.getElementById('embedding-api-key').value;
  settings.embeddingModel = document.getElementById('embedding-model').value;
  settings.showScores = document.getElementById('show-scores').checked;
  settings.showFreshness = document.getElementById('show-freshness').checked;
  settings.enableAutoTrack = document.getElementById('enable-auto-track').checked;
  
  API_BASE_URL = settings.apiEndpoint;
  await chrome.storage.local.set({ settings });
}

async function saveApiKeys() {
  await saveSettings();
  
  // Send to backend if needed
  try {
    await apiRequest('/config/api-keys', {
      method: 'POST',
      body: JSON.stringify({
        llm: {
          provider: settings.llmProvider,
          apiKey: settings.llmApiKey,
          model: settings.llmModel
        },
        embedding: {
          provider: settings.embeddingProvider,
          apiKey: settings.embeddingApiKey,
          model: settings.embeddingModel
        }
      })
    });
    showNotification('API keys saved successfully', 'success');
  } catch (error) {
    showNotification('API keys saved locally (backend unavailable)', 'info');
  }
}

// Connection
async function checkConnection() {
  const dot = document.getElementById('connection-status');
  const label = document.getElementById('connection-label');
  
  try {
    const response = await fetch(`${API_BASE_URL}/health`, {
      method: 'GET',
      signal: AbortSignal.timeout(3000)
    });
    
    if (response.ok) {
      dot.classList.add('online');
      dot.classList.remove('offline');
      label.textContent = 'Connected';
      return true;
    } else {
      throw new Error('Backend error');
    }
  } catch (error) {
    dot.classList.add('offline');
    dot.classList.remove('online');
    label.textContent = 'Offline';
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
    updateStats();
  } catch (error) {
    scopes = [];
    updateStats();
  }
}

function updateStats() {
  const totalPages = scopes.reduce((sum, scope) => sum + (scope.pageCount || 0), 0);
  document.getElementById('total-scopes').textContent = scopes.length;
  document.getElementById('total-pages').textContent = totalPages;
}

function showScopesModal() {
  renderScopesList();
  document.getElementById('scopes-modal').classList.remove('hidden');
}

function hideScopesModal() {
  document.getElementById('scopes-modal').classList.add('hidden');
}

function renderScopesList() {
  const container = document.getElementById('scopes-list');
  
  if (scopes.length === 0) {
    container.innerHTML = '<p style="text-align: center; color: var(--muted); padding: 20px;">No scopes defined</p>';
    return;
  }
  
  container.innerHTML = scopes.map(scope => `
    <div class="scope-item">
      <div class="scope-item-info">
        <div class="scope-item-name">${escapeHtml(scope.name)}</div>
        <div class="scope-item-meta">
          ${scope.patterns ? scope.patterns.length : 0} patterns | 
          ${scope.pageCount || 0} pages
        </div>
      </div>
      <div class="scope-item-actions">
        <button onclick="editScope('${scope.id}')">Edit</button>
        <button onclick="deleteScope('${scope.id}')">Delete</button>
      </div>
    </div>
  `).join('');
}

// Scope CRUD
function showScopeModal(scope = null) {
  const modal = document.getElementById('scope-modal');
  const title = document.getElementById('modal-title');
  
  if (scope) {
    title.textContent = 'Edit Scope';
    document.getElementById('scope-id').value = scope.id;
    document.getElementById('scope-name').value = scope.name;
    document.getElementById('scope-description').value = scope.description || '';
    document.getElementById('scope-patterns').value = (scope.patterns || []).join('\n');
    document.getElementById('scope-auto-track').checked = scope.autoTrack || false;
  } else {
    title.textContent = 'New Scope';
    document.getElementById('scope-id').value = '';
    document.getElementById('scope-name').value = '';
    document.getElementById('scope-description').value = '';
    document.getElementById('scope-patterns').value = '';
    document.getElementById('scope-auto-track').checked = false;
  }
  
  modal.classList.remove('hidden');
}

function hideScopeModal() {
  document.getElementById('scope-modal').classList.add('hidden');
}

async function saveScope() {
  const id = document.getElementById('scope-id').value;
  const name = document.getElementById('scope-name').value.trim();
  const description = document.getElementById('scope-description').value.trim();
  const patternsText = document.getElementById('scope-patterns').value.trim();
  const patterns = patternsText ? patternsText.split('\n').map(p => p.trim()).filter(p => p) : [];
  const autoTrack = document.getElementById('scope-auto-track').checked;
  
  if (!name) {
    showNotification('Scope name is required', 'error');
    return;
  }
  
  try {
    const scopeData = { name, description, patterns, autoTrack };
    
    if (id) {
      await apiRequest(`/scopes/${id}`, {
        method: 'PUT',
        body: JSON.stringify(scopeData)
      });
      showNotification('Scope updated', 'success');
    } else {
      await apiRequest('/scopes', {
        method: 'POST',
        body: JSON.stringify(scopeData)
      });
      showNotification('Scope created', 'success');
    }
    
    hideScopeModal();
    await loadScopes();
    if (document.getElementById('scopes-modal').classList.contains('hidden') === false) {
      renderScopesList();
    }
  } catch (error) {
    showNotification(`Failed: ${error.message}`, 'error');
  }
}

function editScope(id) {
  const scope = scopes.find(s => s.id === id);
  if (scope) {
    hideScopesModal();
    showScopeModal(scope);
  }
}

async function deleteScope(id) {
  if (!confirm('Delete this scope?')) return;
  
  try {
    await apiRequest(`/scopes/${id}`, { method: 'DELETE' });
    showNotification('Scope deleted', 'success');
    await loadScopes();
    renderScopesList();
  } catch (error) {
    showNotification(`Failed: ${error.message}`, 'error');
  }
}

// Data Management
function exportData() {
  const data = {
    scopes,
    settings,
    exportDate: new Date().toISOString()
  };
  
  const json = JSON.stringify(data, null, 2);
  const blob = new Blob([json], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `owlerlite-config-${Date.now()}.json`;
  a.click();
  URL.revokeObjectURL(url);
}

function importData() {
  const input = document.createElement('input');
  input.type = 'file';
  input.accept = '.json';
  input.onchange = async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    
    const reader = new FileReader();
    reader.onload = async (event) => {
      try {
        const data = JSON.parse(event.target.result);
        
        if (data.settings) {
          settings = { ...settings, ...data.settings };
          await chrome.storage.local.set({ settings });
          await loadSettings();
        }
        
        showNotification('Data imported successfully', 'success');
      } catch (error) {
        showNotification(`Import failed: ${error.message}`, 'error');
      }
    };
    reader.readAsText(file);
  };
  input.click();
}

function clearCache() {
  if (confirm('Clear all local cache?')) {
    chrome.storage.local.clear(() => {
      showNotification('Cache cleared', 'success');
      setTimeout(() => {
        window.location.reload();
      }, 1000);
    });
  }
}

// Sidebar
async function openSidebar() {
  try {
    chrome.runtime.sendMessage({ type: 'OPEN_SIDEBAR' });
  } catch (error) {
    console.error('Failed to open sidebar:', error);
    showNotification('Failed to open sidebar', 'error');
  }
}

// Utilities
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

function showNotification(message, type = 'info') {
  // Simple notification - could be enhanced
  alert(message);
}

// Make functions globally accessible
window.editScope = editScope;
window.deleteScope = deleteScope;

// Event Listeners
function attachEventListeners() {
  // Tab navigation
  const tabBtns = document.querySelectorAll('.popup-tab');
  tabBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      const tabName = btn.dataset.tab;
      
      // Update buttons
      tabBtns.forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      
      // Update content
      document.querySelectorAll('.popup-tab-content').forEach(content => {
        content.classList.remove('active');
      });
      document.getElementById(`${tabName}-tab`).classList.add('active');
    });
  });
  
  // Connection
  document.getElementById('test-connection-btn').addEventListener('click', async () => {
    await saveSettings();
    await checkConnection();
  });
  
  // API endpoint auto-save
  document.getElementById('api-endpoint').addEventListener('change', saveSettings);
  
  // API Keys
  document.getElementById('save-api-keys-btn').addEventListener('click', saveApiKeys);
  
  // Scopes
  document.getElementById('view-scopes-btn').addEventListener('click', showScopesModal);
  document.getElementById('new-scope-btn').addEventListener('click', () => showScopeModal());
  document.getElementById('close-scopes-modal-btn').addEventListener('click', hideScopesModal);
  
  // Scope Modal
  document.getElementById('save-scope-btn').addEventListener('click', saveScope);
  document.getElementById('cancel-scope-btn').addEventListener('click', hideScopeModal);
  document.getElementById('close-modal-btn').addEventListener('click', hideScopeModal);
  
  // Settings auto-save
  document.getElementById('show-scores').addEventListener('change', saveSettings);
  document.getElementById('show-freshness').addEventListener('change', saveSettings);
  document.getElementById('enable-auto-track').addEventListener('change', saveSettings);
  
  // Data Management
  document.getElementById('export-data-btn').addEventListener('click', exportData);
  document.getElementById('import-data-btn').addEventListener('click', importData);
  document.getElementById('clear-cache-btn').addEventListener('click', clearCache);
  
  // Sidebar
  document.getElementById('open-sidebar-btn').addEventListener('click', openSidebar);
  
  // Close modals on background click
  document.getElementById('scope-modal').addEventListener('click', (e) => {
    if (e.target.id === 'scope-modal') hideScopeModal();
  });
  document.getElementById('scopes-modal').addEventListener('click', (e) => {
    if (e.target.id === 'scopes-modal') hideScopesModal();
  });
}
