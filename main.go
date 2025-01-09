package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog"
)

const (
	url               = "https://www.binance.com/en/support/announcement/delisting?c=161&navId=161"
	pathBlacklistFile = "blacklist.json"
	pathProcessedFile = "processed.json"
	pathBotsFile      = "bots.json"
	loopSecs          = 90
)

var (
	tokens           []string
	hasBeenProcessed []string
	bots             []map[string]string
)

type Link struct {
	Href  string
	Title string
}

type ListItem struct {
	Spans []SpanItem
}

type SpanItem struct {
	Text string
}

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func main() {
	// loadBotsData()
	// openLocalBlacklist()
	// sendBlacklist(tokens)
	// openLocalProcessed()

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set options for headless mode
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	startTime := time.Now()
	for {
		getDelistTokens(ctx)
		time.Sleep(time.Duration(loopSecs-time.Since(startTime).Seconds()) * time.Second)
	}
}

func getDelistTokens(ctx context.Context) {
	classPListCoins := "css-zwb0rk"
	newBlacklist := []string{}
	newProcessed := []string{}

	log.Println("Scraping delisting page")
	var htmlSource string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.OuterHTML("html", &htmlSource),
	)
	if err != nil {
		log.Println("Failed to navigate:", err)
		return
	}

	links := extractLinks(htmlSource)

	countNotice := 5
	for _, link := range links {
		title := strings.ToUpper(link.Title)
		if title != "" && !contains(hasBeenProcessed, title) && !contains(newProcessed, title) {
			log.Printf("New title: %s\n", title)
			newProcessed = append(newProcessed, title)
			if strings.Contains(title, "BINANCE WILL DELIST ") {
				title = strings.Replace(title, "BINANCE WILL DELIST ", "", 1)
				arrTitle := strings.Split(title, " ON ")
				arrCoins := strings.Split(arrTitle[0], ", ")
				for _, coin := range arrCoins {
					blacklist := fmt.Sprintf("%s/.*", coin)
					if !contains(tokens, blacklist) && !contains(newBlacklist, blacklist) {
						newBlacklist = append(newBlacklist, blacklist)
					}
				}
			} else if strings.Contains(title, "NOTICE OF REMOVAL OF ") && !strings.Contains(title, "MARGIN") && countNotice > 0 {
				countNotice--
				linkURL := fmt.Sprintf("https://www.binance.com%s", link.Href)
				err := chromedp.Run(ctx,
					chromedp.Navigate(linkURL),
					chromedp.OuterHTML("html", &htmlSource),
				)
				if err != nil {
					log.Println("Failed to navigate to link:", err)
					continue
				}

				doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlSource))
				if err != nil {
					log.Println("Error parsing HTML:", err)
					continue
				}

				doc.Find("p." + classPListCoins).Each(func(index int, p *goquery.Selection) {
					fmt.Println("dgvdgavshdv: ", p)
					p.Find("span." + "richtext-text").Each(func(i int, span *goquery.Selection) {

						if strings.Contains(span.Text(), "/") {
							line := strings.Replace(span.Text(), ":", "", 1)
							arrCoins := strings.Split(line, ", ")
							for _, coin := range arrCoins {
								coin = strings.TrimSpace(coin)

								fmt.Println("hbđahs", coin)
								if !contains(tokens, coin) && !contains(newBlacklist, coin) {
									newBlacklist = append(newBlacklist, coin)
								}
							}
						}
					})
				})
				// lis := extractListItems(htmlSource, classPListCoins)
				// fmt.Println("bdahjsb lis: ", lis)
				// for _, li := range lis {
				// 	for _, span := range li.Spans {
				// 		if strings.Contains(span.Text, "/") {
				// 			line := strings.Replace(span.Text, ":", "", 1)
				// 			arrCoins := strings.Split(line, ", ")
				// 			for _, coin := range arrCoins {
				// 				coin = strings.TrimSpace(coin)

				// 				fmt.Println("hbđahs %s", coin)
				// 				if !contains(tokens, coin) && !contains(newBlacklist, coin) {
				// 					newBlacklist = append(newBlacklist, coin)
				// 				}
				// 			}
				// 		}
				// 	}
				// }
			}
		}
	}

	if len(newProcessed) > 0 {
		hasBeenProcessed = append(hasBeenProcessed, newProcessed...)
		saveLocalProcessed()
	}

	if len(newBlacklist) > 0 {
		tokens = append(tokens, newBlacklist...)
		saveLocalBlacklist()
		sendBlacklist(newBlacklist)
	}
}

func extractLinks(html string) []Link {
	var links []Link

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Println("Error parsing HTML:", err)
		return links
	}

	doc.Find("a").Each(func(index int, item *goquery.Selection) {
		href, existsHref := item.Attr("href")
		title := item.Text() // Get the text content of the link

		if existsHref {
			links = append(links, Link{Href: href, Title: title})
		}
	})

	return links
}

func extractListItems(html string, className string) []ListItem {
	var listItems []ListItem

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Println("Error parsing HTML:", err)
		return listItems
	}

	doc.Find("p" + className).Each(func(index int, p *goquery.Selection) {
		var spans []SpanItem
		p.Find("span" + "richtext-text").Each(func(i int, span *goquery.Selection) {
			spans = append(spans, SpanItem{Text: span.Text()})
			listItems = append(listItems, ListItem{Spans: spans})
		})
	})

	return listItems
}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func openLocalBlacklist() {
	log.Println("Loading local blacklist file")
	file, err := os.Open(pathBlacklistFile)
	if err != nil {
		log.Fatalf("Config file \"%s\" not found! Please create a config file or check whether it exists.", pathBlacklistFile)
	}
	defer file.Close()

	var config struct {
		PairBlacklist []string `json:"pair_blacklist"`
	}
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
	}

	tokens = append(tokens, config.PairBlacklist...)
}

func saveLocalBlacklist() {
	log.Println("Saving local blacklist file")
	newBlacklist := map[string]interface{}{
		"pair_blacklist": tokens,
	}
	jsonData, _ := json.Marshal(newBlacklist)
	ioutil.WriteFile(pathBlacklistFile, jsonData, 0644)
}

func openLocalProcessed() {
	log.Println("Loading local processed file")
	file, err := os.Open(pathProcessedFile)
	if err != nil {
		log.Fatalf("Config file \"%s\" not found! Please create a config file or check whether it exists.", pathProcessedFile)
	}
	defer file.Close()

	var config struct {
		Processed []string `json:"processed"`
	}
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
	}

	hasBeenProcessed = append(hasBeenProcessed, config.Processed...)
}

func saveLocalProcessed() {
	log.Println("Saving local processed file")
	newProcessed := map[string]interface{}{
		"processed": hasBeenProcessed,
	}
	jsonData, _ := json.Marshal(newProcessed)
	os.WriteFile(pathProcessedFile, jsonData, 0644)
}

func loadBotsData() {
	file, err := os.Open(pathBotsFile)
	if err != nil {
		log.Fatalf("Config file \"%s\" not found! Please create a config file or check whether it exists.", pathBotsFile)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&bots); err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
	}
}

func sendBlacklist(blacklist []string) {
	fmt.Printf("sssss%+v\n", blacklist)

	if len(blacklist) > 0 {
		for _, bot := range bots {
			log.Printf("Send blacklist list to %s\n", bot["ip_address"])
			// Implement API call to send blacklist to bot
		}
	}
}
