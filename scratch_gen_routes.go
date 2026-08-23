package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

func main() {
	raw, err := os.ReadFile("backend/internal/platform/testdata/routes.txt")
	if err != nil {
		panic(err)
	}

	appPrefixes := map[string]bool{
		"DELETE /api/v1/integrations/{id}": true,
		"GET /api/v1/admin/ai/knowledge": true,
		"GET /api/v1/admin/ai/prompts": true,
		"GET /api/v1/ai/stock-forecast": true,
		"GET /api/v1/integrations/": true,
		"GET /api/v1/integrations/deliveries": true,
		"GET /api/v1/integrations/oauth/callback": true,
		"GET /api/v1/integrations/providers": true,
		"POST /api/v1/admin/ai/knowledge": true,
		"POST /api/v1/ai/chat": true,
		"POST /api/v1/ai/copilot": true,
		"POST /api/v1/ai/stt": true,
		"POST /api/v1/ai/translate": true,
		"POST /api/v1/ai/tts": true,
		"POST /api/v1/integrations/": true,
		"POST /api/v1/integrations/{id}/connect": true,
		"POST /api/v1/integrations/{id}/disconnect": true,
		"PUT /api/v1/admin/ai/prompts/{key}": true,
		"PUT /api/v1/admin/devices/staff-pin": true,
		"PUT /api/v1/integrations/{id}": true,
	}

	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[app] ") {
			coreRoute := strings.TrimPrefix(line, "[app] ")
			if appPrefixes[coreRoute] {
				out = append(out, coreRoute)
			} else {
				out = append(out, line)
			}
		} else {
			out = append(out, line)
		}
	}
	sort.Strings(out)
	result := strings.Join(out, "\n") + "\n"
	if err := os.WriteFile("backend/internal/platform/testdata/routes.txt", []byte(result), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("Updated routes.txt with %d routes\n", len(out))
}
