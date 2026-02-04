package flixhq

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/justchokingaround/greg/internal/providers"
	"github.com/justchokingaround/greg/pkg/extractors"
	"github.com/justchokingaround/greg/pkg/types"
)

type FlixHQ struct {
	BaseURL     string
	Client      *http.Client
	searchCache sync.Map
	infoCache   sync.Map
}

func New() *FlixHQ {
	return &FlixHQ{
		BaseURL: "https://flixhq.to",
		Client:  &http.Client{},
	}
}

func (f *FlixHQ) Name() string {
	return "flixhq"
}

// searchOld searches for movies/shows by query (legacy internal method)
func (f *FlixHQ) searchOld(query string) (*types.SearchResults, error) {
	if cached, ok := f.searchCache.Load(query); ok {
		return cached.(*types.SearchResults), nil
	}

	// Replace non-word characters with hyphens
	re := regexp.MustCompile(`[\W_]+`)
	cleanQuery := re.ReplaceAllString(query, "-")
	searchURL := fmt.Sprintf("%s/search/%s", f.BaseURL, cleanQuery)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to mimic browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Referer", f.BaseURL)

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned status code %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	results := &types.SearchResults{
		Results: []types.SearchResult{},
	}

	// Parse search results
	doc.Find(".film_list-wrap > div.flw-item").Each(func(i int, s *goquery.Selection) {
		// Extract title
		title := strings.TrimSpace(s.Find(".film-detail .film-name a").Text())
		if title == "" {
			return
		}

		// Extract URL and ID
		href, exists := s.Find(".film-poster a").Attr("href")
		if !exists || href == "" {
			return
		}

		// ID is the href without leading slash
		id := strings.TrimPrefix(href, "/")

		// Extract image
		image, _ := s.Find(".film-poster img").Attr("data-src")

		// Extract release date from info
		releaseDate := ""
		typeStr := ""

		s.Find(".film-detail .fd-infor .fdi-item").Each(func(j int, info *goquery.Selection) {
			text := strings.TrimSpace(info.Text())

			// Check if this is the type (Movie/TV Series)
			if strings.Contains(text, "Movie") {
				typeStr = "Movie"
			} else if strings.Contains(text, "TV") {
				typeStr = "TV Series"
			}

			// Try to parse as year
			if year, err := strconv.Atoi(text); err == nil && year > 1900 && year < 2100 {
				releaseDate = text
			}
		})

		results.Results = append(results.Results, types.SearchResult{
			ID:          id,
			Title:       title,
			Image:       image,
			URL:         f.BaseURL + href,
			ReleaseDate: releaseDate,
			Type:        typeStr,
		})
	})

	f.searchCache.Store(query, results)
	return results, nil
}

// GetInfo fetches detailed info for a movie/show
func (f *FlixHQ) GetInfo(id string) (interface{}, error) {
	if cached, ok := f.infoCache.Load(id); ok {
		return cached.(*types.MovieInfo), nil
	}

	// Construct info URL
	infoURL := f.BaseURL + "/" + id
	if strings.HasPrefix(id, "/") {
		infoURL = f.BaseURL + id
	}

	req, err := http.NewRequest("GET", infoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", f.BaseURL)

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movie info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("info request returned status code %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	info := &types.MovieInfo{
		ID:       id,
		URL:      infoURL,
		Episodes: []types.Episode{},
		Genres:   []string{},
	}

	// Extract title
	info.Title = strings.TrimSpace(doc.Find(".heading-name a").First().Text())

	// Extract image
	if img, exists := doc.Find(".m_i-d-poster img").Attr("src"); exists {
		info.Image = img
	}

	// Extract description
	info.Description = strings.TrimSpace(doc.Find(".description").Text())

	// Extract type and release date from row-lines
	doc.Find(".row-line").Each(func(i int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find("strong").Text())

		switch {
		case strings.Contains(label, "Released"):
			info.ReleaseDate = strings.TrimSpace(s.Find("a").First().Text())
		case strings.Contains(label, "Genre"):
			s.Find("a").Each(func(j int, genre *goquery.Selection) {
				genreText := strings.TrimSpace(genre.Text())
				if genreText != "" {
					info.Genres = append(info.Genres, genreText)
				}
			})
		}
	})

	// Determine if it's a movie or TV series
	if strings.Contains(strings.ToLower(info.Title), "season") ||
		doc.Find("#episodes-content").Length() > 0 {
		info.Type = "TV Series"

		// For TV series, extract episodes
		episodes := f.extractEpisodes(doc, id)
		info.Episodes = episodes
	} else {
		info.Type = "Movie"

		// For movies, create a single episode entry
		if watchID, exists := doc.Find(".watch_block").Attr("data-id"); exists {
			info.Episodes = []types.Episode{
				{
					ID:     watchID,
					Number: 1,
					Title:  info.Title,
				},
			}
		}
	}

	f.infoCache.Store(id, info)
	return info, nil
}

