package sflix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/justchokingaround/greg/internal/providers"
	"github.com/justchokingaround/greg/pkg/extractors"
	"github.com/justchokingaround/greg/pkg/types"
)

type SFlix struct {
	BaseURL     string
	Client      *http.Client
	searchCache sync.Map
	infoCache   sync.Map
}

func New() *SFlix {
	return &SFlix{
		BaseURL: "https://sflix.ps",
		Client:  &http.Client{},
	}
}

func (s *SFlix) Name() string {
	return "sflix"
}

func (s *SFlix) Type() providers.MediaType {
	return providers.MediaTypeMovieTV
}

// Search searches for movies/shows by query
func (s *SFlix) Search(ctx context.Context, query string) ([]providers.Media, error) {
	if cached, ok := s.searchCache.Load(query); ok {
		return cached.([]providers.Media), nil
	}

	// Sflix uses dashes instead of spaces in search URLs
	searchQuery := strings.ReplaceAll(query, " ", "-")
	searchURL := fmt.Sprintf("%s/search/%s", s.BaseURL, searchQuery)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", s.BaseURL)

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var results []providers.Media

	doc.Find("div.flw-item").Each(func(i int, sel *goquery.Selection) {
		title := sel.Find("h2.film-name a").Text()
		href, _ := sel.Find("h2.film-name a").Attr("href")
		image, _ := sel.Find("img").Attr("data-src")

		// Extract year
		var year int
		sel.Find(".fdi-item").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if y, err := strconv.Atoi(text); err == nil && len(text) == 4 {
				year = y
			}
		})

		if href != "" {
			// Extract ID from href (include type: movie/id or tv/id)
			parts := strings.Split(strings.TrimPrefix(href, "/"), "/")
			id := ""
			mediaType := providers.MediaTypeMovie
			if len(parts) >= 2 {
				// Include type in ID: "movie/free-inception-hd-19764" or "tv/free-stranger-things-hd-39444"
				if parts[0] == "tv" {
					mediaType = providers.MediaTypeTV
				}
				id = parts[0] + "/" + parts[1]
			} else if len(parts) == 1 {
				id = parts[0]
			}

			results = append(results, providers.Media{
				ID:        id,
				Title:     strings.TrimSpace(title),
				Type:      mediaType,
				PosterURL: image,
				Year:      year,
			})
		}
	})

	s.searchCache.Store(query, results)
	return results, nil
}

func (s *SFlix) GetMediaDetails(ctx context.Context, id string) (*providers.MediaDetails, error) {
	info, err := s.GetInfo(id)
	if err != nil {
		return nil, err
	}

	movieInfo, ok := info.(*types.MovieInfo)
	if !ok {
		return nil, fmt.Errorf("invalid info type")
	}

	mediaType := providers.MediaTypeMovie
	if movieInfo.Type == "tv" {
		mediaType = providers.MediaTypeTV
	}

	return &providers.MediaDetails{
		Media: providers.Media{
			ID:        id,
			Title:     movieInfo.Title,
			Type:      mediaType,
			PosterURL: movieInfo.Image,
			Synopsis:  movieInfo.Description,
			Genres:    movieInfo.Genres,
		},
	}, nil
}

func (s *SFlix) GetSeasons(ctx context.Context, mediaID string) ([]providers.Season, error) {
	info, err := s.GetInfo(mediaID)
	if err != nil {
		return nil, err
	}

	movieInfo, ok := info.(*types.MovieInfo)
	if !ok {
		return nil, fmt.Errorf("invalid info type")
	}

	if len(movieInfo.Episodes) == 0 {
		return []providers.Season{{
			ID:     mediaID,
			Number: 1,
			Title:  "Season 1",
		}}, nil
	}

	seasonsMap := make(map[int]bool)
	var seasons []providers.Season

	for _, ep := range movieInfo.Episodes {
		sNum := ep.Season
		if sNum == 0 {
			sNum = 1
		}
		if !seasonsMap[sNum] {
			seasonsMap[sNum] = true
			seasons = append(seasons, providers.Season{
				ID:     fmt.Sprintf("%s|%d", mediaID, sNum),
				Number: sNum,
				Title:  fmt.Sprintf("Season %d", sNum),
			})
		}
	}

	// Sort seasons
	for i := 0; i < len(seasons); i++ {
		for j := i + 1; j < len(seasons); j++ {
			if seasons[i].Number > seasons[j].Number {
				seasons[i], seasons[j] = seasons[j], seasons[i]
			}
		}
	}

	return seasons, nil
}

