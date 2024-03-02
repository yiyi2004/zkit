package startup

import (
	"zkit/logger"
)

func InitLog() logger.LoggerV1 {
	return logger.NewNoOpLogger()
}
