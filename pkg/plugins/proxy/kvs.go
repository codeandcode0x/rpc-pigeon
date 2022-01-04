package proxy

// k8s vs server
func (plugin *Plugin) KvsServer() {
	defer func() {
		plugin.Status <- false
	}()
}
