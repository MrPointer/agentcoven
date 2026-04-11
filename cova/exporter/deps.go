package exporter

import (
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

// Deps holds injected dependencies for exporter management operations.
type Deps struct {
	Logger      logger.Logger
	FileSystem  utils.FileSystem
	Locker      utils.Locker
	Dispatcher  Dispatcher
	EnvManager  osmanager.EnvironmentManager
	UserManager osmanager.UserManager
}
