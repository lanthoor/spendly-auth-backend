package integrity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	playintegrity "google.golang.org/api/playintegrity/v1"
)

func TestVerifyIntegrity_InvalidJSON(t *testing.T) {
	rr := doPost("/api/v1/integrity/verify", "not json")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	assertErrorBody(t, rr, "invalid JSON body")
}

func TestVerifyIntegrity_MissingToken(t *testing.T) {
	rr := doPost("/api/v1/integrity/verify", `{"packageName":"dev.lanthoor.spendly"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	assertErrorBody(t, rr, "missing token")
}

func TestVerifyIntegrity_InvalidPackage(t *testing.T) {
	rr := doPost("/api/v1/integrity/verify", `{"token":"abc","packageName":"com.evil.app"}`)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
	assertErrorBody(t, rr, "invalid package name")
}

func TestVerdictStringSlice_Nil(t *testing.T) {
	out := verdictStringSlice(nil)
	if out != nil {
		t.Errorf("expected nil, got %v", out)
	}
}

func TestVerdictStringSlice_NilVerdict(t *testing.T) {
	out := verdictStringSlice(&playintegrity.DeviceIntegrity{})
	if out != nil {
		t.Errorf("expected nil, got %v", out)
	}
}

func TestVerdictStringSlice_Dedup(t *testing.T) {
	out := verdictStringSlice(&playintegrity.DeviceIntegrity{
		DeviceRecognitionVerdict: []string{"MEETS_BASIC_INTEGRITY", "MEETS_DEVICE_INTEGRITY", "MEETS_BASIC_INTEGRITY"},
	})
	if len(out) != 2 {
		t.Errorf("expected 2 items, got %d", len(out))
	}
}

func TestAppVerdictString_Nil(t *testing.T) {
	out := appVerdictString(nil)
	if out != nil {
		t.Errorf("expected nil, got %v", out)
	}
}

func TestAppVerdictString_Valid(t *testing.T) {
	out := appVerdictString(&playintegrity.AppIntegrity{AppRecognitionVerdict: "PLAY_RECOGNIZED"})
	if out == nil || *out != "PLAY_RECOGNIZED" {
		t.Errorf("expected PLAY_RECOGNIZED, got %v", out)
	}
}

func TestAccountDetails_Nil(t *testing.T) {
	out := accountDetails(nil)
	if out != nil {
		t.Errorf("expected nil, got %v", out)
	}
}

func TestAccountDetails_Valid(t *testing.T) {
	out := accountDetails(&playintegrity.AccountDetails{AppLicensingVerdict: "LICENSED"})
	if out == nil || out.AppLicensingVerdict != "LICENSED" {
		t.Errorf("expected LICENSED, got %v", out)
	}
}

func TestResponseJSONRoundTrip(t *testing.T) {
	resp := integrityResponse{
		DeviceRecognitionVerdict: []string{"MEETS_DEVICE_INTEGRITY", "MEETS_BASIC_INTEGRITY"},
		AccountDetails:           &accountDetailsResponse{AppLicensingVerdict: "LICENSED"},
	}
	appVerdict := "PLAY_RECOGNIZED"
	resp.AppRecognitionVerdict = &appVerdict

	data, _ := json.Marshal(resp)

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	verdicts, ok := parsed["deviceRecognitionVerdict"].([]interface{})
	if !ok || len(verdicts) != 2 {
		t.Errorf("expected 2 device verdicts, got %v", parsed["deviceRecognitionVerdict"])
	}

	acct, ok := parsed["accountDetails"].(map[string]interface{})
	if !ok || acct["appLicensingVerdict"] != "LICENSED" {
		t.Errorf("expected LICENSED, got %v", parsed["accountDetails"])
	}

	av, ok := parsed["appRecognitionVerdict"].(string)
	if !ok || av != "PLAY_RECOGNIZED" {
		t.Errorf("expected PLAY_RECOGNIZED, got %v", parsed["appRecognitionVerdict"])
	}
}

func TestResponseJSONRoundTrip_Minimal(t *testing.T) {
	resp := integrityResponse{
		DeviceRecognitionVerdict: []string{},
	}

	data, _ := json.Marshal(resp)

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	verdicts, ok := parsed["deviceRecognitionVerdict"].([]interface{})
	if !ok || len(verdicts) != 0 {
		t.Errorf("expected empty verdicts, got %v", parsed["deviceRecognitionVerdict"])
	}

	if _, exists := parsed["accountDetails"]; exists {
		t.Errorf("accountDetails should be omitted when nil")
	}
	if _, exists := parsed["appRecognitionVerdict"]; exists {
		t.Errorf("appRecognitionVerdict should be omitted when nil")
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, http.StatusServiceUnavailable, "downstream failure")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}

	assertErrorBody(t, rr, "downstream failure")
}

func doPost(path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	VerifyIntegrity(rr, req)
	return rr
}

func assertErrorBody(t *testing.T, rr *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var errResp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error body: %v", err)
	}
	if errResp["error"] != expected {
		t.Errorf("expected error %q, got %q", expected, errResp["error"])
	}
}
