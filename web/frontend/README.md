# Datablip Frontend - Web-Based Chunked Download Manager

A modern, responsive React web application for managing high-speed chunked file downloads with real-time progress tracking and WebSocket-powered live updates.

## 🚀 Features

### Core Functionality
- **Chunked Downloads**: Split large files into multiple chunks for parallel downloading
- **Real-Time Progress**: Live progress updates for overall download and individual chunks
- **Speed Monitoring**: Real-time download speed calculation and time remaining estimation
- **WebSocket Integration**: Instant progress updates without polling
- **Automatic File Saving**: Downloads automatically trigger browser save dialog on completion
- **Manual Download**: Click-to-download button for completed files
- **Responsive Design**: Works seamlessly on desktop and mobile devices

### User Interface
- **Modern React UI**: Clean, intuitive interface built with React 18
- **Progress Visualization**: 
  - Overall progress bar for each download
  - Individual chunk progress bars during active downloads
  - Color-coded status indicators (downloading, completed, paused, error)
- **Real-Time Statistics**: 
  - Total downloads count
  - Active downloads
  - Completed downloads
  - Current combined download speed
- **Download Management**:
  - Add new downloads with custom settings
  - Pause/resume downloads
  - Delete downloads
  - View detailed download information

### Advanced Features
- **Configurable Settings**:
  - Number of chunks (1-16)
  - Connection timeout
  - Read timeout
  - Auto-start downloads
  - Maximum concurrent downloads
- **Download Details Modal**: Comprehensive view of download progress, speed, and configuration
- **Error Handling**: Graceful error display and recovery
- **Connection Status**: Visual indicator for WebSocket connection status

## 🛠 Technology Stack

### Frontend
- **React 18.2.0**: Modern React with hooks and functional components
- **Lucide React**: Beautiful, customizable icons
- **CSS3**: Custom responsive styling with flexbox and grid
- **WebSocket API**: Real-time communication with backend

### Build Tools
- **React Scripts 5.0.1**: Create React App toolchain
- **npm**: Package management
- **Webpack**: Module bundling (via React Scripts)
- **Babel**: JavaScript transpilation

## 📦 Installation

### Prerequisites
- Node.js 16+ and npm
- Go 1.23+ (for backend)
- Modern web browser with WebSocket support

### Setup Instructions

1. **Clone the repository**:
   ```bash
   git clone https://github.com/ahdhi/Datablip.git
   cd Datablip/web/frontend
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Start the development server**:
   ```bash
   npm start
   ```

4. **Build for production**:
   ```bash
   npm run build
   ```

The application will be available at `http://localhost:3000`.

## 🔧 Configuration

### Environment Variables
Create a `.env` file in the frontend directory to customize API endpoints:

```env
REACT_APP_API_URL=http://localhost:8080/api
REACT_APP_WS_URL=ws://localhost:8080/ws
```

### Default Settings
- **API Base URL**: `http://localhost:8080/api`
- **WebSocket URL**: `ws://localhost:8080/ws`
- **Default Chunks**: 4
- **Connection Timeout**: 30 seconds
- **Read Timeout**: 10 minutes

## 🎮 Usage Guide

### Adding a Download
1. Click the **"+ Add Download"** button
2. Enter the file URL
3. Specify the output filename
4. Configure download settings:
   - Number of chunks (more chunks = faster download for large files)
   - Connection and read timeouts
5. Click **"Start Download"**

### Monitoring Progress
- **Overall Progress**: Main progress bar shows total completion percentage
- **Chunk Progress**: Individual bars show each chunk's progress during download
- **Speed & ETA**: Real-time speed and estimated time remaining
- **Statistics Panel**: Overview of all download statistics

### Managing Downloads
- **Pause/Resume**: Click the pause/play button next to active downloads
- **Download File**: Click the download icon (📥) for completed files
- **Delete**: Click the trash icon to remove downloads
- **View Details**: Click on any download to see detailed information

### Settings Configuration
1. Click the settings icon (⚙️)
2. Adjust global defaults:
   - Default number of chunks
   - Connection timeouts
   - Auto-start behavior
   - Maximum concurrent downloads
3. Settings apply to new downloads

## 🔌 API Integration

