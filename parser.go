package parser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	pth "github.com/A1esandr/sitemap_parser/internal/path"
)

type (
	Parser struct {
		data map[string][]byte
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

	Sitemapindex struct {
		XMLName xml.Name  `xml:"sitemapindex"`
		Sitemap []Sitemap `xml:"sitemap"`
	}

	Sitemap struct {
		XMLName xml.Name `xml:"sitemap"`
		Loc     string   `xml:"loc"`
	}
)

var site = flag.String("site", "", "URL of the site, for example, https://alextech18.blogspot.com")
var path = flag.String("backup", "", "backup path")

func New() *Parser {
	return &Parser{data: make(map[string][]byte)}
}

func (p *Parser) Get(url string) []URL {
	if len(url) == 0 {
		log.Fatal("no site url found")
	}
	return p.getList(url)
}

func (p *Parser) Parse() {
	flag.Parse()
	url := os.Getenv("SITE")
	if len(url) == 0 {
		url = *site
	}
	if len(url) == 0 {
		log.Fatal("no site url found")
	}

	backupPath := os.Getenv("BACKUP_PATH")
	if len(backupPath) == 0 {
		backupPath = *path
	}

	urlList := strings.Split(url, ",")
	for _, v := range urlList {
		p.process(v, backupPath)
	}
}

func (p *Parser) process(url, baseBackupPath string) {
	if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		log.Fatal("site url must starts from http:// or https:// for", url)
	}

	var backupPath string
	if len(baseBackupPath) > 0 {
		configurator := pth.NewPathConfigurator()
		backupPath = configurator.Configure(baseBackupPath, url)
	}

	baseURL := url
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "sitemap.xml"
	data := p.get(url, 0)
	var urls []URL

	if strings.Contains(string(data), "<sitemapindex") {
		var sitemapindex Sitemapindex
		err := xml.Unmarshal(data, &sitemapindex)
		if err != nil {
			log.Fatalf("error parse response xml %s", err.Error())
		}
		for _, sitemap := range sitemapindex.Sitemap {
			data = p.get(sitemap.Loc, 0)
			pageUrls := p.decode(data)
			urls = append(urls, pageUrls...)
		}
	} else {
		urls = p.decode(data)
	}

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
			page := p.get(url, 0)
			p.backup(page, url, backupPath)
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
	p.archive(backupPath, baseURL)
	p.printList(urls)
}

func (p *Parser) getList(url string) []URL {
	if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		log.Fatal("site url must starts from http:// or https:// for", url)
	}
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "sitemap.xml"
	data := p.get(url, 0)
	var urls []URL

	if strings.Contains(string(data), "<sitemapindex") {
		var sitemapindex Sitemapindex
		err := xml.Unmarshal(data, &sitemapindex)
		if err != nil {
			log.Fatalf("error parse response xml %s", err.Error())
		}
		for _, sitemap := range sitemapindex.Sitemap {
			data = p.get(sitemap.Loc, 0)
			pageUrls := p.decode(data)
			urls = append(urls, pageUrls...)
		}
	} else {
		urls = p.decode(data)
	}

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
			page := p.get(url, 0)
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
	return urls
}

func (p *Parser) decode(data []byte) []URL {
	var urlset URLSet
	err := xml.Unmarshal(data, &urlset)
	if err != nil {
		log.Fatalf("error parse response xml %s", err.Error())
	}
	return urlset.URL
}

func (p *Parser) get(url string, count int) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("error get %s: %s", url, err.Error())
	}
	if resp == nil {
		log.Fatalf("nil response from %s", url)
	}
	if resp.StatusCode != http.StatusOK && count < 3 {
		log.Println("Error loading", url)
		time.Sleep(time.Duration(300+rand.Intn(1000)) * time.Millisecond)
		return p.get(url, count+1)
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

func (p *Parser) backup(file []byte, url, backupPath string) {
	if len(backupPath) == 0 || !strings.HasSuffix(url, ".html") {
		return
	}
	name := strings.ReplaceAll(url, "://", "")
	name = strings.ReplaceAll(name, "/", "_")
	err := os.WriteFile(backupPath+name, file, 0644)
	if err != nil {
		panic(err)
	}
	p.data[name] = file
}

func (p *Parser) archive(backupPath, baseURL string) {
	if len(backupPath) == 0 || len(p.data) == 0 {
		return
	}
	name := strings.ReplaceAll(baseURL, "http://", "")
	name = strings.ReplaceAll(name, "https://", "")
	name = strings.ReplaceAll(name, "/", "_")
	t := time.Now()
	zipFile := t.Format("2006-01-02") + "_" + name + "_archive.zip"
	out, err := os.Create(backupPath + zipFile)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		closeErr := out.Close()
		if closeErr != nil {
			log.Fatalf("error close %s", closeErr.Error())
		}
	}()

	w := zip.NewWriter(out)

	for name, file := range p.data {
		f, err := w.Create(name)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write(file)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = w.Close()
	if err != nil {
		log.Fatal(err)
	}
}
