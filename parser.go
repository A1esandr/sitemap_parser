package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
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

func main() {
	New().Parse()
}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Parse() {
	url := os.Getenv("SITE")
	if len(url) == 0 {
		log.Fatal("no site url env found")
	}
	if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		log.Fatal("site url must starts from http:// or https://")
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
	var wg sync.WaitGroup
	for i, v := range urls {
		wg.Add(1)
		go func(url string, i int) {
			fmt.Println(url)
			page := p.get(url)
			doc, err := html.Parse(bytes.NewReader(page))
			if err != nil {
				log.Fatal(err)
			}
			urls[i].Title = p.parse(doc)
			wg.Done()
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
