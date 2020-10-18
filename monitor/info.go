package monitor

import "time"

// ApplicationInfo with basic description
type ApplicationInfo struct {
	Name     string               `json:"name"`
	Host     string               `json:"host"`
	Hostname string               `json:"hostname"`
	InitedAt time.Time            `json:"inited_at"`
	Tasks    map[string]string    `json:"tasks"`
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
			app.Tasks = make(map[string]string, len(info.Tasks))
		}
		for taskName, taskTarget := range info.Tasks {
			app.Tasks[taskName] = taskTarget
		}
	}
}

// TaskInfo aggregated in one record
type TaskInfo struct {
	TotalCount   uint64        `json:"total_count"`
	ErrorCount   uint64        `json:"error_count"`
	SuccessCount uint64        `json:"success_count"`
	MinExecTime  time.Duration `json:"min_exec_time"`
	AvgExecTime  time.Duration `json:"avg_exec_time"`
	MaxExecTime  time.Duration `json:"max_exec_time"`
}

// Inc counters
func (task *TaskInfo) Inc(err error, execTime time.Duration) {
	task.TotalCount++
	task.SuccessCount++
	if task.MinExecTime == 0 || task.MinExecTime > execTime {
		task.MinExecTime = execTime
	}
	if task.MaxExecTime == 0 || task.MaxExecTime < execTime {
		task.MaxExecTime = execTime
	}
	if task.AvgExecTime == 0 {
		task.AvgExecTime = (task.AvgExecTime + execTime) / 2
	}
}

// Add task info
func (task *TaskInfo) Add(info *TaskInfo) {
	task.TotalCount += info.TotalCount
	task.ErrorCount += info.ErrorCount
	task.SuccessCount += info.SuccessCount
	if task.MinExecTime == 0 || task.MinExecTime > info.MinExecTime {
		task.MinExecTime = info.MinExecTime
	}
	if task.MaxExecTime == 0 || task.MaxExecTime < info.MaxExecTime {
		task.MaxExecTime = info.MaxExecTime
	}
	task.AvgExecTime = (task.AvgExecTime + info.AvgExecTime) / 2
}
