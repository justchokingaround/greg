package extractors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/justchokingaround/greg/pkg/types"
)

// VidCloudExtractor handles extraction from VidCloud/UpCloud/AkCloud servers using decrypt.broggl.farm
type VidCloudExtractor struct {
	Client *http.Client
}

// NewVidCloudExtractor creates a new VidCloud extractor
func NewVidCloudExtractor() *VidCloudExtractor {
	return &VidCloudExtractor{
		Client: &http.Client{
			Timeout: 15 * time.Second, // Reasonable timeout for decrypt.broggl.farm
		},
	}
}

// Extract extracts video sources from VidCloud/UpCloud/AkCloud URLs using decrypt.broggl.farm or https://dec.eatmynerd.live
func (v *VidCloudExtractor) Extract(sourceURL string) (*types.VideoSources, error) {
	// Checks if decrypter is up before using, falling back to alternate decrypter
	decrypter := "https://decrypt.broggl.farm/"
	_, err := v.Client.Head(decrypter)
	if err != nil {
		decrypter = "https://dec.eatmynerd.live/"
		_, err := v.Client.Head(decrypter)
		return nil, fmt.Errorf("failed to reach both decrypters: %w", err)
	}

	// Extract referer from sourceURL
	parsedURL, err := url.Parse(sourceURL)
	referer := ""
	if err == nil && parsedURL.Scheme != "" && parsedURL.Host != "" {
		referer = fmt.Sprintf("%s://%s/", parsedURL.Scheme, parsedURL.Host)
	}

	// Construct the decrypt.broggl.farm URL
	decURL := fmt.Sprintf("%s?url=%s", decrypter, url.QueryEscape(sourceURL))

	// Make request to decrypt.broggl.farm
	req, err := http.NewRequest("GET", decURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating dec request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := v.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed fetching from %s: %w", decrypter, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned status %d", decrypter, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading dec response: %w", err)
	}

	// Parse the JSON response
	var data struct {
		Sources []struct {
			File string `json:"file"`
			Type string `json:"type"`
		} `json:"sources"`
		Tracks []struct {
			File    string `json:"file"`
			Kind    string `json:"kind"`
			Label   string `json:"label"`
			Default bool   `json:"default"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed parsing dec response: %w", err)
	}

	// Convert to our standard format
	var videoSources []types.Source
	var subtitles []types.Subtitle

	// Extract video sources
	for _, s := range data.Sources {
		isM3U8 := strings.HasSuffix(s.File, ".m3u8")

		videoSources = append(videoSources, types.Source{
			URL:     s.File,
			Quality: "auto",
			IsM3U8:  isM3U8,
			Referer: referer,
		})
	}

	// Extract subtitle tracks
	for _, t := range data.Tracks {
		if t.Kind == "captions" || t.Kind == "subtitles" {
			subtitles = append(subtitles, types.Subtitle{
				URL:  t.File,
				Lang: strings.ToLower(t.Label),
			})
		}
	}

	if len(videoSources) == 0 {
		return nil, fmt.Errorf("no video sources found in %s", decrypter)
	}

	return &types.VideoSources{
		Sources:   videoSources,
		Subtitles: subtitles,
	}, nil
}
