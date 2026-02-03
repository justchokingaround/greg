--[[
    GLOBAL API DOCUMENTATION
    These functions are provided by the Go environment:

    1. http.get(url, [headers])
       - Performs a GET request.
       - Returns: table { body: string, status_code: number }

    2. html_parse(html_string)
       - Parses HTML into a queryable Selection object.
       - Selection Methods:
         :find(selector)    -> Returns new Selection
         :text()            -> Returns trimmed text
         :attr(name)        -> Returns attribute value
         :first()           -> Returns first element in selection
         :length()          -> Returns number of elements
         :each(function(index, selection) ... end) -> Iterator

    3. json_parse(json_string)
       - Converts JSON string into a Lua table.
       - Returns: table or nil

    4. extract_sources(embed_url, server_name)
       - Passes URL to Go's VidCloud or MegaCloud extractors.
       - Returns: table { 
           sources: { {url, quality, is_m3u8, referer}... }, 
           subtitles: { {url, lang}... } 
         }
--]]

local http = require("http")

-- --- 1. METADATA ---

--- Returns the display name of the provider.
--- @return string
function get_name()
	return "Example Provider"
end

--- Returns content type: "movie", "tv", or "movie_tv".
--- @return string
function get_type()
	return "movie_tv"
end

-- --- 2. SEARCH & LISTS ---

--- Searches for media.
--- @param query string: Search term
--- @return table: List of {id, title, type, poster_url, year, status}
function search(query)
	local results = {}
	-- logic: http.get -> html_parse -> table.insert
	return results
end

--- Returns trending media for home screen.
--- @return table: Same format as search
function get_trending()
	local results = {}
	-- logic...
	return results
end

-- --- 3. METADATA & STRUCTURE ---

--- Returns details for a movie/show.
--- @param id string: Media ID
--- @return table: {title, type, poster_url, synopsis, genres, status}
function get_media_details(id)
	local details = {
		title = "",
		type = "movie",
		poster_url = "",
		synopsis = "",
		genres = {},
		status = "",
	}
	-- logic...
	return details
end

--- Returns seasons for a TV show.
--- @param media_id string
--- @return table: List of {id, number}
function get_seasons(media_id)
	local seasons = {}
	-- logic...
	return seasons
end

--- Returns episodes for a season.
--- @param season_id string
--- @return table: List of {id, number, title}
function get_episodes(season_id)
	local episodes = {}
	-- logic...
	return episodes
end

-- --- 4. STREAMING ---

--- Converts episode/movie ID into a streamable URL.
--- @param id string
--- @param quality string
--- @return string: Direct .m3u8 or .mp4 URL
function get_stream_url(id, quality)
	local stream_url = ""
	-- logic: find server -> extract_sources(embed_url, "vidcloud")
	return stream_url
end

--- Returns available qualities.
function get_qualities(id)
	return { "auto", "1080p", "720p" }
end

-- --- 5. SYSTEM ---

function health_check()
	return true
end
