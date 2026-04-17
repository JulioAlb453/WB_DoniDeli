package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func fetchContactoChatPeer(apiBaseURL string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	if base == "" {
		return "", fmt.Errorf("URL de API no configurada en este proceso (usa API_BASE_URL o DONUTS_API_BASE_URL)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/admins/contacto-chat", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("contacto-chat HTTP %d", resp.StatusCode)
	}

	var body struct {
		PeerID string `json:"peer_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return strings.TrimSpace(body.PeerID), nil
}
