package monitor

import (
	"strings"
	"time"

	"github.com/demdxx/asyncp/v2/libs/errors"
)

// ApplicationInfo with basic description
type ApplicationInfo struct {
	Name     string               `json:"name"`
	Host     string               `json:"host"`
	Hostname string               `json:"hostname"`
	InitedAt time.Time            `json:"inited_at"`
	Tasks    map[string][]string  `json:"tasks"`
	Servers  map[string]time.Time `json:"servers,omitempty"`
}

// Merge application info
func (app *ApplicationInfo) Merge(info *ApplicationInfo) {
	app.Name = info.Name
	if app.Servers == nil {
		app.Servers = map[string]time.Time{}
	}
	app.Servers[info.Host] = info.InitedAt
	if info.Tasks != nil {
		if app.Tasks == nil {
			app.Tasks = make(map[string][]string, len(info.Tasks))
		}
		for taskName, taskTarget := range info.Tasks {
			app.Tasks[taskName] = append([]string{}, taskTarget...)
		}
	}
}

// TaskInfo aggregated in one record
type TaskInfo struct {
	TotalCount   uint64        `json:"total_count"`
	ErrorCount   uint64        `json:"error_count"`
	SuccessCount uint64        `json:"success_count"`
	SkipCount    uint64        `json:"skip_count"`
	MinExecTime  time.Duration `json:"min_exec_time"`
	AvgExecTime  time.Duration `json:"avg_exec_time"`
	MaxExecTime  time.Duration `json:"max_exec_time"`
	TaskNames    []string      `json:"task_names,omitempty"` // The list of finished task names
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// Inc counters
func (task *TaskInfo) Inc(err error, execTime time.Duration) {
	task.TotalCount++
	if err != nil {
		if errors.Is(err, errors.ErrSkipEvent) || strings.Contains(err.Error(), "skip event") {
			task.SkipCount++
		} else {
			task.ErrorCount++
		}
	} else {
		task.SuccessCount++
	}
	if task.MinExecTime == 0 || task.MinExecTime > execTime {
		task.MinExecTime = execTime
	}
	if task.MaxExecTime == 0 || task.MaxExecTime < execTime {
		task.MaxExecTime = execTime
	}
	if task.AvgExecTime == 0 {
		task.AvgExecTime = execTime
	} else {
		task.AvgExecTime = (task.AvgExecTime*time.Duration(task.TotalCount) + execTime) /
			time.Duration(task.TotalCount+1)
	}
}

// Add task info
func (task *TaskInfo) Add(info *TaskInfo) {
	if !info.IsInited() {
		return
	}
	task.TotalCount += info.TotalCount
	task.ErrorCount += info.ErrorCount
	task.SuccessCount += info.SuccessCount
	task.SkipCount += info.SkipCount
	if task.MinExecTime == 0 || task.MinExecTime > info.MinExecTime {
		task.MinExecTime = info.MinExecTime
	}
	if task.MaxExecTime == 0 || task.MaxExecTime < info.MaxExecTime {
		task.MaxExecTime = info.MaxExecTime
	}
	if task.TotalCount > 0 && info.TotalCount > 0 {
		task.AvgExecTime = (task.AvgExecTime*time.Duration(task.TotalCount) +
			info.AvgExecTime*time.Duration(info.TotalCount)) /
			(time.Duration(task.TotalCount) + time.Duration(info.TotalCount))
	}
	for _, name := range info.TaskNames {
		task.AddTaskName(name)
	}
	task.touch()
}

// AddTaskName to the list
func (task *TaskInfo) AddTaskName(name string) {
	for _, taskName := range task.TaskNames {
		if taskName == name {
			return
		}
	}
	task.TaskNames = append(task.TaskNames, name)
	task.touch()
}

func (task *TaskInfo) IsInited() bool {
	return task != nil && !task.CreatedAt.IsZero()
}

func (task *TaskInfo) touch() {
	now := time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	if now.Sub(task.UpdatedAt) > 0 {
		task.UpdatedAt = now
	}
}
