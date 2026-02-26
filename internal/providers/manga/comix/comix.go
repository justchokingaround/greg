package comix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/justchokingaround/greg/internal/providers"
	"github.com/justchokingaround/greg/pkg/types"
)

type Comix struct {
	BaseURL     string
	Client      *http.Client
	searchCache sync.Map
	infoCache   sync.Map
}

func New() *Comix {
	return &Comix{
		BaseURL: "https://comix.to",
		Client:  &http.Client{},
	}
}

func (c *Comix) Name() string {
	return "comix"
}

func (c *Comix) searchOld(query string) (*types.SearchResults, error) {
	if cached, ok := c.searchCache.Load(query); ok {
		return cached.(*types.SearchResults), nil
	}

	page := 1
	urlStr := fmt.Sprintf("%s/api/v2/manga?keyword=%s&limit=20&page=%d&order[relevance]=desc", c.BaseURL, url.QueryEscape(query), page)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request to %s returned status %d: %s", urlStr, resp.StatusCode, string(bodyBytes))
	}

	var searchResponse struct {
		Result struct {
			Items []struct {
				HashID      string `json:"hash_id"`
				Slug        string `json:"slug"`
				Title       string `json:"title"`
				CoverURL    string `json:"cover_url"`
				SourceURL   string `json:"source_url"`
				ReleaseDate string `json:"release_date"`
			} `json:"items"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, err
	}

	var results []types.SearchResult
	for _, item := range searchResponse.Result.Items {
		results = append(results, types.SearchResult{
			Title:       item.Title,
			ID:          fmt.Sprintf("%s::%s", item.HashID, item.Slug),
			URL:         item.SourceURL,
			Image:       item.CoverURL,
			Type:        "manga",
			ReleaseDate: item.ReleaseDate,
		})
	}

	res := &types.SearchResults{Results: results}
	c.searchCache.Store(query, res)
	return res, nil
}

func (c *Comix) GetInfo(id string) (interface{}, error) {
	if cached, ok := c.infoCache.Load(id); ok {
		return cached.(*types.MangaInfo), nil
	}

	parts := strings.Split(id, "::")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid mangaId format")
	}
	hashId := parts[0]
	slug := parts[1]

	// Get manga details from the page
	mangaUrl := fmt.Sprintf("%s/title/%s-%s", c.BaseURL, hashId, slug)
	req, err := http.NewRequest("GET", mangaUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Restore the body for goquery to read
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var jsonData string
	// Check for escaped JSON format first (Next.js data)
	escapedMarker := `\"manga\":`
	startIndex := strings.Index(string(bodyBytes), escapedMarker)

	if startIndex != -1 {
		tempStr := string(bodyBytes[startIndex:])
		endMarker := `,\"scanlationGroups\"`
		endIndex := strings.Index(tempStr, endMarker)
		if endIndex == -1 {
			return nil, fmt.Errorf("could not find manga data end in response body (escaped format)")
		}
		valueStr := tempStr[len(escapedMarker):endIndex]
		jsonData = strings.ReplaceAll(valueStr, `\"`, `"`)
		jsonData = strings.ReplaceAll(jsonData, `\\`, `\`)
	} else {
		startIndex = strings.Index(string(bodyBytes), `"manga":`)
		if startIndex == -1 {
			return nil, fmt.Errorf("could not find manga data start in response body")
		}

		tempStr := string(bodyBytes[startIndex:])
		endIndex := strings.Index(tempStr, `,"scanlationGroups"`)
		if endIndex == -1 {
			return nil, fmt.Errorf("could not find manga data end in response body")
		}

		jsonData = tempStr[len(`"manga":`):endIndex]
	}

	var mangaData struct {
		Title    string `json:"title"`
		Synopsis string `json:"synopsis"`
		Poster   struct {
			Large string `json:"large"`
		} `json:"poster"`
		Status string `json:"status"`
		Genre  []struct {
			Title string `json:"title"`
		} `json:"genre"`
	}

	if err := json.Unmarshal([]byte(jsonData), &mangaData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from manga data: %w\nRaw JSON: %s", err, jsonData)
	}

	mangaInfo := &types.MangaInfo{
		ID:          id,
		Title:       mangaData.Title,
		Description: mangaData.Synopsis,
		Image:       mangaData.Poster.Large,
		URL:         mangaUrl,
		Status:      types.MediaStatus(mangaData.Status),
	}
	for _, genre := range mangaData.Genre {
		mangaInfo.Genres = append(mangaInfo.Genres, genre.Title)
	}

	chaptersCount := 100 // Limit per request (API max is 100)

	// Get chapters
	var allChapters []struct {
		ChapterID       int         `json:"chapter_id"`
		Number          interface{} `json:"number"`
		Name            string      `json:"name"`
		IsOfficial      any         `json:"is_official"`
		ScanlationGroup struct {
			Name string `json:"name"`
		} `json:"scanlation_group"`
	}

	page := 1
	for {
		chaptersUrl := fmt.Sprintf("%s/api/v2/manga/%s/chapters?limit=%d&page=%d&order[number]=asc", c.BaseURL, hashId, chaptersCount, page)
		req, err = http.NewRequest("GET", chaptersUrl, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")

		resp, err = c.Client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()

		var chaptersResponse struct {
			Result struct {
				Items []struct {
					ChapterID       int         `json:"chapter_id"`
					Number          interface{} `json:"number"`
					Name            string      `json:"name"`
					IsOfficial      any         `json:"is_official"`
					ScanlationGroup struct {
						Name string `json:"name"`
					} `json:"scanlation_group"`
				} `json:"items"`
				Pagination struct {
					LastPage    int `json:"last_page"`
					CurrentPage int `json:"current_page"`
				} `json:"pagination"`
			} `json:"result"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&chaptersResponse); err != nil {
			return nil, err
		}

		allChapters = append(allChapters, chaptersResponse.Result.Items...)

		if chaptersResponse.Result.Pagination.CurrentPage >= chaptersResponse.Result.Pagination.LastPage {
			break
		}
		page++
	}

	// Group chapters by number to handle duplicates
	type ChapterItem struct {
		ChapterID       int
		Number          interface{}
		Name            string
		IsOfficial      bool
		ScanlationGroup string
	}
	chapterMap := make(map[string][]ChapterItem)

	for _, item := range allChapters {
		isOfficial := false
		switch v := item.IsOfficial.(type) {
		case bool:
			isOfficial = v
		case float64:
			isOfficial = v == 1
		case int:
			isOfficial = v == 1
		}

		numStr := fmt.Sprintf("%v", item.Number)
		chapterMap[numStr] = append(chapterMap[numStr], ChapterItem{
			ChapterID:       item.ChapterID,
			Number:          item.Number,
			Name:            item.Name,
			IsOfficial:      isOfficial,
			ScanlationGroup: item.ScanlationGroup.Name,
		})
	}

	// Sort chapter numbers
	var chapterNumbers []float64
	var chapterNumberStrs []string
	for numStr := range chapterMap {
		if num, err := strconv.ParseFloat(numStr, 64); err == nil {
			chapterNumbers = append(chapterNumbers, num)
		} else {
			// Handle non-numeric chapters if any (though API seems to use numbers)
			chapterNumberStrs = append(chapterNumberStrs, numStr)
		}
	}
	sort.Float64s(chapterNumbers)
	sort.Strings(chapterNumberStrs)

	// Process sorted chapters
	processChapter := func(numStr string) {
		items := chapterMap[numStr]
		var selectedItem ChapterItem

		// Strategy: Prefer Official > MangaPlus > TCB Scans > First available
		found := false

		// 1. Try Official
		for _, item := range items {
			if item.IsOfficial {
				selectedItem = item
				found = true
				break
			}
		}

		// 2. Try MangaPlus
		if !found {
			for _, item := range items {
				if strings.EqualFold(item.ScanlationGroup, "MangaPlus") {
					selectedItem = item
					found = true
					break
				}
			}
		}

		// 3. Try TCB Scans
		if !found {
			for _, item := range items {
				if strings.EqualFold(item.ScanlationGroup, "TCB Scans") {
					selectedItem = item
					found = true
					break
				}
			}
		}

		// 4. Fallback to first
		if !found && len(items) > 0 {
			selectedItem = items[0]
		}

		title := selectedItem.Name
		if title == "" {
			title = fmt.Sprintf("Chapter %v", selectedItem.Number)
		}

		mangaInfo.Chapters = append(mangaInfo.Chapters, types.MangaChapter{
			ID:     fmt.Sprintf("%s::%s::%d::%v", hashId, slug, selectedItem.ChapterID, selectedItem.Number),
			Title:  title,
			Number: fmt.Sprintf("%v", selectedItem.Number),
		})
	}

	for _, num := range chapterNumbers {
		processChapter(fmt.Sprintf("%v", num))
	}
	for _, numStr := range chapterNumberStrs {
		processChapter(numStr)
	}

	c.infoCache.Store(id, mangaInfo)
	return mangaInfo, nil
}

func (c *Comix) GetSources(episodeID string) (interface{}, error) {
	parts := strings.Split(episodeID, "::")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid chapterId format")
	}
	hashId := parts[0]
	slug := parts[1]
	chapterApiId := parts[2]
	chapterNumber := parts[3]

	targetPath := fmt.Sprintf("/title/%s-%s/%s-chapter-%s", hashId, slug, chapterApiId, chapterNumber)
	targetUrl := fmt.Sprintf("%s%s", c.BaseURL, targetPath)

	req, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("%s%s", c.BaseURL, targetPath))

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract images from HTML (Next.js data)
	escapedMarker := `\"images\":`
	startIndex := strings.Index(string(bodyBytes), escapedMarker)

	var jsonStr string

	if startIndex != -1 {
		tempStr := string(bodyBytes[startIndex:])
		// The array ends with }] (end of last object and end of array)
		// We look for the first occurrence of }]
		endMarker := `}]`
		endIndex := strings.Index(tempStr, endMarker)
		if endIndex == -1 {
			return nil, fmt.Errorf("failed to find images end in response body")
		}

		// Include the closing }]
		jsonStr = tempStr[len(escapedMarker) : endIndex+2]

		// Unescape
		jsonStr = strings.ReplaceAll(jsonStr, `\"`, `"`)
		jsonStr = strings.ReplaceAll(jsonStr, `\\`, `\`)
	} else {
		// Try unescaped format
		marker := `"images":`
		startIndex = strings.Index(string(bodyBytes), marker)
		if startIndex == -1 {
			return nil, fmt.Errorf("failed to find images in response body")
		}

		tempStr := string(bodyBytes[startIndex:])
		endMarker := `}]`
		endIndex := strings.Index(tempStr, endMarker)
		if endIndex == -1 {
			return nil, fmt.Errorf("failed to find images end in response body")
		}

		jsonStr = tempStr[len(marker) : endIndex+2]
	}

	var images []struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &images); err != nil {
		return nil, fmt.Errorf("failed to parse images JSON: %w", err)
	}

	var pages []*types.MangaPage
	for i, img := range images {
		pages = append(pages, &types.MangaPage{
			URL:   img.URL,
			Index: i,
		})
	}

	return &types.MangaPages{Pages: pages}, nil
}

func (c *Comix) GetServers(episodeID string) ([]types.EpisodeServer, error) {
	return []types.EpisodeServer{}, nil
}

// Type returns the media type this provider supports
func (c *Comix) Type() providers.MediaType {
	return providers.MediaTypeManga
}

// Search (new interface) searches for manga by query
func (c *Comix) Search(ctx context.Context, query string) ([]providers.Media, error) {
	oldResults, err := c.searchOld(query)
	if err != nil {
		return nil, err
	}

	var mediaList []providers.Media
	for _, item := range oldResults.Results {
		year := 0
		if len(item.ReleaseDate) >= 4 {
			if y, err := strconv.Atoi(item.ReleaseDate[:4]); err == nil {
				year = y
			}
		}

		mediaList = append(mediaList, providers.Media{
			ID:        item.ID,
			Title:     item.Title,
			Type:      providers.MediaTypeManga,
			PosterURL: item.Image,
			Year:      year,
			Status:    item.ReleaseDate,
		})
	}
	return mediaList, nil
}

// GetMediaDetails fetches detailed info for a manga
func (c *Comix) GetMediaDetails(ctx context.Context, id string) (*providers.MediaDetails, error) {
	info, err := c.GetInfo(id)
	if err != nil {
		return nil, err
	}

	mangaInfo, ok := info.(*types.MangaInfo)
	if !ok {
		return nil, fmt.Errorf("unexpected info type")
	}

	details := &providers.MediaDetails{
		Media: providers.Media{
			ID:        mangaInfo.ID,
			Title:     mangaInfo.Title,
			Type:      providers.MediaTypeManga,
			PosterURL: mangaInfo.Image,
			Synopsis:  mangaInfo.Description,
			Genres:    mangaInfo.Genres,
			Status:    string(mangaInfo.Status),
		},
		// Manga has one "season" containing all chapters
		Seasons: []providers.Season{{
			ID:     id,
			Number: 1,
			Title:  "Chapters",
		}},
	}

	return details, nil
}

// GetSeasons returns seasons for a manga (always single season)
func (c *Comix) GetSeasons(ctx context.Context, mediaID string) ([]providers.Season, error) {
	return []providers.Season{{
		ID:     mediaID,
		Number: 1,
		Title:  "Chapters",
	}}, nil
}

// GetEpisodes returns chapters as episodes
func (c *Comix) GetEpisodes(ctx context.Context, seasonID string) ([]providers.Episode, error) {
	info, err := c.GetInfo(seasonID)
	if err != nil {
		return nil, err
	}

	mangaInfo, ok := info.(*types.MangaInfo)
	if !ok {
		return nil, fmt.Errorf("unexpected info type")
	}

	var episodes []providers.Episode
	for i, ch := range mangaInfo.Chapters {
		epNum := i + 1
		if num, err := strconv.ParseFloat(ch.Number, 64); err == nil {
			epNum = int(num)
		}

		episodes = append(episodes, providers.Episode{
			ID:     ch.ID,
			Number: epNum,
			Title:  ch.Title,
			Season: 1,
		})
	}

	return episodes, nil
}

// GetStreamURL not applicable for manga
func (c *Comix) GetStreamURL(ctx context.Context, episodeID string, quality providers.Quality) (*providers.StreamURL, error) {
	return nil, fmt.Errorf("not applicable for manga")
}

// GetAvailableQualities not applicable for manga
func (c *Comix) GetAvailableQualities(ctx context.Context, episodeID string) ([]providers.Quality, error) {
	return nil, fmt.Errorf("not applicable for manga")
}

// GetMangaPages fetches manga pages for a chapter
func (c *Comix) GetMangaPages(ctx context.Context, chapterID string) ([]string, error) {
	res, err := c.GetSources(chapterID)
	if err != nil {
		return nil, err
	}

	mangaPages, ok := res.(*types.MangaPages)
	if !ok {
		return nil, fmt.Errorf("unexpected source type")
	}

	var pages []string
	for _, page := range mangaPages.Pages {
		pages = append(pages, page.URL)
	}

	return pages, nil
}

// GetTrending returns trending manga
func (c *Comix) GetTrending(ctx context.Context) ([]providers.Media, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetRecent returns recent manga
func (c *Comix) GetRecent(ctx context.Context) ([]providers.Media, error) {
	return nil, fmt.Errorf("not implemented")
}

// HealthCheck checks if the provider is accessible
func (c *Comix) HealthCheck(ctx context.Context) error {
	return nil
}