// extractEpisodes extracts episode information from the page
func (f *FlixHQ) extractEpisodes(doc *goquery.Document, movieID string) []types.Episode {
	episodes := []types.Episode{}

	// Find all episode items
	doc.Find(".ss-list a.ssl-item.ep-item").Each(func(i int, s *goquery.Selection) {
		epNum := strings.TrimSpace(s.Find(".ssli-order").Text())
		epTitle := strings.TrimSpace(s.Find(".ssli-detail .ep-name").Text())
		epID, _ := s.Attr("data-id")

		// Parse episode number
		num := i + 1
		if parsedNum, err := strconv.Atoi(epNum); err == nil {
			num = parsedNum
		}

		episodes = append(episodes, types.Episode{
			ID:     epID,
			Number: num,
			Title:  epTitle,
		})
	})

	return episodes
}

// GetServers fetches available servers for an episode
func (f *FlixHQ) GetServers(episodeID string) ([]types.EpisodeServer, error) {
	// For movies, episodeID is actually the movie data-id
	// Try movie endpoint first: /ajax/movie/episodes/{id}
	movieServerURL := fmt.Sprintf("%s/ajax/movie/episodes/%s", f.BaseURL, episodeID)

	req, err := http.NewRequest("GET", movieServerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", f.BaseURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// If movie endpoint works, use it
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		servers, err := f.parseServersFromMovieHTML(string(body))
		if err == nil && len(servers) > 0 {
			return servers, nil
		}
	}

	// Fall back to TV series endpoint: /ajax/v2/episode/servers/{id}
	tvServerURL := fmt.Sprintf("%s/ajax/v2/episode/servers/%s", f.BaseURL, episodeID)

	req2, err := http.NewRequest("GET", tvServerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req2.Header.Set("Referer", f.BaseURL)
	req2.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp2, err := f.Client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	// Read response body
	body, err := io.ReadAll(resp2.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Try to parse as JSON first (new API format)
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(body, &jsonResponse); err == nil {
		// Check if it has an 'html' field containing the server HTML
		if htmlContent, ok := jsonResponse["html"].(string); ok && htmlContent != "" {
			return f.parseServersFromHTML(htmlContent)
		}
	}

	// Fall back to parsing the body as HTML directly (old format)
	return f.parseServersFromHTML(string(body))
}

// parseServersFromMovieHTML parses server list from movie episodes HTML
func (f *FlixHQ) parseServersFromMovieHTML(htmlContent string) ([]types.EpisodeServer, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	servers := []types.EpisodeServer{}

	// Parse movie server links (format: <a href="/watch-movie/..." title="Vidcloud">)
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		serverName, _ := s.Attr("title")
		href, exists := s.Attr("href")

		if exists && serverName != "" && href != "" {
			// Extract episode ID from href (e.g., /watch-movie/watch-inception-19764.1613445)
			// The ID after the dot is the server/episode ID
			re := regexp.MustCompile(`\.(\d+)$`)
			if matches := re.FindStringSubmatch(href); len(matches) > 1 {
				serverID := matches[1]
				servers = append(servers, types.EpisodeServer{
					Name: serverName,
					URL:  fmt.Sprintf("%s/ajax/episode/sources/%s", f.BaseURL, serverID),
				})
			}
		}
	})

	return servers, nil
}

