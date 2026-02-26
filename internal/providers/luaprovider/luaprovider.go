package luaprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/cjoudrey/gluahttp"
	"github.com/justchokingaround/greg/internal/providers"
	"github.com/justchokingaround/greg/pkg/extractors"
	"github.com/justchokingaround/greg/pkg/types"
	lua "github.com/yuin/gopher-lua"
)

type LuaProvider struct {
	mu      sync.Mutex
	L       *lua.LState
	LuaFile string
}

// --- Internal Mappers ---

func (p *LuaProvider) ToMediaSlice(v lua.LValue) []providers.Media {
	var list []providers.Media
	if tbl, ok := v.(*lua.LTable); ok {
		tbl.ForEach(func(_, val lua.LValue) {
			if m, ok := val.(*lua.LTable); ok {
				list = append(list, providers.Media{
					ID:        p.L.GetField(m, "id").String(),
					Title:     p.L.GetField(m, "title").String(),
					Type:      providers.MediaType(p.L.GetField(m, "type").String()),
					PosterURL: p.L.GetField(m, "poster_url").String(),
					Year:      int(lua.LVAsNumber(p.L.GetField(m, "year"))),
					Status:    p.L.GetField(m, "status").String(),
				})
			}
		})
	}
	return list
}

func (p *LuaProvider) TableToStringSlice(lv lua.LValue) []string {
	var slice []string
	if tbl, ok := lv.(*lua.LTable); ok {
		tbl.ForEach(func(_, v lua.LValue) {
			slice = append(slice, v.String())
		})
	}
	return slice
}

func FromGoValue(L *lua.LState, v interface{}) lua.LValue {
	switch val := v.(type) {
	case string:
		return lua.LString(val)
	case float64:
		return lua.LNumber(val)
	case bool:
		return lua.LBool(val)
	case map[string]interface{}:
		tbl := L.NewTable()
		for k, v := range val {
			L.SetField(tbl, k, FromGoValue(L, v))
		}
		return tbl
	case []interface{}:
		tbl := L.NewTable()
		for _, v := range val {
			tbl.Append(FromGoValue(L, v))
		}
		return tbl
	default:
		return lua.LNil
	}
}

// PushSelection This is for allowing plugins to access some of goquery
func PushSelection(L *lua.LState, s *goquery.Selection) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = s

	// Create or get the "selection" metatable
	mt := L.NewTypeMetatable("selection")

	// Define the methods available to Lua
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"find": func(L *lua.LState) int {
			// Get the selection from UserData
			s := L.CheckUserData(1).Value.(*goquery.Selection)
			selector := L.CheckString(2)
			// Return a NEW wrapped selection
			L.Push(PushSelection(L, s.Find(selector)))
			return 1
		},
		"text": func(L *lua.LState) int {
			ud := L.CheckUserData(1)
			if ud.Value == nil {
				L.Push(lua.LString(""))
				return 1
			}
			s := ud.Value.(*goquery.Selection)
			L.Push(lua.LString(strings.TrimSpace(s.Text())))
			return 1
		},
		"attr": func(L *lua.LState) int {
			s := L.CheckUserData(1).Value.(*goquery.Selection)
			attrName := L.CheckString(2)
			val, _ := s.Attr(attrName)
			L.Push(lua.LString(val))
			return 1
		},
		"first": func(L *lua.LState) int {
			s := L.CheckUserData(1).Value.(*goquery.Selection)
			L.Push(PushSelection(L, s.First()))
			return 1
		},
		"length": func(L *lua.LState) int {
			s := L.CheckUserData(1).Value.(*goquery.Selection)
			L.Push(lua.LNumber(s.Length()))
			return 1
		},
		"each": func(L *lua.LState) int {
			s := L.CheckUserData(1).Value.(*goquery.Selection)
			fn := L.CheckFunction(2)
			s.Each(func(i int, inner *goquery.Selection) {
				// Call the Lua function for each element
				L.Push(fn)
				L.Push(lua.LNumber(i + 1)) // Lua is 1-indexed
				L.Push(PushSelection(L, inner))
				if err := L.PCall(2, 0, nil); err != nil {
					// TODO: No clue what to do here either
					// fmt.Errorf("Error calling Lua function in .each(): %v\n", err)
				}
			})
			return 0
		},
	}))

	ud.Metatable = mt
	return ud
}

