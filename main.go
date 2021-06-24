package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
)

type Chapter struct {
	Name string
	Href string
}

func main() {
	comicId := "11"
	downloadComic(comicId)
}

func downloadComic(comicId string) {
	parallel := 10 // 并发数

	c := colly.NewCollector()
	chapterList := make([]Chapter, 0)
	// 漫画名称
	comicName := ""
	c.OnHTML("h2", func(h *colly.HTMLElement) {
		comicName = h.Text
		index := strings.Index(comicName, "漫画")
		comicName = comicName[:index]
	})
	// 章节列表
	c.OnHTML("li[class='pure-u-1-2 pure-u-lg-1-4']", func(e *colly.HTMLElement) {
		href := e.ChildAttr("a", "href")
		name := e.ChildAttr("a", "title")
		chapterList = append(chapterList, Chapter{
			Name: name,
			Href: href,
		})
	})

	c.Visit("https://manhua.fffdm.com/" + comicId + "/")
	c.Wait()

	// 每章下载
	n := len(chapterList)
	fmt.Println("total:", n)
	ch := make(chan int, parallel) // 限制最大并发
	var wg sync.WaitGroup
	wg.Add(n)
	for i, chapter := range chapterList {
		ch <- 1
		go downloadChapter(comicId, comicName, chapter.Href, chapter.Name, n-i, ch, &wg)
	}
	wg.Wait() // 等待所有 goroutine 完成
}

func downloadChapter(comicId, comicName, chapterId, chapterName string, i int, ch chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	downloadPath := "./comic/" + comicName + "/"

	c := colly.NewCollector()
	imageC := c.Clone()
	// 获取 img 地址
	c.OnResponse(func(r *colly.Response) {
		reg := regexp.MustCompile(`(mhurl = ")([0-9a-zA-Z./_]+)(")`)
		path := reg.FindStringSubmatch(string(r.Body))
		if len(path) > 2 {
			uri := path[2]
			var url string
			if strings.HasPrefix(uri, "201") {
				url = "http://www-mipengine-org.mipcdn.com/i/p3.manhuapan.com/" + uri
			} else {
				url = "https://p5.manhuapan.com/" + uri
			}
			imageC.Visit(url)
		}
	})

	// 获取下一页
	c.OnHTML("a[class='pure-button pure-button-primary']", func(h *colly.HTMLElement) {
		href := h.Attr("href")
		url := "https://manhua.fffdm.com/" + comicId + "/" + chapterId + href
		c.Visit(url)
	})

	id := 0
	imageC.OnResponse(func(r *colly.Response) {
		thisPath := downloadPath + strconv.Itoa(i) + " " + chapterName
		os.MkdirAll(thisPath, os.ModePerm)
		filePath := thisPath + "/" + strconv.Itoa(id) + ".jpg"
		f, err := os.Create(filePath)
		id++
		if err != nil {
			panic(err)
		}
		io.Copy(f, bytes.NewReader(r.Body))
	})

	c.Visit("https://manhua.fffdm.com/" + comicId + "/" + chapterId)
	c.Wait()
	fmt.Println(i, chapterName)
	<-ch
}
