package api

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

const (
	MojangAPIProfileURL = "https://api.mojang.com/users/profiles/minecraft/%s"
	MojangSessionURL    = "https://sessionserver.mojang.com/session/minecraft/profile/%s?unsigned=false"
)

type MojangProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MojangSessionProperty struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Signature string `json:"signature,omitempty"`
}

type MojangSession struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Properties []MojangSessionProperty `json:"properties"`
}

type MojangPublicKeys struct {
	ProfilePropertyKeys   []map[string]string `json:"profilePropertyKeys"`
	PlayerCertificateKeys []map[string]string `json:"playerCertificateKeys"`
}

func GetCertificateKeys() ([]*rsa.PublicKey, error) {
	resp, err := http.Get("https://api.minecraftservices.com/publickeys")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Mojang public keys: %w", err)
	}
	defer resp.Body.Close()

	var rawKeys MojangPublicKeys
	if err := json.NewDecoder(resp.Body).Decode(&rawKeys); err != nil {
		return nil, fmt.Errorf("failed to decode Mojang public keys: %w", err)
	}

	var result []*rsa.PublicKey
	for _, entry := range rawKeys.PlayerCertificateKeys {
		der, err := base64.StdEncoding.DecodeString(entry["publicKey"])
		if err != nil {
			return nil, fmt.Errorf("failed to decode key: %w", err)
		}
		parsed, err := x509.ParsePKIXPublicKey(der)
		if err != nil {
			return nil, fmt.Errorf("failed to parse key: %w", err)
		}
		rsaKey, ok := parsed.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
		result = append(result, rsaKey)
	}
	return result, nil
}

func GetUserUUID(name string) (uuid.UUID, string, error) {
	resp, err := http.Get(fmt.Sprintf(MojangAPIProfileURL, name))
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to fetch UUID from Mojang API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return uuid.Nil, "", fmt.Errorf("user not found")
	}
	if resp.StatusCode != http.StatusOK {
		return uuid.Nil, "", fmt.Errorf("mojang API error: %s", resp.Status)
	}

	var profile MojangProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to decode Mojang response: %w", err)
	}

	u, err := uuid.Parse(profile.ID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid UUID from Mojang: %w", err)
	}

	return u, profile.Name, nil
}

func GetUserProfile(u uuid.UUID, signed bool) (*MojangSession, error) {
	url := fmt.Sprintf(MojangSessionURL, u.String())
	if !signed {
		url += "?unsigned=true"
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile from Session Server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, fmt.Errorf("profile not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mojang Session API error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sessionProfile MojangSession
	if err := json.Unmarshal(body, &sessionProfile); err != nil {
		return nil, fmt.Errorf("failed to decode Session response: %w", err)
	}

	return &sessionProfile, nil
}