func New(LuaFile string) *LuaProvider {
	l := lua.NewState()

	l.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)

	l.SetGlobal("html_parse", l.NewFunction(func(L *lua.LState) int {
		htmlContent := L.CheckString(1)
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(PushSelection(L, doc.Selection))
		return 1
	}))

	l.SetGlobal("json_parse", l.NewFunction(func(L *lua.LState) int {
		jsonStr := L.CheckString(1)

		var data interface{}
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}

		L.Push(FromGoValue(L, data))
		return 1
	}))

	// internal/providers/lua/luaProvider.go

	l.SetGlobal("extract_sources", l.NewFunction(func(L *lua.LState) int {
		targetURL := L.CheckString(1)
		serverName := strings.ToLower(L.OptString(2, ""))

		var extracted *types.VideoSources
		var err error

		// Strict mapping for your two specific extractors
		switch {
		case strings.Contains(serverName, "vidcloud"):
			extractor := extractors.NewVidCloudExtractor()
			extracted, err = extractor.Extract(targetURL)
		case strings.Contains(serverName, "megacloud"):
			extractor := extractors.NewMegaCloudExtractor()
			extracted, err = extractor.Extract(targetURL)
		default:
			return 0 // Return nothing if it's an unsupported server
		}

		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}

		// Map Go types.VideoSources -> Lua Table
		res := L.NewTable()
		srcs := L.NewTable()
		for _, s := range extracted.Sources {
			sTbl := L.NewTable()
			L.SetField(sTbl, "url", lua.LString(s.URL))
			L.SetField(sTbl, "quality", lua.LString(s.Quality))
			L.SetField(sTbl, "is_m3u8", lua.LBool(s.IsM3U8))
			L.SetField(sTbl, "referer", lua.LString(s.Referer))
			srcs.Append(sTbl)
		}
		L.SetField(res, "sources", srcs)

		subs := L.NewTable()
		for _, t := range extracted.Subtitles {
			tTbl := L.NewTable()
			L.SetField(tTbl, "url", lua.LString(t.URL))
			L.SetField(tTbl, "lang", lua.LString(t.Lang))
			subs.Append(tTbl)
		}
		L.SetField(res, "subtitles", subs)

		L.Push(res)
		return 1
	}))

	if err := l.DoFile(LuaFile); err != nil {
		panic(fmt.Sprintf("failed to load lua file at %s: %v", LuaFile, err))
	}

	return &LuaProvider{
		L:       l,
		LuaFile: LuaFile,
	}
}

// Helper to handle the stack and calls
func (p *LuaProvider) callLua(fnName string, args ...lua.LValue) (lua.LValue, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fn := p.L.GetGlobal(fnName)
	if fn.Type() == lua.LTNil {
		return nil, fmt.Errorf("lua function %s not found", fnName)
	}

	err := p.L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, args...)
	if err != nil {
		return nil, fmt.Errorf("error calling lua, %w", err)
	}

	ret := p.L.Get(-1)
	p.L.Pop(1)
	return ret, nil
}

// --- Implementation ---

func (p *LuaProvider) Name() string {
	res, _ := p.callLua("get_name")
	if res == nil {
		return "Unknown"
	}
	return res.String()
}

func (p *LuaProvider) Type() providers.MediaType {
	res, _ := p.callLua("get_type")
	if res == nil {
		return providers.MediaType("unknown") // Return a default instead of crashing
	}
	return providers.MediaType(res.String())
}

func (p *LuaProvider) Search(ctx context.Context, query string) ([]providers.Media, error) {
	res, err := p.callLua("search", lua.LString(query))
	if err != nil {
		return nil, fmt.Errorf("error calling lua, %w", err)
	}
	return p.ToMediaSlice(res), nil
}

