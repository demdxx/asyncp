package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/demdxx/gocast"
	"github.com/olekukonko/tablewriter"
	"github.com/rivo/tview"
	cli "github.com/urfave/cli/v2"

	"github.com/demdxx/asyncp/monitor"
	"github.com/demdxx/asyncp/monitor/driver/redis"
	"github.com/demdxx/asyncp/monitor/kvstorage"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(interval)
	app := tview.NewApplication()
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() { app.Draw() })

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				updateInfo(textView, storage)
			}
		}
	}()

	return app.SetRoot(textView, true).Run()
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

func updateInfo(textView *tview.TextView, info monitor.ClusterInfoReader) {
	textView.Clear()

	appInfo, _ := info.ApplicationInfo()
	nodeCount := 0

	data := [][]string{}

	if appInfo != nil && appInfo.Tasks != nil {
		if appInfo.Servers != nil {
			nodeCount = len(appInfo.Servers)
		}
		for taskName := range appInfo.Tasks {
			taskInfo, _ := info.TaskInfo(taskName)
			item := []string{taskName, "?", "?", "?", "?", "?", "?"}
			if taskInfo != nil {
				item[1] = taskInfo.MinExecTime.String()
				item[2] = taskInfo.MaxExecTime.String()
				item[3] = taskInfo.AvgExecTime.String()
				item[4] = gocast.ToString(taskInfo.SuccessCount)
				item[5] = gocast.ToString(taskInfo.ErrorCount)
				item[6] = gocast.ToString(taskInfo.TotalCount)
			}
			data = append(data, item)
		}
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i][0] < data[j][0]
	})

	table := tablewriter.NewWriter(textView)
	table.SetHeader([]string{"task", "min", "max", "avg", "success", "error", "total"})
	table.SetFooter([]string{"", "", "", "", "", "Nodes", gocast.ToString(nodeCount)})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetAutoMergeCells(false)
	table.SetRowLine(false)
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()
}
