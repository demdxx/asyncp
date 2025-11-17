package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/demdxx/gocast/v2"
	"github.com/rivo/tview"
	cli "github.com/urfave/cli/v2"

	"github.com/demdxx/asyncp/v2/cmd/apmonitor/tabledata"
	"github.com/demdxx/asyncp/v2/monitor"
	"github.com/demdxx/asyncp/v2/monitor/driver/redis"
	"github.com/demdxx/asyncp/v2/monitor/kvstorage"
)

func main() {
	app := &cli.App{
		Name:  "apmonitor",
		Usage: "watch for the processing metrics",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "storage",
				Aliases:  []string{"s"},
				Usage:    "metrics storage connect redis://hostname:port/dbnum",
				EnvVars:  []string{"APMON_STORAGE_CONNECT"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "app",
				Aliases:  []string{"a"},
				Usage:    "application names by ','",
				EnvVars:  []string{"APMON_APPNAME"},
				Required: true,
			},
			&cli.DurationFlag{
				Name:    "interval",
				Aliases: []string{"i"},
				Usage:   "refresh interval",
				EnvVars: []string{"APMON_REFRESH_INTERVAL"},
				Value:   time.Second,
			},
		},
		Action: runMonitor,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func runMonitor(c *cli.Context) error {
	var (
		applicationName = c.String("app")
		storageURL      = c.String("storage")
		interval        = c.Duration("interval")
		storage, err    = connectStorage(storageURL, applicationName)
	)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	app := tview.NewApplication()

	tableData := tabledata.NewTableData(nil)
	tableData.SetHeaders([]string{"task", "min", "max", "avg", "success", "skip", "error", "total"})
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetContent(tableData)

	go func() {
		iter := 0
		for {
			select {
			case <-c.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				iter++
				updateInfo(iter, app, tableData, storage)
			}
		}
	}()

	return app.SetRoot(table, true).EnableMouse(true).Run()
}

func connectStorage(connectURL, applicationName string) (monitor.ClusterInfoReader, error) {
	parsedURL, err := url.Parse(connectURL)
	if err != nil {
		return nil, err
	}
	switch parsedURL.Scheme {
	case "redis":
		kvaccessor, err := redis.New(connectURL)
		if err != nil {
			return nil, err
		}
		return kvstorage.NewClusterInfoReader(kvaccessor,
			strings.Split(applicationName, ",")...), nil
	default:
		return nil, fmt.Errorf("unsupported monitor storage: %s", parsedURL.Scheme)
	}
}

func updateInfo(iter int, app *tview.Application, tableData *tabledata.TableData, info monitor.ClusterInfoReader) {
	appInfo, _ := info.ApplicationInfo()
	nodeCount := 0

	data := [][]string{}

	if appInfo != nil && appInfo.Tasks != nil {
		if appInfo.Servers != nil {
			nodeCount = len(appInfo.Servers)
		}
		for taskName := range appInfo.Tasks {
			if strings.HasPrefix(taskName, "@") {
				// Skip linked global event as it only for event type separating
				continue
			}
			taskInfo, _ := info.TaskInfo(taskName)
			item := []string{taskName, "?", "?", "?", "?", "?", "?", "?"}
			if taskInfo != nil {
				item[1] = taskInfo.MinExecTime.String()
				item[2] = taskInfo.MaxExecTime.String()
				item[3] = taskInfo.AvgExecTime.String()
				item[4] = gocast.Str(taskInfo.SuccessCount)
				item[5] = gocast.Str(taskInfo.SkipCount)
				item[6] = gocast.Str(taskInfo.ErrorCount)
				item[7] = gocast.Str(taskInfo.TotalCount)
			}
			data = append(data, item)
		}
	}

	if len(data) > 1 {
		sort.Slice(data, func(i, j int) bool { return data[i][0] < data[j][0] })
	}

	tableData.SetData(data)
	tableData.SetFooter([]string{"", "", "", "", "", "",
		gocast.IfThen(iter%2 == 0, "Nodes ", "Nodes:"),
		gocast.Str(nodeCount)})

	app.Draw()
}
