package main

import (
	"fmt"
	"slices"
	"sync"

	"github.com/holmes89/gofeed"
	"golang.org/x/sync/errgroup"
)

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

func (s *ArticleStore) AddArticle(item *gofeed.Item) {
	s.mu.Lock()                          // Lock the mutex before modifying it.
	defer s.mu.Unlock()                  // Ensure the lock is released after the function returns.
	if _, ok := s.cache[item.Link]; ok { // Check if the article already exists in the cache.
		return
	}

	s.cache[item.Link] = nil // Add the article to the cache to prevent duplicates.
	s.sorted = append(s.sorted, item)
	sortArticles(s.sorted) // Sort the articles by PublishedParsed (ascending) after adding a new one.
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
	store := NewArticleStore()
	var wg sync.WaitGroup
	feeds := []string{
		"https://rss.nytimes.com/services/xml/rss/nyt/World.xml",
		"https://feeds.bbci.co.uk/news/rss.xml",
	}

	for _, url := range feeds {
		wg.Add(1)
		go func(feedUrl string) {
			defer wg.Done()
			fp := gofeed.NewParser()
			feed, err := fp.ParseURL(feedUrl)
			if err != nil {
				fmt.Println("Error fetching feed:", err)
				return
			}
			store.AddArticles(feed.Items)
		}(url)
	}

	wg.Wait()

	// Retrieve and print the 5 most recent articles.
	recent := store.GetRecent(5)
	fmt.Println("Most recent articles:")
	for _, article := range recent {
		fmt.Printf("- %s (%s)\n", article.Title, article.Link)
	}

	fmt.Println("Stopped.")
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
