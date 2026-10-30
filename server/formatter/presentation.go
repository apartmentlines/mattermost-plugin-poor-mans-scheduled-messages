package formatter

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

//go:embed locales/*.json
var translationFiles embed.FS

type catalog struct {
	Date     string            `json:"date"`
	Months   []string          `json:"months"`
	Messages map[string]string `json:"messages"`
}

var catalogs = loadCatalogs()

func loadCatalogs() map[string]catalog {
	entries, err := translationFiles.ReadDir("locales")
	if err != nil {
		panic(err)
	}
	result := make(map[string]catalog, len(entries))
	for _, entry := range entries {
		data, err := translationFiles.ReadFile("locales/" + entry.Name())
		if err != nil {
			panic(err)
		}
		var c catalog
		if err := json.Unmarshal(data, &c); err != nil {
			panic(err)
		}
		result[strings.TrimSuffix(entry.Name(), ".json")] = c
	}
	return result
}

// Renderer formats one response using a snapshot of the recipient's preferences.
// Its zero value uses English and the Mattermost default 12-hour clock.
type Renderer struct{ types.DisplaySettings }

// ForUser refreshes preferences at render time, including for existing jobs.
func ForUser(provider ports.DisplayPreferences, userID string) Renderer {
	if provider == nil {
		return Renderer{}
	}
	return Renderer{DisplaySettings: provider.ForUser(userID)}
}

func (r Renderer) catalog() catalog {
	locale := strings.ReplaceAll(r.Locale, "_", "-")
	for key, c := range catalogs {
		if strings.EqualFold(key, locale) {
			return c
		}
	}
	if c, ok := catalogs[strings.ToLower(strings.Split(locale, "-")[0])]; ok {
		return c
	}
	return catalogs["en"]
}

// Text translates plugin-owned labels; technical error details remain unchanged.
func (r Renderer) Text(key string, args ...any) string {
	value := r.catalog().Messages[key]
	if value == "" {
		value = catalogs["en"].Messages[key]
	}
	if len(args) == 0 {
		return value
	}
	return fmt.Sprintf(value, args...)
}

// DateTime localizes the date independently of the personal clock preference.
// The caller supplies the scheduled time in its recorded time zone.
func (r Renderer) DateTime(t time.Time) string {
	c := r.catalog()
	date := strings.ReplaceAll(t.Format(c.Date), t.Month().String(), c.Months[int(t.Month())-1])
	clock := "3:04 PM"
	if r.Use24Hour {
		clock = "15:04"
	}
	return date + " " + t.Format(clock)
}

// Destination formats raw channel metadata without translating user content.
func (r Renderer) Destination(info *ports.ChannelInfo) string {
	if info == nil || info.ChannelID == "" {
		return r.Text("unknown_channel")
	}
	if info.ChannelType == model.ChannelTypeDirect || info.ChannelType == model.ChannelTypeGroup {
		return r.Text("direct", info.ChannelLink)
	}
	return r.Text("channel", info.ChannelLink)
}