// parseServersFromHTML parses server list from HTML content (for TV series)
func (f *FlixHQ) parseServersFromHTML(htmlContent string) ([]types.EpisodeServer, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	servers := []types.EpisodeServer{}

	// Parse server list
	doc.Find(".nav-item a").Each(func(i int, s *goquery.Selection) {
		serverName := strings.TrimSpace(s.Text())
		serverID, exists := s.Attr("data-id")

		if exists && serverName != "" {
			servers = append(servers, types.EpisodeServer{
				Name: serverName,
				URL:  fmt.Sprintf("%s/ajax/episode/sources/%s", f.BaseURL, serverID),
			})
		}
	})

	return servers, nil
}

// GetSources fetches video sources for an episode
func (f *FlixHQ) GetSources(episodeID string) (interface{}, error) {
	// Get servers first
	servers, err := f.GetServers(episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	if len(servers) == 0 {
		return &types.VideoSources{
			Sources:   []types.Source{},
			Subtitles: []types.Subtitle{},
		}, nil
	}

	// Try each server until we get valid sources
	var lastErr error
	for _, server := range servers {
		sources, err := f.extractSourcesFromServer(server)
		if err != nil {
			lastErr = err
			continue
		}

		if len(sources.Sources) > 0 {
			return sources, nil
		}
	}

	// If all servers failed, return the last error
	if lastErr != nil {
		return nil, fmt.Errorf("failed to extract sources from all servers: %w", lastErr)
	}

	return &types.VideoSources{
		Sources:   []types.Source{},
		Subtitles: []types.Subtitle{},
	}, nil
}

// extractSourcesFromServer extracts video sources from a specific server
func (f *FlixHQ) extractSourcesFromServer(server types.EpisodeServer) (*types.VideoSources, error) {
	// Make request to get the embed URL
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", f.BaseURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sources: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d for URL: %s", resp.StatusCode, server.URL)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Try to parse as JSON to get the embed URL
	var jsonResponse map[string]interface{}
	embedURL := ""

	if err := json.Unmarshal(body, &jsonResponse); err == nil {
		// Look for embed URL in different possible fields
		if link, ok := jsonResponse["link"].(string); ok {
			embedURL = link
		} else if link, ok := jsonResponse["embed"].(string); ok {
			embedURL = link
		} else if link, ok := jsonResponse["url"].(string); ok {
			embedURL = link
		}

		// If no direct link found, check for status/result pattern
		if embedURL == "" {
			// Some responses have {"status": 200, "result": {"url": "..."}}
			if result, ok := jsonResponse["result"].(map[string]interface{}); ok {
				if link, ok := result["url"].(string); ok {
					embedURL = link
				} else if link, ok := result["link"].(string); ok {
					embedURL = link
				}
			}
		}
	}

	// If we have an embed URL, extract sources from it
	if embedURL != "" {
		extractor := extractors.GetExtractor(server.Name)
		extracted, err := extractor.Extract(embedURL)
		if err != nil {
			return nil, fmt.Errorf("failed to extract from embed URL %s: %w", embedURL, err)
		}

		return extracted, nil
	}

	// If no embed URL found, return error with more context
	return nil, fmt.Errorf("no embed URL found in response from %s", server.URL)
}

// Type returns the media type this provider supports
func (f *FlixHQ) Type() providers.MediaType {
	return providers.MediaTypeMovieTV
}

