package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/holmes89/gofeed"
)

type App struct { // Create an app struct to hold our store.
	store *ArticleStore
}

type ArticleStore struct {
	mu     sync.RWMutex   // Use a regular mutex (mu.Lock/Unlock) to protect writes and modifications to the store.
	cache  map[string]any // This cache will be used to check if a article has already been added.
	sorted []*gofeed.Item
}

func NewArticleStore() *ArticleStore {
	return &ArticleStore{
		cache: make(map[string]any),
	}
}

func (a *App) articlesHandler(w http.ResponseWriter, r *http.Request) { // Create a new handler based on the Handler type definition.
	numberofArticles := 10
	if sizeStr := r.URL.Query().Get("size"); sizeStr != "" { // Find a query parameter for the length of the list of articles to return.
		if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
			numberofArticles = size
		}
	}
	articles := a.store.GetRecent(numberofArticles)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles) // Encode article values.
}

func (s *ArticleStore) AddArticles(items []*gofeed.Item) {
	s.mu.Lock()                  // Lock the mutex before batch modification.
	defer s.mu.Unlock()          // Ensure the lock is released after the function returns.
	var hasNewArticles bool      // Track if any new articles were added.
	for _, item := range items { // Iterate over the incoming articles.
		if _, ok := s.cache[item.Link]; ok { // Skip articles already in the cache.
			continue
		}
		hasNewArticles = true
		s.cache[item.Link] = nil
		s.sorted = append(s.sorted, item)
	}
	if hasNewArticles { // Only sort if new articles were added.
		sortArticles(s.sorted)
	}
}

func (s *ArticleStore) GetRecent(n int) []*gofeed.Item {
	s.mu.RLock()           // Create a read lock.
	defer s.mu.RUnlock()   // Ensure the lock is released.
	if n > len(s.sorted) { // If the slice is smaller than the total requested, return the whole list.
		return s.sorted
	}
	return s.sorted[:n] // Truncate to the requested size.
}

func (a *App) fetchFeed(ctx context.Context, url string) (gofeed.Feed, error) { // Change functions to methods on the new struct.
	fmt.Println("Fetching feed:", url)
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(url, ctx) // Use the context-based call to leverage cancellation.
	fmt.Println("Fetched:", url)
	out := make(chan *gofeed.Item) // Create a channel to pass to the save function.
	go saveFeed(out)               // Pass as part of a goroutine.
	defer close(out)               // Close the channel when done.
	for _, article := range feed.Items {
		fmt.Println("Ingesting article:", article.Link)
		out <- article // Write values to channel.
	}
	fmt.Println("Processed:", url)
	return *feed, err
}

func (a *App) fetchFeeds(ctx context.Context, store *ArticleStore, urls []string) chan error { // You can return a channel just like any other type.
	errChan := make(chan error) // Use the make keyword to create a new unbuffered channel.
	for _, url := range urls {
		go func() {
			feed, err := a.fetchFeed(ctx, url)
			if err != nil {
				errChan <- err // Add items to a channel via the ← operator.
			}
			store.AddArticles(feed.Items)
			fmt.Println("Feed title:", feed.Title)
		}()
	}
	return errChan // Return the channel at the end.
}

func saveFeed(articles chan *gofeed.Item) { // Pass in channels like any other parameter.
	fmt.Println("Adding articles...")
	for article := range articles { // Read data from channel using range.
		// fmt.Println("Adding article:", article.Link)
		fmt.Println("Article added:", article.Link)
	}
	fmt.Println("All articles added.")
}

func main() {
	fmt.Println("Starting...")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Create initial context with a timeout.
	defer cancel()                                                           // Call cancellation function before returning.

	store := NewArticleStore() // Create a store to put in the struct.
	app := &App{               // Initialize new structure.
		store: store,
	}

	app.fetchFeeds(ctx, store, []string{
		"https://rss.nytimes.com/services/xml/rss/nyt/World.xml",
		"https://feeds.bbci.co.uk/news/rss.xml",
	})

	http.HandleFunc("/articles", app.articlesHandler) // Handle calls to the /articles endpoint.
	fmt.Println("HTTP server listening on :8080")
	// curl -v -s http://localhost:8080/articles | jq

	if err := http.ListenAndServe(":8080", nil); err != nil { // Run the HTTP server until an error occurs.
		fmt.Println("Server error:", err)
	}
}

// sortArticles sorts a slice of *gofeed.Item in ascending order by PublishedParsed.
// *gofeed.Item is not cmp.Ordered, so slices.Sort cannot be used; SortFunc with a
// custom comparator is required. nil PublishedParsed values sort first.
func sortArticles(items []*gofeed.Item) {
	slices.SortFunc(items, func(a, b *gofeed.Item) int {
		ta, tb := a.PublishedParsed, b.PublishedParsed
		switch {
		case ta == nil && tb == nil:
			return 0
		case ta == nil:
			return -1 // nil timestamps sort first.
		case tb == nil:
			return 1
		default:
			return ta.Compare(*tb)
		}
	})
}
