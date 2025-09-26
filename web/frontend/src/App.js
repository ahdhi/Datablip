import React, { useState, useEffect, useRef } from 'react';
import { 
  Download, Plus, Trash2, Pause, Play, Settings, 
  Clock, Zap, HardDrive, X, CheckCircle, AlertCircle, 
  Loader, FolderOpen, Globe,
  Activity, BarChart3, FileDown, Link2, Menu, Bell,
  Wifi, WifiOff
} from 'lucide-react';
import apiClient from './api/client';
import './App.css';

const DatablipUI = () => {
  const [downloads, setDownloads] = useState([]);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [globalSettings, setGlobalSettings] = useState({
    defaultChunks: 4,
    connectTimeout: '30s',
    readTimeout: '10m',
    autoStart: true,
    maxConcurrentDownloads: 3
  });
  
  const [newDownload, setNewDownload] = useState({
    url: '',
    output: '',
    chunks: 4,
    connectTimeout: '30s',
    readTimeout: '10m'
  });

  const [activeTab, setActiveTab] = useState('active');
  const [selectedDownload, setSelectedDownload] = useState(null);
  const wsRef = useRef(null);

  // Initialize connection to backend
  useEffect(() => {
    initializeApp();
    
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  const initializeApp = async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Load initial settings
      await loadSettings();
      
      // Load existing downloads
      await loadDownloads();
      
      // Connect to WebSocket for real-time updates
      connectWebSocket();
      
    } catch (err) {
      console.error('Failed to initialize app:', err);
      setError('Failed to connect to server. Please make sure the backend is running.');
    } finally {
      setLoading(false);
    }
  };

  const loadSettings = async () => {
    try {
      const settings = await apiClient.getSettings();
      setGlobalSettings(settings);
      // Update new download defaults with global settings
      setNewDownload(prev => ({
        ...prev,
        chunks: settings.defaultChunks,
        connectTimeout: settings.connectTimeout,
        readTimeout: settings.readTimeout
      }));
    } catch (error) {
      console.error('Failed to load settings:', error);
    }
  };

  const loadDownloads = async () => {
    try {
      const downloadsList = await apiClient.getDownloads();
      setDownloads(downloadsList || []);
    } catch (error) {
      console.error('Failed to load downloads:', error);
      setDownloads([]);
    }
  };

  const downloadCompletedFile = async (downloadId) => {
    try {
      // Create a temporary anchor element to trigger download
      const link = document.createElement('a');
      link.href = `http://localhost:8080/api/downloads/${downloadId}/file`;
      link.download = ''; // This will use the filename from Content-Disposition header
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } catch (error) {
      console.error('Failed to download completed file:', error);
    }
  };

  const connectWebSocket = () => {
    wsRef.current = apiClient.connectWebSocket((update) => {
      handleWebSocketUpdate(update);
    });
  };

  const handleWebSocketUpdate = (update) => {
    console.log('WebSocket update received:', update);
    
    switch (update.type) {
      case 'progress':
        console.log('Progress update data:', update.data);
        console.log('ChunkProgress:', update.data?.chunkProgress);
        setDownloads(prev => prev.map(d => 
          d.id === update.downloadId ? { ...d, ...update.data } : d
        ));
        break;
        
      case 'status':
        setDownloads(prev => prev.map(d => 
          d.id === update.downloadId ? { ...d, ...update.data } : d
        ));
        break;
        
      case 'completed':
        setDownloads(prev => prev.map(d => 
          d.id === update.downloadId 
            ? { ...d, status: 'completed', progress: 100, speed: 0 } 
            : d
        ));
        // Automatically trigger download of completed file
        downloadCompletedFile(update.downloadId);
        break;
        
      case 'error':
        setDownloads(prev => prev.map(d => 
          d.id === update.downloadId 
            ? { ...d, status: 'error', error: update.data.error } 
            : d
        ));
        break;
        
      case 'paused':
        setDownloads(prev => prev.map(d => 
          d.id === update.downloadId 
            ? { ...d, status: 'paused', speed: 0 } 
            : d
        ));
        break;
        
      case 'resumed':
        setDownloads(prev => prev.map(d => 
          d.id === update.downloadId 
            ? { ...d, status: 'downloading' } 
            : d
        ));
        break;
        
      default:
        console.log('Unknown update type:', update.type);
    }
  };

  const addDownload = async () => {
    if (!newDownload.url) {
      alert('Please enter a URL');
      return;
    }
    
    try {
      setError(null);
      
      const downloadData = {
        url: newDownload.url,
        filename: newDownload.output || newDownload.url.split('/').pop() || 'download',
        chunks: parseInt(newDownload.chunks) || globalSettings.defaultChunks,
        connectTimeout: newDownload.connectTimeout || globalSettings.connectTimeout,
        readTimeout: newDownload.readTimeout || globalSettings.readTimeout,
      };
      
      console.log('Creating download:', downloadData);
      const download = await apiClient.createDownload(downloadData);
      
      // Add to local state
      setDownloads(prev => [...prev, download]);
      
      // Reset form
      setNewDownload({
        url: '',
        output: '',
        chunks: globalSettings.defaultChunks,
        connectTimeout: globalSettings.connectTimeout,
        readTimeout: globalSettings.readTimeout
      });
      
      setShowAddModal(false);
      
    } catch (error) {
      console.error('Failed to create download:', error);
      setError('Failed to create download. Please check the URL and try again.');
    }
  };

  const toggleDownload = async (id) => {
    const download = downloads.find(d => d.id === id);
    if (!download) return;
    
    try {
      setError(null);
      
      if (download.status === 'downloading') {
        await apiClient.pauseDownload(id);
        // Update will come through WebSocket
      } else if (download.status === 'paused') {
        await apiClient.resumeDownload(id);
        // Update will come through WebSocket
      }
    } catch (error) {
      console.error('Failed to toggle download:', error);
      setError('Failed to pause/resume download');
    }
  };

  const removeDownload = async (id) => {
    try {
      setError(null);
      await apiClient.deleteDownload(id);
      setDownloads(prev => prev.filter(d => d.id !== id));
    } catch (error) {
      console.error('Failed to remove download:', error);
      setError('Failed to remove download');
    }
  };

  const saveSettings = async () => {
    try {
      setError(null);
      await apiClient.updateSettings(globalSettings);
      setShowSettingsModal(false);
      
      // Update new download defaults
      setNewDownload(prev => ({
        ...prev,
        chunks: globalSettings.defaultChunks,
        connectTimeout: globalSettings.connectTimeout,
        readTimeout: globalSettings.readTimeout
      }));
      
    } catch (error) {
      console.error('Failed to save settings:', error);
      setError('Failed to save settings');
    }
  };

  const retryConnection = () => {
    initializeApp();
  };

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatTime = (seconds) => {
    if (!seconds || seconds === Infinity) return '—';
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    
    if (h > 0) return `${h}h ${m}m`;
    if (m > 0) return `${m}m ${s}s`;
    return `${s}s`;
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'downloading': return 'text-blue-600';
      case 'completed': return 'text-green-600';
      case 'paused': return 'text-yellow-600';
      case 'error': return 'text-red-600';
      default: return 'text-gray-600';
    }
  };

  const getStatusBg = (status) => {
    switch (status) {
      case 'downloading': return 'bg-blue-50';
      case 'completed': return 'bg-green-50';
      case 'paused': return 'bg-yellow-50';
      case 'error': return 'bg-red-50';
      default: return 'bg-gray-50';
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'downloading':
        return <Loader className="w-4 h-4 animate-spin" />;
      case 'completed':
        return <CheckCircle className="w-4 h-4" />;
      case 'paused':
        return <Pause className="w-4 h-4" />;
      case 'error':
        return <AlertCircle className="w-4 h-4" />;
      default:
        return null;
    }
  };

  const filteredDownloads = downloads.filter(download => {
    if (activeTab === 'active') return ['downloading', 'paused'].includes(download.status);
    if (activeTab === 'completed') return download.status === 'completed';
    if (activeTab === 'failed') return download.status === 'error';
    return true;
  });

  const stats = {
    active: downloads.filter(d => d.status === 'downloading').length,
    total: downloads.length,
    completed: downloads.filter(d => d.status === 'completed').length,
    totalDownloaded: downloads.reduce((acc, d) => acc + d.downloaded, 0),
    currentSpeed: downloads.filter(d => d.status === 'downloading').reduce((acc, d) => acc + d.speed, 0)
  };

  return (
    <div className="min-h-screen bg-gray-50 flex">
      {/* Sidebar */}
      <div className={`bg-white border-r border-gray-200 transition-all duration-300 ${sidebarCollapsed ? 'w-16' : 'w-64'}`}>
        <div className="p-4">
          <div className="flex items-center justify-between">
            <div className={`flex items-center space-x-3 ${sidebarCollapsed ? 'justify-center' : ''}`}>
              <div className="p-2 bg-blue-600 rounded-lg">
                <Download className="w-6 h-6 text-white" />
              </div>
              {!sidebarCollapsed && (
                <div>
                  <h1 className="text-xl font-bold text-gray-900">Datablip</h1>
                  <p className="text-xs text-gray-500">Download Manager</p>
                </div>
              )}
            </div>
            <button
              onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
              className="p-1 hover:bg-gray-100 rounded-lg lg:hidden"
            >
              <Menu className="w-5 h-5 text-gray-600" />
            </button>
          </div>
        </div>

        <nav className="mt-8">
          <div className={`px-4 ${sidebarCollapsed ? 'px-2' : ''}`}>
            {!sidebarCollapsed && (
              <p className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-3">Navigation</p>
            )}
            <div className="space-y-1">
              {[
                { id: 'active', icon: Activity, label: 'Active Downloads' },
                { id: 'completed', icon: CheckCircle, label: 'Completed' },
                { id: 'failed', icon: AlertCircle, label: 'Failed' },
                { id: 'all', icon: BarChart3, label: 'All Downloads' }
              ].map(item => (
                <button
                  key={item.id}
                  onClick={() => setActiveTab(item.id)}
                  className={`w-full flex items-center ${sidebarCollapsed ? 'justify-center' : ''} px-3 py-2 rounded-lg transition-colors ${
                    activeTab === item.id 
                      ? 'bg-blue-50 text-blue-600' 
                      : 'text-gray-700 hover:bg-gray-50'
                  }`}
                >
                  <item.icon className={`w-5 h-5 ${sidebarCollapsed ? '' : 'mr-3'}`} />
                  {!sidebarCollapsed && <span className="text-sm font-medium">{item.label}</span>}
                </button>
              ))}
            </div>
          </div>

          {!sidebarCollapsed && (
            <div className="mt-8 px-4">
              <p className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-3">Statistics</p>
              <div className="space-y-3">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-gray-600">Active</span>
                  <span className="font-semibold text-gray-900">{stats.active}</span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span className="text-gray-600">Completed</span>
                  <span className="font-semibold text-gray-900">{stats.completed}</span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span className="text-gray-600">Total Size</span>
                  <span className="font-semibold text-gray-900">{formatBytes(stats.totalDownloaded)}</span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span className="text-gray-600">Speed</span>
                  <span className="font-semibold text-gray-900">{formatBytes(stats.currentSpeed)}/s</span>
                </div>
              </div>
            </div>
          )}

          {!sidebarCollapsed && (
            <div className="absolute bottom-4 left-4 right-4">
              <div className="bg-gray-50 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs font-semibold text-gray-600">Connection Status</span>
                  {wsRef.current?.readyState === 1 ? (
                    <Wifi className="w-4 h-4 text-green-600" />
                  ) : (
                    <WifiOff className="w-4 h-4 text-red-600" />
                  )}
                </div>
                <div className="flex items-center space-x-2">
                  <div className={`w-2 h-2 rounded-full ${wsRef.current?.readyState === 1 ? 'bg-green-500' : 'bg-red-500'}`} />
                  <span className="text-xs text-gray-600">
                    {wsRef.current?.readyState === 1 ? 'Connected' : 'Disconnected'}
                  </span>
                </div>
              </div>
            </div>
          )}
        </nav>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <header className="bg-white border-b border-gray-200">
          <div className="px-6 py-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-4">
                <button
                  onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
                  className="p-2 hover:bg-gray-100 rounded-lg hidden lg:block"
                >
                  <Menu className="w-5 h-5 text-gray-600" />
                </button>
                <div>
                  <h2 className="text-2xl font-bold text-gray-900">
                    {activeTab === 'active' && 'Active Downloads'}
                    {activeTab === 'completed' && 'Completed Downloads'}
                    {activeTab === 'failed' && 'Failed Downloads'}
                    {activeTab === 'all' && 'All Downloads'}
                  </h2>
                  <p className="text-sm text-gray-600">Manage and monitor your downloads</p>
                </div>
              </div>
              
              <div className="flex items-center space-x-3">
                <button className="p-2 hover:bg-gray-100 rounded-lg relative">
                  <Bell className="w-5 h-5 text-gray-600" />
                  {stats.active > 0 && (
                    <span className="absolute top-1 right-1 w-2 h-2 bg-blue-600 rounded-full"></span>
                  )}
                </button>
                <button
                  onClick={() => setShowSettingsModal(true)}
                  className="p-2 hover:bg-gray-100 rounded-lg"
                >
                  <Settings className="w-5 h-5 text-gray-600" />
                </button>
                <button
                  onClick={() => setShowAddModal(true)}
                  className="flex items-center space-x-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
                >
                  <Plus className="w-5 h-5" />
                  <span className="font-medium">New Download</span>
                </button>
              </div>
            </div>
          </div>
        </header>

        {/* Content Area */}
        <div className="flex-1 overflow-auto">
          <div className="p-6">
            {/* Quick Stats */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
              <div className="bg-white rounded-lg p-4 border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-600">Active</p>
                    <p className="text-2xl font-bold text-gray-900">{stats.active}</p>
                  </div>
                  <div className="p-3 bg-blue-100 rounded-lg">
                    <Activity className="w-6 h-6 text-blue-600" />
                  </div>
                </div>
              </div>
              <div className="bg-white rounded-lg p-4 border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-600">Completed</p>
                    <p className="text-2xl font-bold text-gray-900">{stats.completed}</p>
                  </div>
                  <div className="p-3 bg-green-100 rounded-lg">
                    <CheckCircle className="w-6 h-6 text-green-600" />
                  </div>
                </div>
              </div>
              <div className="bg-white rounded-lg p-4 border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-600">Total Size</p>
                    <p className="text-2xl font-bold text-gray-900">{formatBytes(stats.totalDownloaded)}</p>
                  </div>
                  <div className="p-3 bg-purple-100 rounded-lg">
                    <HardDrive className="w-6 h-6 text-purple-600" />
                  </div>
                </div>
              </div>
              <div className="bg-white rounded-lg p-4 border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-600">Current Speed</p>
                    <p className="text-2xl font-bold text-gray-900">{formatBytes(stats.currentSpeed)}/s</p>
                  </div>
                  <div className="p-3 bg-orange-100 rounded-lg">
                    <Zap className="w-6 h-6 text-orange-600" />
                  </div>
                </div>
              </div>
            </div>

            {/* Downloads List */}
            <div className="bg-white rounded-lg border border-gray-200">
              {filteredDownloads.length === 0 ? (
                <div className="p-12 text-center">
                  <FileDown className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                  <h3 className="text-lg font-medium text-gray-900 mb-2">No downloads found</h3>
                  <p className="text-sm text-gray-600 mb-4">Start by adding a new download</p>
                  <button
                    onClick={() => setShowAddModal(true)}
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
                  >
                    Add Download
                  </button>
                </div>
              ) : (
                <div className="divide-y divide-gray-200">
                  {filteredDownloads.map(download => (
                    <div 
                      key={download.id} 
                      className="p-6 hover:bg-gray-50 transition-colors cursor-pointer"
                      onClick={() => setSelectedDownload(download)}
                    >
                      <div className="flex items-center justify-between mb-4">
                        <div className="flex items-center space-x-4 flex-1">
                          <div className={`p-2 rounded-lg ${getStatusBg(download.status)}`}>
                            <FileDown className={`w-5 h-5 ${getStatusColor(download.status)}`} />
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center space-x-3">
                              <h3 className="text-sm font-semibold text-gray-900 truncate">{download.filename}</h3>
                              <span className={`inline-flex items-center space-x-1 px-2 py-1 rounded-full text-xs font-medium ${getStatusBg(download.status)} ${getStatusColor(download.status)}`}>
                                {getStatusIcon(download.status)}
                                <span className="capitalize">{download.status}</span>
                              </span>
                            </div>
                            <div className="flex items-center space-x-4 mt-1">
                              <p className="text-xs text-gray-500 truncate flex items-center">
                                <Link2 className="w-3 h-3 mr-1" />
                                {download.url}
                              </p>
                            </div>
                          </div>
                        </div>
                        
                        <div className="flex items-center space-x-2 ml-4">
                          {download.status !== 'completed' && download.status !== 'error' && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                toggleDownload(download.id);
                              }}
                              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                            >
                              {download.status === 'downloading' ? 
                                <Pause className="w-4 h-4 text-gray-600" /> : 
                                <Play className="w-4 h-4 text-gray-600" />
                              }
                            </button>
                          )}
                          {download.status === 'completed' && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                downloadCompletedFile(download.id);
                              }}
                              className="p-2 hover:bg-green-100 rounded-lg transition-colors"
                              title="Download file"
                            >
                              <Download className="w-4 h-4 text-green-600" />
                            </button>
                          )}
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              removeDownload(download.id);
                            }}
                            className="p-2 hover:bg-red-50 rounded-lg transition-colors group"
                          >
                            <Trash2 className="w-4 h-4 text-gray-600 group-hover:text-red-600" />
                          </button>
                        </div>
                      </div>

                      {/* Progress Section */}
                      <div className="space-y-3">
                        <div className="flex items-center justify-between text-xs text-gray-600">
                          <span>{formatBytes(download.downloaded)} of {formatBytes(download.totalSize)}</span>
                          <span>{download.progress.toFixed(1)}%</span>
                        </div>
                        
                        <div className="relative">
                          <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                            <div 
                              className={`h-full transition-all duration-300 ${
                                download.status === 'completed' ? 'bg-green-500' :
                                download.status === 'error' ? 'bg-red-500' :
                                download.status === 'paused' ? 'bg-yellow-500' :
                                'bg-blue-600'
                              }`}
                              style={{ width: `${download.progress}%` }}
                            />
                          </div>
                        </div>

                        {/* Chunk Progress Visualization */}
                        {download.chunkProgress && download.chunkProgress.length > 0 && download.status === 'downloading' && (
                          <div className="space-y-2">
                            <div className="text-xs text-gray-500 font-medium">Chunk Progress:</div>
                            <div className="grid grid-cols-2 gap-2">
                              {download.chunkProgress.map((chunkProgress, index) => (
                                <div key={index} className="space-y-1">
                                  <div className="flex justify-between text-xs text-gray-600">
                                    <span>Chunk {index + 1}</span>
                                    <span>{chunkProgress.toFixed(1)}%</span>
                                  </div>
                                  <div className="h-1 bg-gray-100 rounded-full overflow-hidden">
                                    <div 
                                      className="h-full bg-blue-400 transition-all duration-300"
                                      style={{ width: `${chunkProgress}%` }}
                                    />
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        {/* Stats Row */}
                        <div className="flex items-center justify-between">
                          <div className="flex items-center space-x-6 text-xs text-gray-500">
                            <span className="flex items-center">
                              <Zap className="w-3 h-3 mr-1" />
                              {download.chunks} chunks
                            </span>
                            {download.status === 'downloading' && (
                              <>
                                <span className="flex items-center">
                                  <Activity className="w-3 h-3 mr-1" />
                                  {formatBytes(download.speed)}/s
                                </span>
                                <span className="flex items-center">
                                  <Clock className="w-3 h-3 mr-1" />
                                  {formatTime(download.timeRemaining)} remaining
                                </span>
                              </>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Add Download Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl p-6 w-full max-w-lg">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-xl font-bold text-gray-900">Add New Download</h2>
                <p className="text-sm text-gray-600 mt-1">Enter the URL and configure download settings</p>
              </div>
              <button
                onClick={() => setShowAddModal(false)}
                className="p-2 hover:bg-gray-100 rounded-lg"
              >
                <X className="w-5 h-5 text-gray-400" />
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Download URL <span className="text-red-500">*</span>
                </label>
                <div className="relative">
                  <Globe className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <input
                    type="url"
                    value={newDownload.url}
                    onChange={(e) => setNewDownload({...newDownload, url: e.target.value})}
                    className="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    placeholder="https://example.com/file.zip"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Save As
                </label>
                <div className="relative">
                  <FolderOpen className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <input
                    type="text"
                    value={newDownload.output}
                    onChange={(e) => setNewDownload({...newDownload, output: e.target.value})}
                    className="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    placeholder="custom-filename.zip (optional)"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Number of Chunks
                </label>
                <select
                  value={newDownload.chunks}
                  onChange={(e) => setNewDownload({...newDownload, chunks: parseInt(e.target.value)})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                >
                  {[1, 2, 4, 8, 16].map(num => (
                    <option key={num} value={num}>{num} {num === 1 ? 'chunk' : 'chunks'}</option>
                  ))}
                </select>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Connect Timeout
                  </label>
                  <input
                    type="text"
                    value={newDownload.connectTimeout}
                    onChange={(e) => setNewDownload({...newDownload, connectTimeout: e.target.value})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    placeholder="30s"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Read Timeout
                  </label>
                  <input
                    type="text"
                    value={newDownload.readTimeout}
                    onChange={(e) => setNewDownload({...newDownload, readTimeout: e.target.value})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    placeholder="10m"
                  />
                </div>
              </div>
            </div>

            <div className="flex items-center justify-end space-x-3 mt-6 pt-6 border-t border-gray-200">
              <button
                onClick={() => setShowAddModal(false)}
                className="px-4 py-2 text-gray-700 hover:text-gray-900 font-medium transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={addDownload}
                disabled={!newDownload.url}
                className="px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors"
              >
                Start Download
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Settings Modal */}
      {showSettingsModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl p-6 w-full max-w-lg">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-xl font-bold text-gray-900">Settings</h2>
                <p className="text-sm text-gray-600 mt-1">Configure default download preferences</p>
              </div>
              <button
                onClick={() => setShowSettingsModal(false)}
                className="p-2 hover:bg-gray-100 rounded-lg"
              >
                <X className="w-5 h-5 text-gray-400" />
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Default Number of Chunks
                </label>
                <select
                  value={globalSettings.defaultChunks}
                  onChange={(e) => setGlobalSettings({...globalSettings, defaultChunks: parseInt(e.target.value)})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {[1, 2, 4, 8, 16].map(num => (
                    <option key={num} value={num}>{num} {num === 1 ? 'chunk' : 'chunks'}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Maximum Concurrent Downloads
                </label>
                <input
                  type="number"
                  min="1"
                  max="10"
                  value={globalSettings.maxConcurrentDownloads}
                  onChange={(e) => setGlobalSettings({...globalSettings, maxConcurrentDownloads: parseInt(e.target.value) || 3})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Default Connect Timeout
                  </label>
                  <input
                    type="text"
                    value={globalSettings.connectTimeout}
                    onChange={(e) => setGlobalSettings({...globalSettings, connectTimeout: e.target.value})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Default Read Timeout
                  </label>
                  <input
                    type="text"
                    value={globalSettings.readTimeout}
                    onChange={(e) => setGlobalSettings({...globalSettings, readTimeout: e.target.value})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </div>

              <div className="pt-4 border-t border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <label className="text-sm font-medium text-gray-700">Auto-start Downloads</label>
                    <p className="text-xs text-gray-500 mt-1">Automatically begin downloading when added</p>
                  </div>
                  <button
                    onClick={() => setGlobalSettings({...globalSettings, autoStart: !globalSettings.autoStart})}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                      globalSettings.autoStart ? 'bg-blue-600' : 'bg-gray-200'
                    }`}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        globalSettings.autoStart ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </div>
              </div>
            </div>

            <div className="flex items-center justify-end space-x-3 mt-6 pt-6 border-t border-gray-200">
              <button
                onClick={() => setShowSettingsModal(false)}
                className="px-4 py-2 text-gray-700 hover:text-gray-900 font-medium transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => setShowSettingsModal(false)}
                className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors"
              >
                Save Changes
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Download Details Sidebar */}
      {selectedDownload && (
        <div className="fixed right-0 top-0 h-full w-96 bg-white shadow-xl z-40">
          <div className="p-6 border-b border-gray-200">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold text-gray-900">Download Details</h3>
              <button
                onClick={() => setSelectedDownload(null)}
                className="p-2 hover:bg-gray-100 rounded-lg"
              >
                <X className="w-5 h-5 text-gray-400" />
              </button>
            </div>
          </div>

          <div className="p-6 space-y-6">
            <div>
              <h4 className="text-sm font-medium text-gray-700 mb-2">File Information</h4>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Filename</span>
                  <span className="text-gray-900 font-medium">{selectedDownload.filename}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Size</span>
                  <span className="text-gray-900 font-medium">{formatBytes(selectedDownload.totalSize)}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Status</span>
                  <span className={`font-medium capitalize ${getStatusColor(selectedDownload.status)}`}>
                    {selectedDownload.status}
                  </span>
                </div>
              </div>
            </div>

            <div>
              <h4 className="text-sm font-medium text-gray-700 mb-2">Download Progress</h4>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Progress</span>
                  <span className="text-gray-900 font-medium">{selectedDownload.progress.toFixed(1)}%</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Downloaded</span>
                  <span className="text-gray-900 font-medium">{formatBytes(selectedDownload.downloaded)}</span>
                </div>
                {selectedDownload.status === 'downloading' && (
                  <>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-500">Speed</span>
                      <span className="text-gray-900 font-medium">{formatBytes(selectedDownload.speed)}/s</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-500">Time Remaining</span>
                      <span className="text-gray-900 font-medium">{formatTime(selectedDownload.timeRemaining)}</span>
                    </div>
                  </>
                )}
              </div>
            </div>

            {/* Chunk Progress Section in Modal */}
            {selectedDownload.chunkProgress && selectedDownload.chunkProgress.length > 0 && selectedDownload.status === 'downloading' && (
              <div>
                <h4 className="text-sm font-medium text-gray-700 mb-2">Chunk Progress</h4>
                <div className="space-y-3">
                  {selectedDownload.chunkProgress.map((chunkProgress, index) => (
                    <div key={index} className="space-y-1">
                      <div className="flex justify-between text-sm">
                        <span className="text-gray-500">Chunk {index + 1}</span>
                        <span className="text-gray-900 font-medium">{chunkProgress.toFixed(1)}%</span>
                      </div>
                      <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
                        <div 
                          className="h-full bg-blue-500 transition-all duration-300"
                          style={{ width: `${chunkProgress}%` }}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div>
              <h4 className="text-sm font-medium text-gray-700 mb-2">Configuration</h4>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Chunks</span>
                  <span className="text-gray-900 font-medium">{selectedDownload.chunks}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Connect Timeout</span>
                  <span className="text-gray-900 font-medium">{selectedDownload.connectTimeout || '30s'}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Read Timeout</span>
                  <span className="text-gray-900 font-medium">{selectedDownload.readTimeout || '10m'}</span>
                </div>
              </div>
            </div>

            <div>
              <h4 className="text-sm font-medium text-gray-700 mb-2">Source URL</h4>
              <div className="p-3 bg-gray-50 rounded-lg">
                <p className="text-xs text-gray-600 break-all">{selectedDownload.url}</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default DatablipUI;