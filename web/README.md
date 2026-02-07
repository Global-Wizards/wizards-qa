# Wizards QA Web Dashboard

Modern web dashboard for visualizing test results, managing flows, and monitoring QA automation.

## Stack

**Backend:**
- **Go** - High-performance backend
- **Chi Router** - Lightweight HTTP router
- **CORS** - Cross-origin resource sharing

**Frontend:**
- **Vue.js 3** - Progressive JavaScript framework
- **Vite** - Fast build tool
- **Tailwind CSS** - Utility-first CSS framework
- **Axios** - HTTP client
- **Vue Router** - Official router

## Features

### Dashboard
- 📊 Real-time test statistics
- 📈 Success rate trends
- 🕐 Recent test history
- 💡 At-a-glance metrics

### Test History
- 📝 Complete test execution history
- 🔍 Filter and search tests
- ⏱️ Duration and performance metrics
- ✅ Pass/fail status indicators

### Reports
- 📄 Browse generated test reports
- 🔗 Quick access to report files
- 📅 Organized by date
- 💾 Download reports

### Flow Templates
- 📁 Browse available templates
- 👁️ Preview template content
- 📋 Copy and customize
- 🎨 Organized by category

## Getting Started

### Prerequisites

- **Go 1.21+**
- **Node.js 18+**
- **npm or pnpm**

### Installation

#### Backend

```bash
cd web/backend
go mod download
go build -o dashboard-server
./dashboard-server
```

Server runs on `http://localhost:8080`

#### Frontend

```bash
cd web/frontend
npm install
npm run dev
```

Dashboard available at `http://localhost:3000`

### Production Build

```bash
# Build frontend
cd web/frontend
npm run build

# Build backend
cd ../backend
go build -o dashboard-server

# Run backend (serves static files)
./dashboard-server
```

## API Endpoints

### Health
```
GET /api/health
```

### Tests
```
GET /api/tests          # List all tests
GET /api/tests/{id}     # Get specific test
POST /api/tests/run     # Execute new test
```

### Reports
```
GET /api/reports        # List all reports
GET /api/reports/{id}   # Get specific report
```

### Flows
```
GET /api/flows          # List all flow templates
GET /api/flows/{name}   # Get specific flow
```

### Statistics
```
GET /api/stats          # Get dashboard statistics
```

## Configuration

### Backend Environment Variables

```bash
PORT=8080                           # Server port
WIZARDS_QA_DATA_DIR=./data          # Data directory
WIZARDS_QA_REPORTS_DIR=./reports    # Reports directory
WIZARDS_QA_FLOWS_DIR=./flows        # Flows directory
```

### Frontend Configuration

Edit `vite.config.js` to change API proxy settings:

```js
export default defineConfig({
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
})
```

## Development

### Backend

```bash
cd web/backend

# Run with hot reload (requires air)
air

# Or run normally
go run .
```

### Frontend

```bash
cd web/frontend

# Development server with HMR
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

## Project Structure

```
web/
├── backend/
│   ├── server.go           # Main server
│   ├── go.mod              # Go dependencies
│   └── go.sum
│
├── frontend/
│   ├── src/
│   │   ├── views/          # Page components
│   │   │   ├── Dashboard.vue
│   │   │   ├── Tests.vue
│   │   │   ├── Reports.vue
│   │   │   └── Flows.vue
│   │   ├── App.vue         # Root component
│   │   ├── main.js         # Entry point
│   │   └── style.css       # Global styles
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   └── tailwind.config.js
│
└── README.md
```

## Features Roadmap

- [ ] **Real-time Updates** - WebSocket for live test updates
- [ ] **Test Execution** - Run tests directly from dashboard
- [ ] **Flow Editor** - Visual flow editor with drag-and-drop
- [ ] **Analytics** - Advanced metrics and trends
- [ ] **User Management** - Multi-user support with roles
- [ ] **Integrations** - Slack, GitHub, Jira notifications
- [ ] **Custom Themes** - Dark mode and custom themes
- [ ] **Export** - Export data as CSV, PDF
- [ ] **Scheduling** - Schedule automated test runs
- [ ] **AI Insights** - AI-powered test analysis

## Deployment

### Docker

```bash
# Build Docker image
docker build -t wizards-qa-dashboard .

# Run container
docker run -p 8080:8080 wizards-qa-dashboard
```

### Production

For production deployment, consider:

1. **Reverse Proxy** - Use Nginx or Caddy
2. **HTTPS** - Enable SSL/TLS
3. **Process Manager** - Use systemd or PM2
4. **Monitoring** - Add health checks
5. **Logging** - Centralized logging
6. **Backups** - Regular data backups

## Troubleshooting

### Port Already in Use

```bash
# Change backend port
PORT=3001 ./dashboard-server

# Change frontend port
npm run dev -- --port 3001
```

### API Connection Issues

Check that:
1. Backend is running on correct port
2. Frontend proxy is configured correctly
3. No CORS issues (check browser console)

### Build Errors

```bash
# Clear node_modules and reinstall
rm -rf node_modules
npm install

# Clear Go module cache
go clean -modcache
go mod download
```

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for development guidelines.

## License

MIT License - see [LICENSE](../LICENSE) for details.

---

**Built with** 🧙‍♂️ **by Wizards QA** | Powered by Vue.js + Go
