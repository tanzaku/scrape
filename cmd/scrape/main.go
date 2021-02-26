package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/antchfx/antch"
	"github.com/antchfx/htmlquery"

	"github.com/tanzaku/scrape/internal/message"
)

type item struct {
	Title string   `json:"title"`
	Desc  []string `json:"desc"`
}

type trimSpacePipeline struct {
	next antch.PipelineHandler
}

func (p *trimSpacePipeline) ServePipeline(v antch.Item) {
	vv := v.(*item)
	vv.Title = strings.TrimSpace(vv.Title)
	// vv.Desc = strings.TrimSpace(vv.Desc)
	p.next.ServePipeline(vv)
}

func newTrimSpacePipeline() antch.Pipeline {
	return func(next antch.PipelineHandler) antch.PipelineHandler {
		return &trimSpacePipeline{next}
	}
}

type jsonOutputPipeline struct{}

func (p *jsonOutputPipeline) ServePipeline(v antch.Item) {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	std := os.Stdout
	std.Write(b)
	std.Write([]byte("\n"))
}

func newJsonOutputPipeline() antch.Pipeline {
	return func(next antch.PipelineHandler) antch.PipelineHandler {
		return &jsonOutputPipeline{}
	}
}

type dmozSpider struct{}

func (s *dmozSpider) ServeSpider(c chan<- antch.Item, res *http.Response) {
	doc, err := antch.ParseHTML(res)
	if err != nil {
		panic(err)
	}
	for _, node := range htmlquery.Find(doc, "//dl[@class='common-warn-entries is-alert-information clearfix']") {
		v := new(item)
		// v.Title = htmlquery.InnerText(htmlquery.FindOne(node, "//dt"))
		v.Title = htmlquery.InnerText(htmlquery.FindOne(node, "/dt"))
		list := htmlquery.Find(node, "/dd[@class='alert-entry']")
		v.Desc = make([]string, 0, len(list))
		for _, n := range list {
			v.Desc = append(v.Desc, htmlquery.InnerText(n))
		}
		fmt.Println(message.Hello())
		c <- v
	}
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	c := make(chan struct{})
	crawler := &antch.Crawler{Exit: c}
	crawler.UseCompression()

	crawler.Handle("tenki.jp", &dmozSpider{})
	crawler.UsePipeline(newTrimSpacePipeline(), newJsonOutputPipeline())

	startURLs := []string{
		"https://tenki.jp/forecast/3/17/4610/14134/",
	}

	go func() {
		crawler.StartURLs(startURLs)
		<-sigs // `CTRL-C` to stop crawler.
		close(c)
	}()
	// crawler is block waiting for a signal.
	<-crawler.Exit
	fmt.Println("exiting crawler")
}