func (s *SFlix) GetEpisodes(ctx context.Context, seasonID string) ([]providers.Episode, error) {
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

	info, err := s.GetInfo(mediaID)
	if err != nil {
		return nil, err
	}

	movieInfo, ok := info.(*types.MovieInfo)
	if !ok {
		return nil, fmt.Errorf("invalid info type")
	}

	var episodes []providers.Episode

	for _, ep := range movieInfo.Episodes {
		epSeason := ep.Season
		if epSeason == 0 {
			epSeason = 1
		}

		if epSeason == seasonNum {
			id := ep.ID
			if ep.URL != "" && ep.URL != ep.ID {
				id = fmt.Sprintf("%s|%s", ep.ID, ep.URL)
			}

			episodes = append(episodes, providers.Episode{
				ID:     id,
				Number: ep.Number,
				Title:  ep.Title,
				Season: epSeason,
			})
		}
	}

	if len(movieInfo.Episodes) == 0 && seasonNum == 1 {
		episodes = append(episodes, providers.Episode{
			ID:     movieInfo.ID,
			Number: 1,
			Title:  movieInfo.Title,
			Season: 1,
		})
	}

	return episodes, nil
}

func (s *SFlix) GetStreamURL(ctx context.Context, episodeID string, quality providers.Quality) (*providers.StreamURL, error) {
	res, err := s.GetSources(episodeID)
	if err != nil {
		return nil, err
	}

	v, ok := res.(*types.VideoSources)
	if !ok {
		return nil, fmt.Errorf("invalid source type")
	}

	if len(v.Sources) == 0 {
		return nil, fmt.Errorf("no sources found")
	}

	var selectedSource types.Source
	found := false

	targetQuality := string(quality)
	if quality == providers.QualityAuto {
		targetQuality = "auto"
	}

	for _, src := range v.Sources {
		if strings.EqualFold(src.Quality, targetQuality) {
			selectedSource = src
			found = true
			break
		}
	}

	if !found {
		selectedSource = v.Sources[0]
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

	for _, sub := range v.Subtitles {
		streamURL.Subtitles = append(streamURL.Subtitles, providers.Subtitle{
			Language: sub.Lang,
			URL:      sub.URL,
		})
	}

	return streamURL, nil
}

func (s *SFlix) GetAvailableQualities(ctx context.Context, episodeID string) ([]providers.Quality, error) {
	res, err := s.GetSources(episodeID)
	if err != nil {
		return nil, err
	}

	var qualities []providers.Quality
	if v, ok := res.(*types.VideoSources); ok {
		for _, src := range v.Sources {
			qualities = append(qualities, providers.Quality(src.Quality))
		}
	}
	return qualities, nil
}

func (s *SFlix) GetTrending(ctx context.Context) ([]providers.Media, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SFlix) GetRecent(ctx context.Context) ([]providers.Media, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SFlix) HealthCheck(ctx context.Context) error {
	return nil
}

// GetInfo fetches detailed info for a movie/show with episodes
func (s *SFlix) GetInfo(id string) (interface{}, error) {
	if cached, ok := s.infoCache.Load(id); ok {
		return cached.(*types.MovieInfo), nil
	}

	// Determine media type from id
	var mediaType string
	var infoURL string

	if strings.HasPrefix(id, "movie/") {
		mediaType = "movie"
		infoURL = fmt.Sprintf("%s/%s", s.BaseURL, id)
	} else if strings.HasPrefix(id, "tv/") {
		mediaType = "tv"
		infoURL = fmt.Sprintf("%s/%s", s.BaseURL, id)
	} else {
		// Try movie URL first
		infoURL = fmt.Sprintf("%s/movie/%s", s.BaseURL, id)
		mediaType = "movie"
	}

	req, err := http.NewRequest("GET", infoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", s.BaseURL)

	resp, err := s.Client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Try TV show URL if movie fails
		if resp != nil {
			_ = resp.Body.Close()
		}
		infoURL = fmt.Sprintf("%s/tv/%s", s.BaseURL, id)
		mediaType = "tv"

		req, err = http.NewRequest("GET", infoURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", s.BaseURL)

		resp, err = s.Client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch info: %w", err)
		}
	}
	defer func() { _ = resp.Body.Close() }()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract clean media ID (e.g., "movie/watch-inception-19764")
	cleanMediaID := id
	if !strings.Contains(id, "/") {
		cleanMediaID = mediaType + "/" + id
	}

	info := &types.MovieInfo{
		ID:       cleanMediaID,
		URL:      infoURL,
		Episodes: []types.Episode{},
		Genres:   []string{},
		Type:     mediaType,
	}

	// Extract title
	info.Title = strings.TrimSpace(doc.Find("h2.heading-name").Text())

	// Extract image
	if img, exists := doc.Find("img.film-poster-img").Attr("src"); exists {
		info.Image = img
	}

	// Extract description
	info.Description = strings.TrimSpace(doc.Find("div.description").Text())

	// Extract rating from IMDB field (format: "IMDB: 7.5")
	ratingText := strings.TrimSpace(doc.Find("span.imdb").Text())
	if ratingText != "" {
		// Extract just the number from "IMDB: 7.5"
		ratingText = strings.TrimPrefix(ratingText, "IMDB:")
		info.Rating = strings.TrimSpace(ratingText)
	}

	// Extract release date from elements section (format: "Released: YYYY-MM-DD")
	doc.Find("div.elements .row-line").Each(func(i int, sel *goquery.Selection) {
		text := sel.Text()
		if strings.Contains(text, "Released:") {
			// Extract date after "Released: "
			parts := strings.Split(text, "Released:")
			if len(parts) > 1 {
				info.ReleaseDate = strings.TrimSpace(parts[1])
			}
		}
	})

	// Extract genres - only from the row-line that contains "Genre:"
	doc.Find("div.elements .row-line").Each(func(i int, sel *goquery.Selection) {
		text := sel.Text()
		if strings.Contains(text, "Genre:") {
			// This is the genre row, extract all genre links
			sel.Find("a").Each(func(j int, genreSel *goquery.Selection) {
				genre := strings.TrimSpace(genreSel.Text())
				if genre != "" && !strings.Contains(strings.ToLower(genre), "http") {
					info.Genres = append(info.Genres, genre)
				}
			})
		}
	})

	// Extract data-id for fetching episodes
	dataID, exists := doc.Find(".detail_page-watch").Attr("data-id")
	if !exists {
		dataID, _ = doc.Find("#watch").Attr("data-id")
	}

	// For movies, create single episode with mediaID stored in URL field
	if mediaType == "movie" && dataID != "" {
		info.Episodes = []types.Episode{
			{
				ID:     dataID,
				Number: 1,
				Title:  info.Title,
				URL:    cleanMediaID, // Store mediaID in URL field for later use
			},
		}
	} else if mediaType == "tv" && dataID != "" {
		// For TV shows, fetch episode list
		episodes, err := s.fetchEpisodeList(dataID)
		if err == nil && len(episodes) > 0 {
			// Add mediaID to each episode
			for i := range episodes {
				episodes[i].URL = cleanMediaID
			}
			info.Episodes = episodes

			// Calculate last season and episode count for that season
			lastSeason := 0
			episodeCountInLastSeason := 0
			for _, ep := range episodes {
				if ep.Season > lastSeason {
					lastSeason = ep.Season
					episodeCountInLastSeason = 1
				} else if ep.Season == lastSeason {
					episodeCountInLastSeason++
				}
			}
			info.LastSeason = lastSeason
			info.TotalEpisodesLastSeason = episodeCountInLastSeason
		}
	}

	s.infoCache.Store(id, info)
	return info, nil
}

// fetchEpisodeList fetches episodes for TV shows using the new two-step Sflix API
func (s *SFlix) fetchEpisodeList(showID string) ([]types.Episode, error) {
	// Step 1: Get all seasons
	seasonURL := fmt.Sprintf("%s/ajax/season/list/%s", s.BaseURL, showID)

	req, err := http.NewRequest("GET", seasonURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create season list request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", s.BaseURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch season list: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	seasonBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read season list response: %w", err)
	}

	// Parse seasons HTML
	seasonDoc, err := goquery.NewDocumentFromReader(strings.NewReader(string(seasonBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse season HTML: %w", err)
	}

	episodes := []types.Episode{}

	// Step 2: For each season, fetch its episodes
	seasonDoc.Find(".ss-item").Each(func(seasonIdx int, seasonSel *goquery.Selection) {
		seasonID, exists := seasonSel.Attr("data-id")
		if !exists {
			return
		}

		seasonNumber := seasonIdx + 1
		seasonText := strings.TrimSpace(seasonSel.Text())
		// Try to extract season number from text like "Season 1"
		if strings.Contains(seasonText, "Season ") {
			parts := strings.Split(seasonText, "Season ")
			if len(parts) > 1 {
				if num, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					seasonNumber = num
				}
			}
		}

		// Fetch episodes for this season
		episodeURL := fmt.Sprintf("%s/ajax/season/episodes/%s", s.BaseURL, seasonID)

		epReq, err := http.NewRequest("GET", episodeURL, nil)
		if err != nil {
			return
		}

		epReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		epReq.Header.Set("Referer", s.BaseURL)
		epReq.Header.Set("X-Requested-With", "XMLHttpRequest")

		epResp, err := s.Client.Do(epReq)
		if err != nil {
			return
		}
		defer func() { _ = epResp.Body.Close() }()

		epBody, err := io.ReadAll(epResp.Body)
		if err != nil {
			return
		}

		// Parse episode HTML
		epDoc, err := goquery.NewDocumentFromReader(strings.NewReader(string(epBody)))
		if err != nil {
			return
		}

		// Find all episodes in this season
		epDoc.Find(".eps-item").Each(func(epIdx int, epSel *goquery.Selection) {
			epID, exists := epSel.Attr("data-id")
			if !exists {
				return
			}

			// Extract episode number and title from the HTML
			epNumber := epIdx + 1
			epTitle := ""

			// Try to get episode number from the episode-number div
			epNumberText := epSel.Find(".episode-number").Text()
			epNumberText = strings.TrimSpace(strings.TrimPrefix(epNumberText, "Episode "))
			epNumberText = strings.TrimSuffix(epNumberText, ":")
			if parsedNum, err := strconv.Atoi(epNumberText); err == nil {
				epNumber = parsedNum
			}

			// Get episode title
			epTitle = strings.TrimSpace(epSel.Find(".film-name a").Text())

			episodes = append(episodes, types.Episode{
				ID:     epID,
				Number: epNumber,
				Season: seasonNumber,
				Title:  epTitle,
			})
		})
	})

	return episodes, nil
}

// GetServers fetches available servers for an episode
func (s *SFlix) GetServers(episodeID string) ([]types.EpisodeServer, error) {
	// Check if episodeID contains mediaID (format: "id|mediaID")
	var actualEpisodeID, mediaID string
	parts := strings.Split(episodeID, "|")
	if len(parts) == 2 {
		actualEpisodeID = parts[0]
		mediaID = parts[1]
	} else {
		actualEpisodeID = episodeID
	}

	return s.FetchEpisodeServersWithMediaID(actualEpisodeID, mediaID)
}

// FetchEpisodeServersWithMediaID fetches available servers with mediaID context
func (s *SFlix) FetchEpisodeServersWithMediaID(episodeID string, mediaID string) ([]types.EpisodeServer, error) {
	// Determine endpoint based on whether it's a movie or TV show
	var endpoint string
	isMovie := strings.Contains(mediaID, "movie")

	if isMovie {
		// For movies, use /ajax/episode/list/{episodeId}
		endpoint = fmt.Sprintf("%s/ajax/episode/list/%s", s.BaseURL, episodeID)
	} else {
		// For TV shows, use /ajax/episode/servers/{episodeId}
		endpoint = fmt.Sprintf("%s/ajax/episode/servers/%s", s.BaseURL, episodeID)
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create servers request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", s.BaseURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read servers response: %w", err)
	}

	// Parse the HTML content
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse servers HTML: %w", err)
	}

	servers := []types.EpisodeServer{}

	// Find all server items
	doc.Find(".ulclear > li").Each(func(i int, sel *goquery.Selection) {
		dataID, exists := sel.Find("a").Attr("data-id")
		if !exists {
			return
		}

		// Server name is in <span> tag
		serverName := strings.TrimSpace(sel.Find("a span").Text())
		if serverName == "" {
			return
		}

		// Build server URL like consumet.ts: ${baseUrl}/${mediaId}.${dataId}
		// Then replace /movie/ with /watch-movie/ or /tv/ with /watch-tv/
		var urlPattern, replacement string
		if isMovie {
			urlPattern = "/movie/"
			replacement = "/watch-movie/"
		} else {
			urlPattern = "/tv/"
			replacement = "/watch-tv/"
		}

		serverURL := fmt.Sprintf("%s/%s.%s", s.BaseURL, mediaID, dataID)
		serverURL = strings.Replace(serverURL, urlPattern, replacement, 1)

		servers = append(servers, types.EpisodeServer{
			Name: strings.ToLower(serverName),
			URL:  serverURL, // Full watch URL (for compatibility, not used in new flow)
		})

		// Store dataID in URL field for extraction (will update this in extractSourcesFromServer)
		servers[len(servers)-1].URL = dataID
	})

	return servers, nil
}

