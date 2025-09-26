const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';
const WS_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8080/ws';

class ApiClient {
  async createDownload(data) {
    const response = await fetch(`${API_BASE_URL}/downloads`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    return response.json();
  }

  async getDownloads() {
    const response = await fetch(`${API_BASE_URL}/downloads`);
    return response.json();
  }

  async pauseDownload(id) {
    await fetch(`${API_BASE_URL}/downloads/${id}/pause`, {
      method: 'POST',
    });
  }

  async resumeDownload(id) {
    await fetch(`${API_BASE_URL}/downloads/${id}/resume`, {
      method: 'POST',
    });
  }

  async deleteDownload(id) {
    await fetch(`${API_BASE_URL}/downloads/${id}`, {
      method: 'DELETE',
    });
  }

  async getSettings() {
    const response = await fetch(`${API_BASE_URL}/settings`);
    return response.json();
  }

  async updateSettings(settings) {
    await fetch(`${API_BASE_URL}/settings`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(settings),
    });
  }

  connectWebSocket(onMessage) {
    const ws = new WebSocket(WS_URL);
    
    ws.onopen = () => {
      console.log('WebSocket connected');
    };
    
    ws.onmessage = (event) => {
      const update = JSON.parse(event.data);
      onMessage(update);
    };
    
    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
    
    ws.onclose = () => {
      console.log('WebSocket disconnected');
      // Attempt to reconnect after 3 seconds
      setTimeout(() => this.connectWebSocket(onMessage), 3000);
    };
    
    return ws;
  }
}

export default new ApiClient();