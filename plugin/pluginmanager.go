package plugin

// PluginManager instance
var PluginManager Manager

// Manager contains the loaded plugins
type Manager struct {
	PluginDirs []string
	Plugins    map[string]Plugin
}

// LoadPLugins will load external plugins
func (m *Manager) LoadPLugins() error {
	var error error
	// Recurse Parse folder for executable files
	// launch it with --help
	// Parse the result and create a new entry for plugins to call
	return error
}

// Register a new plugin
func (m *Manager) Register(plugin Plugin) {
	if m.Plugins == nil {
		m.Plugins = make(map[string]Plugin)
	}
	m.Plugins[plugin.GetMetadata().Name] = plugin
}