// Search (new interface) searches for movies/shows by query
func (f *FlixHQ) Search(ctx context.Context, query string) ([]providers.Media, error) {
	oldResults, err := f.searchOld(query)
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

		mediaType := providers.MediaTypeMovie
		if strings.Contains(strings.ToLower(item.Type), "tv") || strings.Contains(strings.ToLower(item.Type), "series") {
			mediaType = providers.MediaTypeTV
		}

		mediaList = append(mediaList, providers.Media{
			ID:        item.ID,
			Title:     item.Title,
			Type:      mediaType,
			PosterURL: item.Image,
			Year:      year,
			Status:    item.ReleaseDate,
		})
	}
	return mediaList, nil
}

// GetTrending returns trending media
func (f *FlixHQ) GetTrending(ctx context.Context) ([]providers.Media, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetRecent returns recent media
func (f *FlixHQ) GetRecent(ctx context.Context) ([]providers.Media, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetMediaDetails fetches detailed info for a movie/show
func (f *FlixHQ) GetMediaDetails(ctx context.Context, id string) (*providers.MediaDetails, error) {
	info, err := f.GetInfo(id)
	if err != nil {
		return nil, err
	}

	movieInfo, ok := info.(*types.MovieInfo)
	if !ok {
		return nil, fmt.Errorf("unexpected info type")
	}

	mediaType := providers.MediaTypeMovie
	if movieInfo.Type == "TV Series" {
		mediaType = providers.MediaTypeTV
	}

	details := &providers.MediaDetails{
		Media: providers.Media{
			ID:        movieInfo.ID,
			Title:     movieInfo.Title,
			Type:      mediaType,
			PosterURL: movieInfo.Image,
			Synopsis:  movieInfo.Description,
			Genres:    movieInfo.Genres,
			Status:    movieInfo.ReleaseDate,
		},
	}

	// Create seasons
	if mediaType == providers.MediaTypeTV && len(movieInfo.Episodes) > 0 {
		seasonsMap := make(map[int]bool)
		for _, ep := range movieInfo.Episodes {
			season := ep.Season
			if season == 0 {
				season = 1
			}
			seasonsMap[season] = true
		}

		for sNum := range seasonsMap {
			details.Seasons = append(details.Seasons, providers.Season{
				ID:     fmt.Sprintf("%s|%d", id, sNum),
				Number: sNum,
				Title:  fmt.Sprintf("Season %d", sNum),
			})
		}
	} else {
		// Movie - single "season"
		details.Seasons = []providers.Season{{
			ID:     id,
			Number: 1,
			Title:  "Movie",
		}}
	}

	return details, nil
}

// GetSeasons returns seasons for a media
func (f *FlixHQ) GetSeasons(ctx context.Context, mediaID string) ([]providers.Season, error) {
	info, err := f.GetInfo(mediaID)
	if err != nil {
		return nil, err
	}

	movieInfo, ok := info.(*types.MovieInfo)
	if !ok {
		return nil, fmt.Errorf("unexpected info type")
	}

	if len(movieInfo.Episodes) == 0 {
		return []providers.Season{{
			ID:     mediaID,
			Number: 1,
			Title:  "Season 1",
		}}, nil
	}

	seasonsMap := make(map[int]bool)
	for _, ep := range movieInfo.Episodes {
		sNum := ep.Season
		if sNum == 0 {
			sNum = 1
		}
		seasonsMap[sNum] = true
	}

	var seasons []providers.Season
	for sNum := range seasonsMap {
		seasons = append(seasons, providers.Season{
			ID:     fmt.Sprintf("%s|%d", mediaID, sNum),
			Number: sNum,
			Title:  fmt.Sprintf("Season %d", sNum),
		})
	}

	return seasons, nil
}

// GetEpisodes returns episodes for a season
func (f *FlixHQ) GetEpisodes(ctx context.Context, seasonID string) ([]providers.Episode, error) {
	var mediaID string
	var seasonNum = 1

	if strings.Contains(seasonID, "|") {
		parts := strings.Split(seasonID, "|")
		mediaID = parts[0]
		if len(parts) > 1 {
			if n, err := strconv.Atoi(parts[1]); err == nil {
				seasonNum = n
			}
		}
	} else {
		mediaID = seasonID
	}

	info, err := f.GetInfo(mediaID)
	if err != nil {
		return nil, err
	}

	movieInfo, ok := info.(*types.MovieInfo)
	if !ok {
		return nil, fmt.Errorf("unexpected info type")
	}

	var episodes []providers.Episode
	if len(movieInfo.Episodes) == 0 && movieInfo.Type == "Movie" {
		// Single movie episode
		episodes = append(episodes, providers.Episode{
			ID:     movieInfo.ID,
			Number: 1,
			Title:  movieInfo.Title,
			Season: 1,
		})
	} else {
		for _, ep := range movieInfo.Episodes {
			epSeason := ep.Season
			if epSeason == 0 {
				epSeason = 1
			}

			if epSeason == seasonNum {
				episodes = append(episodes, providers.Episode{
					ID:     ep.ID,
					Number: ep.Number,
					Title:  ep.Title,
					Season: epSeason,
				})
			}
		}
	}

	return episodes, nil
}

// GetStreamURL fetches video stream URL for an episode
func (f *FlixHQ) GetStreamURL(ctx context.Context, episodeID string, quality providers.Quality) (*providers.StreamURL, error) {
	res, err := f.GetSources(episodeID)
	if err != nil {
		return nil, err
	}

	videoSources, ok := res.(*types.VideoSources)
	if !ok {
		return nil, fmt.Errorf("unexpected source type")
	}

	if len(videoSources.Sources) == 0 {
		return nil, fmt.Errorf("no sources found")
	}

	// Find best quality match
	var selectedSource types.Source
	found := false

	targetQuality := string(quality)
	if quality == providers.QualityAuto {
		targetQuality = "auto"
	}

	for _, src := range videoSources.Sources {
		if strings.EqualFold(src.Quality, targetQuality) {
			selectedSource = src
			found = true
			break
		}
	}

	if !found {
		selectedSource = videoSources.Sources[0]
	}

	streamType := providers.StreamTypeHLS
	if !selectedSource.IsM3U8 {
		streamType = providers.StreamTypeMP4
	}

	streamURL := &providers.StreamURL{
		URL:     selectedSource.URL,
		Quality: providers.Quality(selectedSource.Quality),
		Type:    streamType,
		Referer: selectedSource.Referer,
		Headers: map[string]string{
			"Referer": selectedSource.Referer,
		},
	}

	for _, sub := range videoSources.Subtitles {
		streamURL.Subtitles = append(streamURL.Subtitles, providers.Subtitle{
			Language: sub.Lang,
			URL:      sub.URL,
		})
	}

	return streamURL, nil
}

// GetAvailableQualities returns available video qualities
func (f *FlixHQ) GetAvailableQualities(ctx context.Context, episodeID string) ([]providers.Quality, error) {
	res, err := f.GetSources(episodeID)
	if err != nil {
		return nil, err
	}

	videoSources, ok := res.(*types.VideoSources)
	if !ok {
		return nil, fmt.Errorf("unexpected source type")
	}

	var qualities []providers.Quality
	for _, src := range videoSources.Sources {
		qualities = append(qualities, providers.Quality(src.Quality))
	}

	return qualities, nil
}

// HealthCheck checks if the provider is accessible
func (f *FlixHQ) HealthCheck(ctx context.Context) error {
	return nil
}

// GetMovieEpisodeID retrieves the episode ID for a movie
func (f *FlixHQ) GetMovieEpisodeID(ctx context.Context, mediaID string) (string, error) {
	info, err := f.GetInfo(mediaID)
	if err != nil {
		return "", err
	}
	movieInfo, ok := info.(*types.MovieInfo)
	if !ok {
		return "", fmt.Errorf("invalid info type")
	}
	if len(movieInfo.Episodes) > 0 {
		return movieInfo.Episodes[0].ID, nil
	}
	return "", fmt.Errorf("no episodes found for movie %s", mediaID)
}