// GetSources fetches video sources for an episode
func (s *SFlix) GetSources(episodeID string) (interface{}, error) {
	// Check if episodeID contains mediaID (format: "id|mediaID")
	var actualEpisodeID, mediaID string
	parts := strings.Split(episodeID, "|")
	if len(parts) == 2 {
		actualEpisodeID = parts[0]
		mediaID = parts[1]
	} else {
		actualEpisodeID = episodeID
	}

	return s.FetchEpisodeSourcesWithMediaID(actualEpisodeID, mediaID)
}

// FetchEpisodeSourcesWithMediaID fetches video sources with mediaID context
func (s *SFlix) FetchEpisodeSourcesWithMediaID(episodeID string, mediaID string) (*types.VideoSources, error) {
	// Get available servers with mediaID
	servers, err := s.FetchEpisodeServersWithMediaID(episodeID, mediaID)
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
		sources, err := s.extractSourcesFromServer(server)
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
func (s *SFlix) extractSourcesFromServer(server types.EpisodeServer) (*types.VideoSources, error) {
	// The server.URL now contains just the dataID (server ID)
	serverID := server.URL

	// Use /ajax/episode/sources/{serverID} endpoint
	sourcesURL := fmt.Sprintf("%s/ajax/episode/sources/%s", s.BaseURL, serverID)

	req, err := http.NewRequest("GET", sourcesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create sources request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", s.BaseURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch embed URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read sources response: %w", err)
	}

	// Parse JSON response to get embed URL
	// Response format: {"link":"https://megacloud.tv/..."}
	var jsonResponse struct {
		Link string `json:"link"`
	}

	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return nil, fmt.Errorf("failed to parse sources JSON: %w", err)
	}

	if jsonResponse.Link == "" {
		return nil, fmt.Errorf("no embed link found in response")
	}

	embedURL := jsonResponse.Link

	// Use the extractor to get actual video sources
	extractor := extractors.GetExtractor(server.Name)
	extracted, err := extractor.Extract(embedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract from embed URL %s: %w", embedURL, err)
	}

	return extracted, nil
}

// GetMovieEpisodeID retrieves the episode ID for a movie
func (s *SFlix) GetMovieEpisodeID(ctx context.Context, mediaID string) (string, error) {
	info, err := s.GetInfo(mediaID)
	if err != nil {
		return "", err
	}
	movieInfo, ok := info.(*types.MovieInfo)
	if !ok {
		return "", fmt.Errorf("invalid info type")
	}
	if len(movieInfo.Episodes) > 0 {
		ep := movieInfo.Episodes[0]
		// If URL is available (it stores mediaID), append it to the ID separated by |
		if ep.URL != "" {
			return fmt.Sprintf("%s|%s", ep.ID, ep.URL), nil
		}
		return ep.ID, nil
	}
	return "", fmt.Errorf("no episodes found for movie %s", mediaID)
}
