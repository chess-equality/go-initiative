package main

import (
	"context"
	"fmt"
	"time"

	"github.com/holmes89/gofeed"
	"golang.org/x/sync/errgroup"
)

func fetchFeed(ctx context.Context, url string) (gofeed.Feed, error) { // Pass context into the fetch function.
	fmt.Println("Fetching feed:", url)
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(url, ctx) // Use the context-based call to leverage cancellation.
	fmt.Println("Fetched:", url)
	return *feed, err
}

func fetchFeeds(ctx context.Context, urls []string) { // Modify the function to pass the context.
	eg, ectx := errgroup.WithContext(ctx) // Create a new context for errgroup that appends its own cancellation policy.
	for _, url := range urls {
		eg.Go(func() error { // Closure must return an error.
			feed, err := fetchFeed(ectx, url) // Pass in new context.
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Create initial context with a timeout.
	defer cancel()                                                           // Call cancellation function before returning.
	fetchFeeds(ctx, []string{
		"https://rss.nytimes.com/services/xml/rss/nyt/World.xml",
		"https://feeds.bbci.co.uk/news/rss.xml",
	})
	fmt.Println("Stopped.")
}