func (p *LuaProvider) GetMediaDetails(ctx context.Context, id string) (*providers.MediaDetails, error) {
	res, err := p.callLua("get_media_details", lua.LString(id))
	if err != nil || res == lua.LNil {
		return nil, fmt.Errorf("error calling lua, %w", err)
	}

	tbl := res.(*lua.LTable)
	return &providers.MediaDetails{
		Media: providers.Media{
			ID:        id,
			Title:     p.L.GetField(tbl, "title").String(),
			Type:      providers.MediaType(p.L.GetField(tbl, "type").String()),
			PosterURL: p.L.GetField(tbl, "poster_url").String(),
			Synopsis:  p.L.GetField(tbl, "synopsis").String(),
			Status:    p.L.GetField(tbl, "status").String(),
			Genres:    p.TableToStringSlice(p.L.GetField(tbl, "genres")),
		},
	}, nil
}

func (p *LuaProvider) GetSeasons(ctx context.Context, mediaID string) ([]providers.Season, error) {
	res, err := p.callLua("get_seasons", lua.LString(mediaID))
	if err != nil {
		return nil, fmt.Errorf("error calling lua, %w", err)
	}

	var seasons []providers.Season
	if tbl, ok := res.(*lua.LTable); ok {
		tbl.ForEach(func(_, v lua.LValue) {
			s := v.(*lua.LTable)
			seasons = append(seasons, providers.Season{
				ID:     p.L.GetField(s, "id").String(),
				Number: int(lua.LVAsNumber(p.L.GetField(s, "number"))),
			})
		})
	}
	return seasons, nil
}

func (p *LuaProvider) GetEpisodes(ctx context.Context, seasonID string) ([]providers.Episode, error) {
	res, err := p.callLua("get_episodes", lua.LString(seasonID))
	if err != nil {
		return nil, fmt.Errorf("error calling lua, %w", err)
	}

	var episodes []providers.Episode
	if tbl, ok := res.(*lua.LTable); ok {
		tbl.ForEach(func(_, v lua.LValue) {
			e := v.(*lua.LTable)
			episodes = append(episodes, providers.Episode{
				ID:     p.L.GetField(e, "id").String(),
				Number: int(lua.LVAsNumber(p.L.GetField(e, "number"))),
				Title:  p.L.GetField(e, "title").String(),
			})
		})
	}
	return episodes, nil
}

func (p *LuaProvider) GetStreamURL(ctx context.Context, episodeID string, quality providers.Quality) (*providers.StreamURL, error) {
	res, err := p.callLua("get_stream_url", lua.LString(episodeID), lua.LString(string(quality)))
	if err != nil {
		return nil, fmt.Errorf("error calling lua, %w", err)
	}
	return &providers.StreamURL{URL: res.String()}, nil
}

func (p *LuaProvider) GetAvailableQualities(ctx context.Context, episodeID string) ([]providers.Quality, error) {
	res, err := p.callLua("get_qualities", lua.LString(episodeID))
	if err != nil {
		return nil, fmt.Errorf("error calling lua, %w", err)
	}

	var qualities []providers.Quality
	if tbl, ok := res.(*lua.LTable); ok {
		tbl.ForEach(func(_, v lua.LValue) {
			qualities = append(qualities, providers.Quality(v.String()))
		})
	}
	return qualities, nil
}

func (p *LuaProvider) GetTrending(ctx context.Context) ([]providers.Media, error) {
	res, err := p.callLua("get_trending")
	if err != nil {
		return nil, fmt.Errorf("error calling lua, %w", err)
	}
	return p.ToMediaSlice(res), nil
}

func (p *LuaProvider) GetRecent(ctx context.Context) ([]providers.Media, error) {
	res, err := p.callLua("get_recent")
	if err != nil {
		return nil, fmt.Errorf("error calling lua, %w", err)
	}
	return p.ToMediaSlice(res), nil
}

func (p *LuaProvider) HealthCheck(ctx context.Context) error {
	_, err := p.callLua("health_check")
	return fmt.Errorf("error calling lua, %w", err)
}

func (p *LuaProvider) GetInfo(id string) (interface{}, error) {
	res, err := p.callLua("get_info", lua.LString(id))
	return res, fmt.Errorf("error calling lua, %w", err)
}
