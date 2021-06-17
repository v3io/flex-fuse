package journal

import (
	"context"
	"fmt"
	"github.com/coreos/go-systemd/journal"
	"github.com/nuclio/logger"
)

var j = Logger{}

func Error(message interface{}, vars ...interface{}) {
	j.Error(message, vars...)
}

func Warn(message interface{}, vars ...interface{}) {
	j.Warn(message, vars...)
}

func Info(message interface{}, vars ...interface{}) {
	j.Info(message, vars...)
}

func Debug(message interface{}, vars ...interface{}) {
	j.Debug(message, vars...)
}

type Logger struct {
}

func (j *Logger) journal(priority journal.Priority, message interface{}, vars ...interface{}) {
	format := ""
	if len(vars) > 0 {
		format = fmt.Sprintf("%s: %s", message, vars)
	} else {
		format = fmt.Sprint(message)
	}
	journal.Send(format, priority, nil) // nolint: errcheck
}

func (j *Logger) Error(message interface{}, vars ...interface{}) {
	j.journal(journal.PriErr, message, vars...)
}

func (j *Logger) Warn(message interface{}, vars ...interface{}) {
	j.journal(journal.PriWarning, message, vars...)
}

func (j *Logger) Info(message interface{}, vars ...interface{}) {
	j.journal(journal.PriInfo, message, vars...)
}

func (j *Logger) Debug(message interface{}, vars ...interface{}) {
	j.journal(journal.PriDebug, message, vars...)
}

func (j *Logger) ErrorWith(message interface{}, vars ...interface{}) {
	j.journal(journal.PriErr, message, vars...)
}

func (j *Logger) WarnWith(message interface{}, vars ...interface{}) {
	j.journal(journal.PriWarning, message, vars...)
}

func (j *Logger) InfoWith(message interface{}, vars ...interface{}) {
	j.journal(journal.PriInfo, message, vars...)
}

func (j *Logger) DebugWith(message interface{}, vars ...interface{}) {
	j.journal(journal.PriDebug, message, vars...)
}

// ErrorCtx emits an unstructured error log with context
func (j *Logger) ErrorCtx(ctx context.Context, format interface{}, vars ...interface{}) {

}

// WarnCtx emits an unstructured warning log with context
func (j *Logger) WarnCtx(ctx context.Context, format interface{}, vars ...interface{}) {

}

// InfoCtx emits an unstructured informational log with context
func (j *Logger) InfoCtx(ctx context.Context, format interface{}, vars ...interface{}) {

}

// DebugCtx emits an unstructured debug log with context
func (j *Logger) DebugCtx(ctx context.Context, format interface{}, vars ...interface{}) {

}

// ErrorWithCtx emits a structured error log with context
func (j *Logger) ErrorWithCtx(ctx context.Context, format interface{}, vars ...interface{}) {

}

// WarnWithCtx emits a structured warning log with context
func (j *Logger) WarnWithCtx(ctx context.Context, format interface{}, vars ...interface{}) {

}

// InfoWithCtx emits a structured info log with context
func (j *Logger) InfoWithCtx(ctx context.Context, format interface{}, vars ...interface{}) {

}

// DebugWithCtx emits a structured debug log with context
func (j *Logger) DebugWithCtx(ctx context.Context, format interface{}, vars ...interface{}) {

}

func (j *Logger) Flush() {
}

func (j *Logger) GetChild(name string) logger.Logger {
	return j
}
