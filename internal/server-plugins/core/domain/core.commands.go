package domain

// CoreCommand represents allowed Dokku commands for the core plugin
type CoreCommand string

const (
	// System commands
	CommandVersion CoreCommand = "version"
	CommandEvents  CoreCommand = "events"

	// Proxy commands
	CommandProxyReport CoreCommand = "proxy:report"
	CommandProxySet    CoreCommand = "proxy:set"

	// Scheduler commands
	CommandSchedulerReport CoreCommand = "scheduler:report"
	CommandSchedulerSet    CoreCommand = "scheduler:set"

	// Git commands
	CommandGitReport CoreCommand = "git:report"
	CommandGitSet    CoreCommand = "git:set"

	// Plugin management commands
	CommandPluginList      CoreCommand = "plugin:list"
	CommandPluginInstall   CoreCommand = "plugin:install"
	CommandPluginUninstall CoreCommand = "plugin:uninstall"
	CommandPluginEnable    CoreCommand = "plugin:enable"
	CommandPluginDisable   CoreCommand = "plugin:disable"
	CommandPluginUpdate    CoreCommand = "plugin:update"

	// SSH key commands
	CommandSSHKeysList   CoreCommand = "ssh-keys:list"
	CommandSSHKeysRemove CoreCommand = "ssh-keys:remove"

	// Registry commands
	CommandRegistryLogout CoreCommand = "registry:logout"

	// Logs commands
	CommandLogsSet CoreCommand = "logs:set"
)

// IsValid checks if the command is a valid core command
func (c CoreCommand) IsValid() bool {
	switch c {
	case CommandVersion, CommandEvents,
		CommandProxyReport, CommandProxySet,
		CommandSchedulerReport, CommandSchedulerSet,
		CommandGitReport, CommandGitSet,
		CommandPluginList, CommandPluginInstall, CommandPluginUninstall,
		CommandPluginEnable, CommandPluginDisable, CommandPluginUpdate,
		CommandSSHKeysList, CommandSSHKeysRemove,
		CommandRegistryLogout,
		CommandLogsSet:
		return true
	default:
		return false
	}
}

// String returns the string representation of the command
func (c CoreCommand) String() string {
	return string(c)
}

// GetAllowedCommands returns all allowed core commands
func GetAllowedCoreCommands() []CoreCommand {
	return []CoreCommand{
		CommandVersion,
		CommandEvents,
		CommandProxyReport,
		CommandProxySet,
		CommandSchedulerReport,
		CommandSchedulerSet,
		CommandGitReport,
		CommandGitSet,
		CommandPluginList,
		CommandPluginInstall,
		CommandPluginUninstall,
		CommandPluginEnable,
		CommandPluginDisable,
		CommandPluginUpdate,
		CommandSSHKeysList,
		CommandSSHKeysRemove,
		CommandRegistryLogout,
		CommandLogsSet,
	}
}
