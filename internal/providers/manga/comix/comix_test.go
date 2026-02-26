package comix

import (
	"context"
	"fmt"
	"testing"

	"github.com/justchokingaround/greg/internal/providers"
)

func TestChainsawMan(t *testing.T) {
	c := New()

	ctx := context.Background()

	// Step 1: Search for Chainsaw Man
	t.Log("=== Step 1: Searching for Chainsaw Man ===")
	results, err := c.Search(ctx, "chainsaw man")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("No results found")
	}

	var chainsawMan providers.Media
	for _, r := range results {
		t.Logf("Found: %s (ID: %s)", r.Title, r.ID)
		if r.Title == "Chainsaw Man" {
			chainsawMan = r
			break
		}
	}
	if chainsawMan.Title == "" {
		chainsawMan = results[0]
	}
	t.Logf("Selected: %s (ID: %s)", chainsawMan.Title, chainsawMan.ID)

	// Step 2: Get manga details (includes chapters)
	t.Log("=== Step 2: Getting manga details ===")
	details, err := c.GetMediaDetails(ctx, chainsawMan.ID)
	if err != nil {
		t.Fatalf("GetMediaDetails failed: %v", err)
	}
	t.Logf("Title: %s", details.Title)
	t.Logf("Synopsis: %s", details.Synopsis)
	t.Logf("Status: %s", details.Status)
	t.Logf("Seasons: %d", len(details.Seasons))

	// Step 3: Get chapters (episodes)
	t.Log("=== Step 3: Getting chapters ===")
	episodes, err := c.GetEpisodes(ctx, chainsawMan.ID)
	if err != nil {
		t.Fatalf("GetEpisodes failed: %v", err)
	}
	t.Logf("Found %d chapters", len(episodes))
	if len(episodes) == 0 {
		t.Fatal("No chapters found")
	}

	// Print first 5 chapters
	for i, ep := range episodes {
		if i >= 5 {
			break
		}
		t.Logf("  Chapter %d: %s (ID: %s)", ep.Number, ep.Title, ep.ID)
	}

	// Step 4: Get manga pages for first chapter
	t.Log("=== Step 4: Getting manga pages ===")
	pages, err := c.GetMangaPages(ctx, episodes[0].ID)
	if err != nil {
		t.Fatalf("GetMangaPages failed: %v", err)
	}
	t.Logf("Found %d pages", len(pages))
	if len(pages) == 0 {
		t.Fatal("No pages found")
	}

	// Print first page URL
	fmt.Printf("First page URL: %s\n", pages[0])

	t.Log("=== All tests passed! ===")
}
