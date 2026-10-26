package main

import (
	"sync"
	"testing"
	"time"

	"github.com/holmes89/gofeed"
)

// TestGetRecentAliasesSorted proves that a slice returned by GetRecent is
// mutated in place by a concurrent AddArticles -> sortArticles, i.e. a data
// race once GetRecent has released the read lock.
func TestGetRecentAliasesSorted(t *testing.T) {
	s := NewArticleStore()

	base := time.Now()
	seed := make([]*gofeed.Item, 200)
	for i := range seed {
		ts := base.Add(time.Duration(i) * time.Minute)
		seed[i] = &gofeed.Item{Link: "seed-" + ts.String(), PublishedParsed: &ts}
	}
	s.AddArticles(seed)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writers: keep adding new items so sortArticles mutates s.sorted in place.
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				ts := base.Add(time.Duration(w*i+i) * time.Second)
				s.AddArticles([]*gofeed.Item{{Link: "w" + string(rune(w)) + "-" + ts.String(), PublishedParsed: &ts}})
			}
		}(w)
	}

	// Readers: mimic articlesHandler — take the slice, then read it after the
	// read lock has been released.
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				for _, it := range s.GetRecent(100) {
					_ = it.Link
					if it.PublishedParsed != nil {
						_ = it.PublishedParsed.Unix()
					}
				}
			}
		}()
	}

	time.Sleep(500 * time.Millisecond)
	close(stop)
	wg.Wait()
}
