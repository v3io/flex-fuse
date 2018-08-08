package journal

import (
	"fmt"
	"github.com/coreos/go-systemd/journal"
	"github.com/nuclio/logger"
)

var j = JournalLogger{}

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

type JournalLogger struct {
}

func (j *JournalLogger) journal(priority journal.Priority, message interface{}, vars ...interface{}) {
	format := ""
	if len(vars) > 0 {
		format = fmt.Sprintf("%s: %s", message, vars)
	} else {
		format = fmt.Sprint(message)
	}
	journal.Send(format, priority, nil)
}

func (j *JournalLogger) Error(message interface{}, vars ...interface{}) {
	j.journal(journal.PriErr, message, vars...)
}

func (j *JournalLogger) Warn(message interface{}, vars ...interface{}) {
	j.journal(journal.PriWarning, message, vars...)
}

func (j *JournalLogger) Info(message interface{}, vars ...interface{}) {
	j.journal(journal.PriInfo, message, vars...)
}

func (j *JournalLogger) Debug(message interface{}, vars ...interface{}) {
	j.journal(journal.PriDebug, message, vars...)
}

func (j *JournalLogger) ErrorWith(message interface{}, vars ...interface{}) {
	j.journal(journal.PriErr, message, vars...)
}

func (j *JournalLogger) WarnWith(message interface{}, vars ...interface{}) {
	j.journal(journal.PriWarning, message, vars...)
}

func (j *JournalLogger) InfoWith(message interface{}, vars ...interface{}) {
	j.journal(journal.PriInfo, message, vars...)
}

func (j *JournalLogger) DebugWith(message interface{}, vars ...interface{}) {
	j.journal(journal.PriDebug, message, vars...)
}

func (j *JournalLogger) Flush() {
}

func (j *JournalLogger) GetChild(name string) logger.Logger {
	return j
}
