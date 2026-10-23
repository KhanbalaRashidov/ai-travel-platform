package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// POIInfo is a popular place suggested from the model's knowledge. It carries NO
// rating/review fields — those must come from a real source, never the model.
type POIInfo struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Desc     string `json:"desc"`
	Free     bool   `json:"free"`
	Price    string `json:"price"` // approximate "~"-marked entry fee for paid ones
}

const poiSystem = "You list the most popular, real points of interest in a city for a traveler. " +
	"For each: name, category (sight/museum/park/viewpoint/food/landmark), a short one-line desc, " +
	"free=true/false, and an APPROXIMATE entry price like \"~€15\" for paid ones (prefix ~; free ones empty). " +
	"Use only well-known REAL places. Do NOT include ratings or review counts. " +
	"Return STRICT JSON: {\"pois\":[{\"name\":\"\",\"category\":\"\",\"desc\":\"\",\"free\":true,\"price\":\"\"}]}."

// PopularPOIs asks the LLM for a city's notable places (knowledge-based), used to
// fill the POI list when the open-data (OSM) source is empty. Prices are the
// model's approximate estimates; ratings are never produced here.
func PopularPOIs(ctx context.Context, llm LLM, city, language string, limit int) ([]POIInfo, error) {
	if llm == nil {
		return nil, fmt.Errorf("poi: no llm configured")
	}
	if limit <= 0 {
		limit = 8
	}
	user := fmt.Sprintf("City: %s. List up to %d popular places.", city, limit)
	if lang := strings.TrimSpace(language); lang != "" {
		user += " Write names recognizable; desc in " + lang + "."
	}
	comp, err := completeStructured(WithMaxTokens(ctx, 2048), llm, []Message{
		{Role: RoleSystem, Content: poiSystem},
		{Role: RoleUser, Content: user},
	})
	if err != nil {
		return nil, err
	}
	var env struct {
		POIs []POIInfo `json:"pois"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(comp.Content)), &env); err != nil {
		return nil, fmt.Errorf("poi: parse: %w (raw: %q)", err, snippet(comp.Content, 160))
	}
	out := env.POIs
	for i := range out {
		out[i].Name = strings.TrimSpace(out[i].Name)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
