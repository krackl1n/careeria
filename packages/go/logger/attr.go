package logger

import "log/slog"

func Error(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}

	return slog.Any("error", err)
}
