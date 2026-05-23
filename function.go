package integrity

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/auth/credentials"
	"google.golang.org/api/option"
	playintegrity "google.golang.org/api/playintegrity/v1"
)

const (
	packageName        = "dev.lanthoor.spendly"
	apiBaseURL         = "https://playintegrity.googleapis.com"
	playIntegrityScope = "https://www.googleapis.com/auth/playintegrity"
	requestTimeout     = 8 * time.Second
)

func VerifyIntegrity(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	var req integrityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Token) == "" {
		writeError(w, http.StatusBadRequest, "missing token")
		return
	}
	if req.PackageName != packageName {
		writeError(w, http.StatusForbidden, "invalid package name")
		return
	}

	log.Printf("Decoding integrity token for package=%s", req.PackageName)

	payload, err := decodeIntegrityToken(ctx, req.Token)
	if err != nil {
		log.Printf("Google API error: %v", err)
		writeError(w, http.StatusBadGateway, "Google Play Integrity API error")
		return
	}

	writeJSON(w, http.StatusOK, integrityResponse{
		DeviceRecognitionVerdict: verdictStringSlice(payload.DeviceIntegrity),
		AccountDetails:           accountDetails(payload.AccountDetails),
		AppRecognitionVerdict:    appVerdictString(payload.AppIntegrity),
	})
}

func decodeIntegrityToken(ctx context.Context, token string) (*playintegrity.TokenPayloadExternal, error) {
	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		Scopes: []string{playIntegrityScope},
	})
	if err != nil {
		return nil, err
	}

	svc, err := playintegrity.NewService(ctx, option.WithEndpoint(apiBaseURL), option.WithAuthCredentials(creds))
	if err != nil {
		return nil, err
	}

	decodeReq := &playintegrity.DecodeIntegrityTokenRequest{IntegrityToken: token}
	decoded, err := svc.V1.DecodeIntegrityToken(packageName, decodeReq).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return decoded.TokenPayloadExternal, nil
}

func verdictStringSlice(di *playintegrity.DeviceIntegrity) []string {
	if di == nil || di.DeviceRecognitionVerdict == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, v := range di.DeviceRecognitionVerdict {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func appVerdictString(ai *playintegrity.AppIntegrity) *string {
	if ai == nil {
		return nil
	}
	return &ai.AppRecognitionVerdict
}

func accountDetails(ad *playintegrity.AccountDetails) *accountDetailsResponse {
	if ad == nil {
		return nil
	}
	return &accountDetailsResponse{AppLicensingVerdict: ad.AppLicensingVerdict}
}

type integrityRequest struct {
	Token       string `json:"token"`
	PackageName string `json:"packageName"`
}

type integrityResponse struct {
	DeviceRecognitionVerdict []string                `json:"deviceRecognitionVerdict"`
	AccountDetails           *accountDetailsResponse `json:"accountDetails,omitempty"`
	AppRecognitionVerdict    *string                 `json:"appRecognitionVerdict,omitempty"`
}

type accountDetailsResponse struct {
	AppLicensingVerdict string `json:"appLicensingVerdict"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
