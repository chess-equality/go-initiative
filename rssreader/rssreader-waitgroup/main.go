package main

import (
	"fmt"

	"github.com/holmes89/gofeed"
	"golang.org/x/sync/errgroup"
)

func printFeedTitle(url string) {
	fmt.Println("Fetching feed:", url)
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL(url)
	fmt.Println("Feed title:", feed.Title)
	fmt.Println("Fetched:", url)
}

func fetchFeed(url string) (gofeed.Feed, error) { // Change the function to return errors.
	fmt.Println("Fetching feed:", url)
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	fmt.Println("Fetched:", url)
	return *feed, err
}

func fetchFeeds(urls []string) {
	eg := new(errgroup.Group) // Use errgroup to manage goroutines and error handling.
	for _, url := range urls {
		eg.Go(func() error { // Closure must return an error.
			feed, err := fetchFeed(url)
			if err != nil {
				return err
			}
			fmt.Println("Feed title:", feed.Title)
			return nil
		})
	}
	fmt.Println("Waiting...")
	err := eg.Wait() // Wait returns an error if any goroutine failed.
	if err != nil {
		fmt.Println("Not all urls were fetched:", err)
	}
}

func main() {
	fmt.Println("Starting...")
	urls := []string{
		"https://rss.nytimes.com/services/xml/rss/nyt/World.xml",
		"https://feeds.bbci.co.uk/news/rss.xml",
	}
	fetchFeeds(urls)
	fmt.Println("Stopped.")
}