### REST API Endpoints
- `GET /api/downloads` - List all downloads
- `POST /api/downloads` - Create new download
- `GET /api/downloads/{id}` - Get download details
- `POST /api/downloads/{id}/pause` - Pause download
- `POST /api/downloads/{id}/resume` - Resume download
- `GET /api/downloads/{id}/file` - Download completed file
- `DELETE /api/downloads/{id}` - Delete download
- `GET /api/settings` - Get global settings
- `PUT /api/settings` - Update global settings

### WebSocket Events
- `progress` - Real-time progress updates with chunk details
- `status` - Download status changes
- `completed` - Download completion notification
- `error` - Error notifications

### Data Structures

#### Download Object
```javascript
{
  id: "string",
  url: "string",
  filename: "string",
  status: "pending|downloading|paused|completed|error",
  progress: 85.5,           // Overall progress percentage
  totalSize: 1000000000,    // Total file size in bytes
  downloaded: 855000000,    // Downloaded bytes
  speed: 2500000,           // Current speed in bytes/second
  chunks: 4,                // Number of chunks
  chunkProgress: [100, 100, 85.5, 0], // Individual chunk progress
  timeRemaining: 58,        // Estimated seconds remaining
  startTime: "2025-09-27T04:18:06Z",
  connectTimeout: "30s",
  readTimeout: "10m"
}
```

## 🎨 UI Components

### Main Components
- **DatablipUI**: Root component managing application state
- **AddDownloadModal**: Form for creating new downloads
- **SettingsModal**: Global settings configuration
- **DownloadDetailsModal**: Detailed view of individual downloads

### Key Features
- **Responsive Layout**: Adapts to different screen sizes
- **Progress Bars**: Animated progress indicators
- **Status Icons**: Visual download status representation
- **Real-time Updates**: Live data refresh via WebSocket
- **Error States**: User-friendly error messages

## 🔍 Troubleshooting

### Common Issues

**Connection Errors**
- Ensure backend server is running on port 8080
- Check WebSocket connection in browser developer tools
- Verify CORS settings allow requests from frontend port

**Download Issues**
- Confirm the URL supports HTTP Range requests
- Check network connectivity
- Verify sufficient disk space

**Performance**
- Reduce number of chunks for smaller files
- Increase chunks for larger files and better connections
- Monitor browser memory usage for many concurrent downloads

### Debug Mode
Open browser developer tools to see:
- WebSocket connection status
- API request/response details
- Real-time progress update logs
- Error messages and stack traces

## 📊 Performance Optimization

### Best Practices
- **Chunk Configuration**: Use 4-8 chunks for files over 100MB
- **Concurrent Limits**: Limit concurrent downloads to avoid overwhelming bandwidth
- **Progress Updates**: Updates occur 4 times per second for smooth UI
- **Memory Management**: Completed downloads are cached for quick access

### Browser Compatibility
- **Chrome 90+**: Full support
- **Firefox 88+**: Full support
- **Safari 14+**: Full support
- **Edge 90+**: Full support

## 🔒 Security Considerations

- **CORS Policy**: Configured for development (allow all origins)
- **File Downloads**: Browser-native download security
- **WebSocket Security**: No authentication required for demo
- **Input Validation**: URL and filename validation

## 🚧 Development

### Project Structure
```
web/frontend/
├── public/
│   ├── index.html          # HTML template
│   └── manifest.json       # PWA manifest
├── src/
│   ├── api/
│   │   └── client.js       # API client and WebSocket
│   ├── App.js              # Main application component
│   ├── App.css             # Application styles
│   └── index.js            # React app entry point
├── package.json            # Dependencies and scripts
└── README.md              # This file
```

### Available Scripts
- `npm start` - Start development server
- `npm test` - Run test suite
- `npm run build` - Create production build
- `npm run eject` - Eject from Create React App

### Development Features
- **Hot Reload**: Automatic refresh on code changes
- **Error Overlay**: In-browser error display
- **Source Maps**: Debug with original source code
- **Linting**: ESLint integration for code quality

## 📈 Future Enhancements

### Planned Features
- **Authentication**: User accounts and secure downloads
- **Download History**: Persistent download records
- **Scheduling**: Queue downloads for specific times
- **Bandwidth Limiting**: Control download speed
- **Download Categories**: Organize downloads by type
- **Dark Mode**: Alternative UI theme
- **Mobile App**: React Native version

### Contributing
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🤝 Support

For support and questions:
- Create an issue on GitHub
- Check the troubleshooting guide above
- Review browser developer tools for error details

---

**Datablip Frontend** - Built with ❤️ using React and modern web technologies.
