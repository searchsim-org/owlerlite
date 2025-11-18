// Background service worker for OwlerLite extension

// Listen for extension installation
chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    console.log('OwlerLite extension installed');
    
    // Set default configuration
    chrome.storage.local.set({
      settings: {
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
      }
    });
  } else if (details.reason === 'update') {
    console.log('OwlerLite extension updated');
  }
});

// Listen for keyboard shortcut to open sidebar
chrome.commands.onCommand.addListener((command) => {
  if (command === 'open-sidebar') {
    openSidebar();
  }
});

// Listen for messages from popup or sidebar
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'GET_CURRENT_TAB') {
    chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
      if (tabs[0]) {
        sendResponse({
          url: tabs[0].url,
          title: tabs[0].title,
          id: tabs[0].id
        });
      } else {
        sendResponse({ error: 'No active tab found' });
      }
    });
    return true;
  }
  
  if (message.type === 'TRACK_PAGE') {
    trackPage(message.url, message.scopeId).then(sendResponse);
    return true;
  }
  
  if (message.type === 'OPEN_SIDEBAR') {
    openSidebar();
    return false;
  }
  
  if (message.type === 'OPEN_POPUP') {
    chrome.action.openPopup();
    return false;
  }
});

// Context menu for quick actions
chrome.contextMenus.create({
  id: 'add-to-owlerlite',
  title: 'Add to OwlerLite',
  contexts: ['page', 'link']
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
  if (info.menuItemId === 'add-to-owlerlite') {
    const url = info.linkUrl || info.pageUrl || tab.url;
    // Open sidebar in new tab
    openSidebar();
  }
});

// Helper functions
async function openSidebar() {
  try {
    // Check if sidebar tab is already open
    const tabs = await chrome.tabs.query({});
    const sidebarTab = tabs.find(tab => tab.url && tab.url.includes('sidebar.html'));
    
    if (sidebarTab) {
      // Focus existing sidebar tab
      await chrome.tabs.update(sidebarTab.id, { active: true });
      await chrome.windows.update(sidebarTab.windowId, { focused: true });
    } else {
      // Open new sidebar tab
      await chrome.tabs.create({
        url: chrome.runtime.getURL('sidebar.html'),
        active: true
      });
    }
  } catch (error) {
    console.error('Failed to open sidebar:', error);
  }
}

async function trackPage(url, scopeId) {
  try {
    const config = await chrome.storage.local.get('settings');
    const apiBaseUrl = config.settings?.apiEndpoint || 'http://localhost:7001';
    
    const response = await fetch(`${apiBaseUrl}/scopes/${scopeId}/pages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ url })
    });
    
    if (response.ok) {
      return { success: true };
    } else {
      const error = await response.json();
      return { success: false, error: error.message };
    }
  } catch (error) {
    return { success: false, error: error.message };
  }
}

// Periodic sync (optional)
chrome.alarms.create('sync-scopes', { periodInMinutes: 30 });

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'sync-scopes') {
    console.log('Syncing scopes...');
    // Implement sync logic if needed
  }
});
