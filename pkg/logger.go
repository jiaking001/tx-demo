package pkg

import "go.uber.org/zap"

func NewLogger() (*zap.Logger, error) {
	return zap.NewDevelopment()
}
