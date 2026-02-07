package plugins

import (
	"fmt"
	"plugin"
	"sync"

	"github.com/Global-Wizards/wizards-qa/pkg/maestro"
)

// Plugin represents a wizards-qa plugin
type Plugin interface {
	// Name returns the plugin name
	Name() string
	
	// Version returns the plugin version
	Version() string
	
	// Initialize sets up the plugin with configuration
	Initialize(config map[string]interface{}) error
	
	// Hooks returns the hooks this plugin implements
	Hooks() []string
}

// TestHook allows plugins to hook into test execution
type TestHook interface {
	Plugin
	
	// BeforeTest runs before test execution
	BeforeTest(context *TestContext) error
	
	// AfterTest runs after test execution
	AfterTest(context *TestContext, results *maestro.TestResults) error
}

// AnalysisHook allows plugins to hook into AI analysis
type AnalysisHook interface {
	Plugin
	
	// BeforeAnalysis runs before AI analysis
	BeforeAnalysis(gameURL, specPath string) error
	
	// AfterAnalysis runs after AI analysis
	AfterAnalysis(analysis interface{}) error
}

// ReportHook allows plugins to hook into report generation
type ReportHook interface {
	Plugin
	
	// BeforeReport runs before report generation
	BeforeReport(results *maestro.TestResults) error
	
	// AfterReport runs after report generation
	AfterReport(reportPath string) error
}

// TestContext provides context for test hooks
type TestContext struct {
	GameURL   string
	SpecPath  string
	FlowsDir  string
	Config    map[string]interface{}
}

// Manager manages loaded plugins
type Manager struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
	}
}

// Load loads a plugin from a shared library file
func (m *Manager) Load(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for NewPlugin symbol
	symPlugin, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("plugin missing NewPlugin function: %w", err)
	}

	// Cast to plugin constructor
	newPlugin, ok := symPlugin.(func() Plugin)
	if !ok {
		return fmt.Errorf("NewPlugin has wrong signature")
	}

	// Create plugin instance
	plug := newPlugin()
	
	// Register plugin
	m.plugins[plug.Name()] = plug
	
	return nil
}

// Register manually registers a plugin (useful for built-in plugins)
func (m *Manager) Register(p Plugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[p.Name()] = p
}

// Get retrieves a plugin by name
func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

// List returns all registered plugins
func (m *Manager) List() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	list := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		list = append(list, p)
	}
	return list
}

// ExecuteTestHooks executes all test hooks
func (m *Manager) ExecuteTestHooks(before bool, context *TestContext, results *maestro.TestResults) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, p := range m.plugins {
		hook, ok := p.(TestHook)
		if !ok {
			continue
		}
		
		var err error
		if before {
			err = hook.BeforeTest(context)
		} else {
			err = hook.AfterTest(context, results)
		}
		
		if err != nil {
			return fmt.Errorf("plugin %s hook failed: %w", p.Name(), err)
		}
	}
	
	return nil
}

// ExecuteAnalysisHooks executes all analysis hooks
func (m *Manager) ExecuteAnalysisHooks(before bool, gameURL, specPath string, analysis interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, p := range m.plugins {
		hook, ok := p.(AnalysisHook)
		if !ok {
			continue
		}
		
		var err error
		if before {
			err = hook.BeforeAnalysis(gameURL, specPath)
		} else {
			err = hook.AfterAnalysis(analysis)
		}
		
		if err != nil {
			return fmt.Errorf("plugin %s hook failed: %w", p.Name(), err)
		}
	}
	
	return nil
}

// ExecuteReportHooks executes all report hooks
func (m *Manager) ExecuteReportHooks(before bool, results *maestro.TestResults, reportPath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, p := range m.plugins {
		hook, ok := p.(ReportHook)
		if !ok {
			continue
		}
		
		var err error
		if before {
			err = hook.BeforeReport(results)
		} else {
			err = hook.AfterReport(reportPath)
		}
		
		if err != nil {
			return fmt.Errorf("plugin %s hook failed: %w", p.Name(), err)
		}
	}
	
	return nil
}

// Example built-in plugin - Slack notifications
type SlackNotifierPlugin struct {
	webhookURL string
}

func NewSlackNotifierPlugin() Plugin {
	return &SlackNotifierPlugin{}
}

func (p *SlackNotifierPlugin) Name() string {
	return "slack-notifier"
}

func (p *SlackNotifierPlugin) Version() string {
	return "1.0.0"
}

func (p *SlackNotifierPlugin) Initialize(config map[string]interface{}) error {
	url, ok := config["webhookURL"].(string)
	if !ok {
		return fmt.Errorf("webhookURL not provided")
	}
	p.webhookURL = url
	return nil
}

func (p *SlackNotifierPlugin) Hooks() []string {
	return []string{"AfterTest"}
}

func (p *SlackNotifierPlugin) BeforeTest(context *TestContext) error {
	return nil
}

func (p *SlackNotifierPlugin) AfterTest(context *TestContext, results *maestro.TestResults) error {
	// TODO: Send Slack notification with test results
	fmt.Printf("Would send Slack notification to: %s\n", p.webhookURL)
	return nil
}
