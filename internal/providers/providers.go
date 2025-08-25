package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

type Modalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
}

type Limit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

type Model struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Attachment  bool       `json:"attachment"`
	Reasoning   bool       `json:"reasoning"`
	Temperature bool       `json:"temperature"`
	ToolCall    bool       `json:"tool_call"`
	Knowledge   string     `json:"knowledge"`
	ReleaseDate string     `json:"release_date"`
	LastUpdated string     `json:"last_updated"`
	Modalities  Modalities `json:"modalities"`
	OpenWeights bool       `json:"open_weights"`
	Cost        Cost       `json:"cost"`
	Limit       Limit      `json:"limit"`
}

type Provider struct {
	ID     string           `json:"id"`
	Env    []string         `json:"env"`
	Npm    string           `json:"npm"`
	API    string           `json:"api"`
	Name   string           `json:"name"`
	Doc    string           `json:"doc"`
	Models map[string]Model `json:"models"`
}

// GetAll returns all registered providers.
func GetAll() []catwalk.Provider {
	return loadProvidersFromModelsDev()
}

// loadProviders gets all providers from https://models.dev
func loadProvidersFromModelsDev() []catwalk.Provider {
	var providers []catwalk.Provider
	resp, err := http.Get("https://models.dev/api.json")
	if err != nil {
		fmt.Println("Error fetching JSON:", err)
		return providers
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return providers
	}

	var data map[string]Provider
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return providers
	}

	for _, prov := range data {
		cp := catwalk.Provider{
			Name:        prov.Name,
			ID:          catwalk.InferenceProvider(prov.ID),
			APIEndpoint: prov.API,
			Type:        catwalk.Type(prov.ID),
		}

		var largeModel catwalk.Model
		var smallModel catwalk.Model
		maxCtx := int64(0)
		minCtx := int64(-1)

		for _, mod := range prov.Models {
			cm := catwalk.Model{
				ID:                 mod.ID,
				Name:               mod.Name,
				CostPer1MIn:        mod.Cost.Input,
				CostPer1MOut:       mod.Cost.Output,
				CostPer1MInCached:  mod.Cost.CacheRead,
				CostPer1MOutCached: mod.Cost.CacheWrite,
				ContextWindow:      int64(mod.Limit.Context),
				DefaultMaxTokens:   int64(mod.Limit.Output),
				CanReason:          mod.Reasoning,
				HasReasoningEffort: false,
				SupportsImages:     mod.Attachment,
			}
			cp.Models = append(cp.Models, cm)

			if cm.ContextWindow > maxCtx {
				maxCtx = cm.ContextWindow
				largeModel = cm
			}
			if minCtx == -1 || cm.ContextWindow < minCtx {
				minCtx = cm.ContextWindow
				smallModel = cm
			}
		}

		if largeModel.ID != "" {
			cp.DefaultLargeModelID = largeModel.ID
		}
		if smallModel.ID != "" {
			cp.DefaultSmallModelID = smallModel.ID
		}

		providers = append(providers, cp)
	}

	return providers
}
