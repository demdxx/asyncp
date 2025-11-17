package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/demdxx/asyncp/v2"
	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/geniusrabbit/notificationcenter/v2/gochan"
)

type rssItem struct {
	Title       string `json:"title" xml:"title"`
	Link        string `json:"link" xml:"link"`
	Description string `json:"description" xml:"description"`
	PubDate     string `json:"lastBuildDate" xml:"lastBuildDate"`
	Category    string `json:"category" xml:"category"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Language      string    `xml:"language"`
	Item          []rssItem `xml:"item"`
}

type rss struct {
	XMLName xml.Name     `xml:"rss"`
	Channel []rssChannel `xml:"channel"`
}

var loadCounter int

func downloadRSSList(ctx context.Context, event asyncp.Event, responseWriter asyncp.ResponseWriter) error {
	fmt.Println("downloadProxyList", event)
	var rsslink string
	if err := event.Payload().Decode(&rsslink); err != nil {
		return err
	}
	res, err := http.Get(rsslink)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var rss rss
	if err = xml.Unmarshal(data, &rss); err != nil {
		return err
	}
	for _, channel := range rss.Channel {
		fmt.Println("Chanel:     ", channel.Title, channel.Language)
		fmt.Println("Link:       ", channel.Link)
		fmt.Println("Description:", channel.Description)
		for _, item := range channel.Item {
			err := responseWriter.WriteResonse(item)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func downloadRSSItem(ctx context.Context, event asyncp.Event, responseWriter asyncp.ResponseWriter) error {
	var item rssItem
	if err := event.Payload().Decode(&item); err != nil {
		return err
	}
	fmt.Println("\n============================================")
	fmt.Println("Download item: ", item.Link)
	fmt.Println("Download title:", item.Title)

	res, err := http.Get(item.Link)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	loadCounter++
	return responseWriter.WriteResonse(string(data))
}

func printResults(ctx context.Context, event asyncp.Event, responseWriter asyncp.ResponseWriter) error {
	var data string
	if err := event.Payload().Decode(&data); err != nil {
		return err
	}
	fmt.Println("[[DATA]]:", len(data), loadCounter)
	return responseWriter.WriteResonse(loadCounter)
}

func closeAction(sub nc.Subscriber) asyncp.FuncTask {
	return asyncp.FuncTask(func(ctx context.Context, event asyncp.Event, responseWriter asyncp.ResponseWriter) error {
		if loadCounter--; loadCounter <= 0 {
			return sub.Close()
		}
		return nil
	})
}

func main() {
	mempr := gochan.New(100)
	proxy := asyncp.NewProxySubscriber(mempr)

	mx := asyncp.NewTaskMux(asyncp.WithStreamResponsePublisher(mempr.Publisher()))
	mx.Handle("rss", downloadRSSList).
		Then(downloadRSSItem).
		Then(printResults).
		Then(closeAction(proxy))
	_ = mx.Failover(asyncp.Retranslator(0, mempr.Publisher()))

	_ = mempr.Publisher().Publish(context.Background(), asyncp.WithPayload("rss", "https://www.uber.com/blog/rss/"))

	_ = proxy.Subscribe(context.Background(), mx)
	_ = proxy.Listen(context.Background())
}
