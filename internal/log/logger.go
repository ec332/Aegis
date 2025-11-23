package log

import (
    "go.uber.org/zap"
)

func New() *zap.Logger {
    cfg := zap.NewProductionConfig()
    cfg.Encoding = "json"
    cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
    logger, _ := cfg.Build()
    return logger
}