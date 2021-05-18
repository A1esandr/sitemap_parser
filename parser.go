package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type (
	Parser struct {
		backupPath string
	}

	URLSet struct {
		XMLName xml.Name `xml:"urlset"`
		URL     []URL    `xml:"url"`
	}

	URL struct {
		XMLName xml.Name `xml:"url"`
		Loc     string   `xml:"loc"`
		LastMod string   `xml:"lastmod"`
		Title   string
	}
)

var site = flag.String("site", "", "URL of the site, for example, https://alextech18.blogspot.com")
var path = flag.String("backup", "", "backup path")

func main() {
	flag.Parse()
	New().Parse()
}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Parse() {
	url := os.Getenv("SITE")
	if len(url) == 0 {
		url = *site
	}
	if len(url) == 0 {
		log.Fatal("no site url found")
	}
	if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		log.Fatal("site url must starts from http:// or https://")
	}

	backupPath := os.Getenv("BACKUP_PATH")
	if len(backupPath) == 0 {
		backupPath = *path
	}

	if len(backupPath) > 0 {
		configurator := NewPathConfigurator()
		p.backupPath = configurator.Configure(backupPath, url)
	}

	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "sitemap.xml"
	data := p.get(url)
	var urlset URLSet
	err := xml.Unmarshal(data, &urlset)
	if err != nil {
		log.Fatalf("error parse response xml %s", err.Error())
	}
	urls := urlset.URL
	sort.SliceStable(urls, func(i, j int) bool {
		return urls[i].LastMod < urls[j].LastMod
	})
	c := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		c <- struct{}{}
	}
	var wg sync.WaitGroup
	for i, v := range urls {
		<-c
		wg.Add(1)
		go func(url string, i int) {
			fmt.Println(url)
			page := p.get(url)
			p.backup(page, url)
			doc, err := html.Parse(bytes.NewReader(page))
			if err != nil {
				log.Fatal(err)
			}
			urls[i].Title = p.parse(doc)
			wg.Done()
			c <- struct{}{}
		}(v.Loc, i)
	}
	wg.Wait()
	p.printList(urls)
}

func (p *Parser) get(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("error get %s: %s", url, err.Error())
	}
	if resp == nil {
		log.Fatalf("nil response from %s", url)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			log.Fatalf("error close response body %s", closeErr.Error())
		}
	}()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error resp response body %s", err.Error())
	}
	return data
}

func (p *Parser) parse(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "h3" {
		for _, at := range n.Attr {
			if at.Key == "class" && at.Val == "post-title entry-title" {
				return strings.ReplaceAll(n.FirstChild.Data, "\n", "")
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result := p.parse(c)
		if len(result) > 0 {
			return result
		}
	}
	return ""
}

func (p *Parser) printList(urls []URL) {
	var sb strings.Builder
	sb.WriteString("<ol>\n")
	for _, v := range urls {
		sb.WriteString("<li><a href=\"")
		sb.WriteString(v.Loc)
		sb.WriteString("\">")
		sb.WriteString(v.Title)
		sb.WriteString("</a></li>\n")
	}
	sb.WriteString("</ol>")
	fmt.Println(sb.String())
}

func (p *Parser) backup(file []byte, url string) {
	if len(p.backupPath) == 0 || !strings.HasSuffix(url, ".html") {
		return
	}
	name := strings.ReplaceAll(url, "://", "")
	name = strings.ReplaceAll(name, "/", "_")
	err := ioutil.WriteFile(p.backupPath+name, file, 0644)
	if err != nil {
		panic(err)
	}
}
