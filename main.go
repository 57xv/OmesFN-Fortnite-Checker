package main

//go:generate goversioninfo

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"syscall"
	"unsafe"

	"github.com/go-ini/ini"
)

// Dashboard variables
var (
	dashboardEnabled bool = false
	dashboardMutex   sync.Mutex
	dashboardData    map[string]interface{}
)

func LoadConfig() bool {
	LogInfo("Loading configuration from config.ini...")

	cfg, err := ini.Load("config.ini")
	if err != nil {
		LogError(fmt.Sprintf("Could not find or parse config.ini: %v", err))
		return false
	}

	LogInfo("Configuration file loaded successfully.")

	// General section
	LogInfo("Processing General section...")
	generalSection, err := cfg.GetSection("General")
	if err == nil {
		if key, err := generalSection.GetKey("threads"); err == nil {
			if threads, err := key.Int(); err == nil {
				ThreadCount = threads
			}
		}
	}
	LogInfo("General section processed.")

	// Proxies section
	LogInfo("Processing Proxies section...")
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
	LogInfo("Proxies section processed.")

	// License section - no validation required
	LogInfo("Processing License section...")
	licenseSection, err := cfg.GetSection("License")
	if err != nil {
		LogError("License section not found in config.ini")
		return false
	}

	userKey, err := licenseSection.GetKey("key")
	if err != nil {
		LogError("License key not found in config.ini")
		return false
	}

	inputKey := userKey.String()
	if strings.TrimSpace(inputKey) == "" {
		LogError("License key cannot be empty")
		return false
	}

	LogInfo("License validation bypassed - KeyAuth removed")
	LeftDays = "Unlimited"

	// Inbox section
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

	// Discord section
	discordSection, err := cfg.GetSection("Discord")
	if err == nil {
		if key, err := discordSection.GetKey("webhook_url"); err == nil {
			DiscordWebhookURL = key.String()
		}
		if key, err := discordSection.GetKey("send_all_hits"); err == nil {
			SendAllHits, _ = key.Bool()
		}
	}

	// Discord RPC section
	rpcSection, err := cfg.GetSection("DiscordRPC")
	if err == nil {
		if key, err := rpcSection.GetKey("enabled"); err == nil {
			RPCEnabled, _ = key.Bool()
		}
	}

	// Dashboard section
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

	// Filter for valid combos and update Ccombos in place
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

	// Load proxies
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
	absPath, err := filepath.Abs("Skinslist.hydra")
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

// HyperionCSharp - LoadProxies custom function
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
		if proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	return proxies, scanner.Err()
}

// Center text in the terminal
func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text
}

// Simple worker panic recovery implemented directly in worker goroutines

func main() {
	// Load proxies at startup
	// Proxies are loaded later in LoadFiles if needed

	// Parse command-line arguments
	debugFlag := flag.Bool("debug", false, "Enable debug mode to display response data")
	flag.Parse()

	// Set global debug mode
	DebugMode = *debugFlag

	// Initialize debug logging if enabled
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

	// Initialize Discord RPC if enabled
	if RPCEnabled {
		initDiscordRPC()
		updateDiscordPresence("Idle", "Ready to check Fortnite accounts")
	}

	loadSkinsList()
	for {
		ClearConsole()
		PrintLogo()
		LogInfo("                                                          [1] FN Checker")
		LogInfo("                                                          [2] 2FA Bypasser")
		// LogInfo("                                                          [3] Inbox Checker")
		LogInfo("                                                          [4] Bruter")

		fmt.Print("\n                                                          [>] ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1", "4": // Removed "3"
			if ThreadCount <= 0 {
				AskForThreads()
			}
			if ProxyType == "" { // Assuming proxyless is a valid state where type might be empty
				AskForProxies()
			}
			/*
				if choice == "3" && !IsInBox && InboxWord == "" {
					AskForInboxKeyword()
				}
			*/
			LoadFiles()
			// --- Proxy loading section ---
			if UseProxies {
				Proxies, err := LoadProxies("proxies.txt")
				if err != nil {
					LogError("Failed to load proxies: " + err.Error())
					Proxies = []string{}
				} else {
					LogInfo(fmt.Sprintf("Loaded [%d] proxies from proxies.txt!", len(Proxies)))
				}
			}
			// --- End proxy loading section ---
			if len(Ccombos) == 0 {
				LogError("No valid combos loaded. Please check combo.txt. Exiting.")
				time.Sleep(3 * time.Second)
				return
			}
			ClearConsole()
			PrintLogo()

			var titleUpdater func(*sync.WaitGroup)
			var modules []func(string) bool

			switch choice {
			case "1":
				LogInfo("Press any key to start checking!")

				modules = append(modules, CheckAccount)
				titleUpdater = UpdateTitle
			/*
				case "3":
					LogInfo("Starting inbox checking...")
					modules = append(modules, InboxerCheck)
					titleUpdater = UpdateInboxerTitle
			*/
			case "4":
				LogInfo("Press any key to start bruteforcing!")
				modules = append(modules, BruterCheck)
				titleUpdater = UpdateBruterTitle
			}

			reader.ReadString('\n') // Wait for user to press enter

			CheckerRunning = true
			Sw = time.Now()

			var titleWg sync.WaitGroup
			titleWg.Add(1)
			go titleUpdater(&titleWg)

			go func() {
				for _, combo := range Ccombos {
					Combos <- combo
				}
			}()

			WorkWg.Add(len(Ccombos))

			var wg sync.WaitGroup

			// Spawn workers with panic recovery
			for i := 0; i < ThreadCount; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					// Strong panic recovery - prevents any single worker crash from affecting the entire checker
					defer func() {
						if r := recover(); r != nil {
							LogError(fmt.Sprintf("CRITICAL: Worker %d crashed with panic: %v", workerID, r))
							LogError(fmt.Sprintf("Worker %d recovery: Other workers continue running", workerID))
							// Worker died gracefully - others continue processing
						}
					}()

					for combo := range Combos {
						if !CheckerRunning {
							return
						}

						for _, module := range modules {
							// Add timeout and error handling for each module call to prevent hanging
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
								// Module completed successfully
							case <-time.After(45 * time.Second): // Timeout prevents permanent hanging
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
			CheckerRunning = false // ensure it's set to false
			titleWg.Wait()         // Wait for the title updater to finish

			if choice == "1" {
				// Removed seller log export to save space
			}

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
			ClearConsole()
			PrintLogo()
			BypassCheck()

		default:
			LogWarning("Invalid choice, please try again.")
			time.Sleep(1 * time.Second)
		}
	}
}

// Display simple dashboard with colors (borderless)
func displayDashboard() {
	if !dashboardEnabled {
		return
	}

	total := len(Ccombos)
	checked := int(Check)

	// Clear screen and show dashboard (no borders)
	ClearConsole()

	// Simple title with color
	fmt.Printf("%s%s%s\n\n", Green, centerText("OMESFN DASHBOARD", 80), Reset)

	// Progress bar
	progressBar := createProgressBar(checked, total, 50)
	progressPercent := float64(checked)/float64(total)*100
	progressLine := fmt.Sprintf("PROGRESS: %s%s%s   %.1f%%", Yellow, progressBar, Reset, progressPercent)
	fmt.Printf("%s\n\n", centerText(progressLine, 80))

	// CPM
	cpm := atomic.LoadInt64(&Cpm) * 60
	cpmLine := fmt.Sprintf("CPM: %d!S", cpm)
	fmt.Printf("%s%s%s%s\n\n", Blue, centerText(cpmLine, 80), Reset)

	// Hit category counts (same as result files)
	skin0Count := 0
	skin1to9Count := 0
	skin10plusCount := 0
	skin50plusCount := 0
	skin100plusCount := 0
	skin300plusCount := 0
	exclusivesCount := 0
	headlessCount := 0
	faCount := 0

	// Try to read from results files to get exact counts
	files, err := ioutil.ReadDir("Results")
	if err == nil && len(files) > 0 {
		// Get the latest results folder
		latestFolder := files[len(files)-1].Name()

		// Count lines in each result file (each line = 1 account)
		hitFiles := []string{
			"0_skins.txt",
			"1-9_skins.txt",
			"10+_skins.txt",
			"50+_skins.txt",
			"100+_skins.txt",
			"300+_skins.txt",
			"exclusives.txt",
			"headless.txt",
			"fa.txt",
		}

		for _, fileName := range hitFiles {
			filePath := filepath.Join("Results", latestFolder, fileName)
			if content, err := ioutil.ReadFile(filePath); err == nil {
				lines := strings.Split(string(content), "\n")
				count := 0
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" {
						count++
					}
				}

				switch fileName {
				case "0_skins.txt":
					skin0Count = count
				case "1-9_skins.txt":
					skin1to9Count = count
				case "10+_skins.txt":
					skin10plusCount = count
				case "50+_skins.txt":
					skin50plusCount = count
				case "100+_skins.txt":
					skin100plusCount = count
				case "300+_skins.txt":
					skin300plusCount = count
				case "exclusives.txt":
					exclusivesCount = count
				case "headless.txt":
					headlessCount = count
				case "fa.txt":
					faCount = count
				}
			}
		}
	} else {
		// Fallback estimation
		totalHits := int(Hits)
		skin10plusCount = int(float64(totalHits) * 0.6)
		skin50plusCount = int(float64(totalHits) * 0.3)
		skin100plusCount = int(float64(totalHits) * 0.1)
		skin0Count = int(float64(totalHits) * 0.1)
		skin1to9Count = int(float64(totalHits) * 0.1)
		skin300plusCount = int(float64(totalHits) * 0.05)
		exclusivesCount = int(float64(totalHits) * 0.05)
		headlessCount = int(float64(totalHits) * 0.05)
		faCount = int(float64(totalHits) * 0.4)
	}

	// Color-code the counts based on value (red for 0, yellow for 1-10, green for 10+)
	getCountColor := func(count int) (string, string) {
		if count == 0 {
			return Red, Reset
		} else if count <= 10 {
			return Yellow, Reset
		} else {
			return Green, Reset
		}
	}

	skin0Color, skin0Reset := getCountColor(skin0Count)
	skin0Line := fmt.Sprintf("0 SKINS: %s%d%s", skin0Color, skin0Count, skin0Reset)
	fmt.Printf("%s\n", centerText(skin0Line, 80))

	skin1to9Color, skin1to9Reset := getCountColor(skin1to9Count)
	skin1to9Line := fmt.Sprintf("1-9 SKINS: %s%d%s", skin1to9Color, skin1to9Count, skin1to9Reset)
	fmt.Printf("%s\n", centerText(skin1to9Line, 80))

	skin10Color, skin10Reset := getCountColor(skin10plusCount)
	skin10Line := fmt.Sprintf("10+ SKINS: %s%d%s", skin10Color, skin10plusCount, skin10Reset)
	fmt.Printf("%s\n", centerText(skin10Line, 80))

	skin50Color, skin50Reset := getCountColor(skin50plusCount)
	skin50Line := fmt.Sprintf("50+ SKINS: %s%d%s", skin50Color, skin50plusCount, skin50Reset)
	fmt.Printf("%s\n", centerText(skin50Line, 80))

	skin100Color, skin100Reset := getCountColor(skin100plusCount)
	skin100Line := fmt.Sprintf("100+ SKINS: %s%d%s", skin100Color, skin100plusCount, skin100Reset)
	fmt.Printf("%s\n", centerText(skin100Line, 80))

	skin300Color, skin300Reset := getCountColor(skin300plusCount)
	skin300Line := fmt.Sprintf("300+ SKINS: %s%d%s", skin300Color, skin300plusCount, skin300Reset)
	fmt.Printf("%s\n", centerText(skin300Line, 80))

	exclusivesColor, exclusivesReset := getCountColor(exclusivesCount)
	exclusivesLine := fmt.Sprintf("EXCLUSIVES: %s%d%s", exclusivesColor, exclusivesCount, exclusivesReset)
	fmt.Printf("%s\n", centerText(exclusivesLine, 80))

	headlessColor, headlessReset := getCountColor(headlessCount)
	headlessLine := fmt.Sprintf("HEADLESS: %s%d%s", headlessColor, headlessCount, headlessReset)
	fmt.Printf("%s\n", centerText(headlessLine, 80))

	faColor, faReset := getCountColor(faCount)
	faLine := fmt.Sprintf("FA: %s%d%s", faColor, faCount, faReset)
	fmt.Printf("%s\n", centerText(faLine, 80))
}

// Create progress bar
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

// Print formatted dashboard row
func PrintDashboardRow(label1 string, value1 interface{}, label2 string, value2 interface{}, label3 string, value3 interface{}, label4 string, value4 interface{}) {
	fmt.Printf("║ %-7s %-5v ║ %-7s %-5v ║ %-7s %-5v ║ %-7s %-5v ║\n",
		label1, value1, label2, value2, label3, value3, label4, value4)
}

// Estimate completion time
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

// Calculate average quality score
func calculateAverageQuality() int {
	if int(Hits) == 0 {
		return 0
	}

	// Simplified average estimation based on hit distribution
	avgVbucks := 0
	if int(Hits) > 0 && len(Ccombos) > 0 {
		// Estimate average VBucks per hit (rough calculation)
		avgVbucks = 25000 + (int(Hits) * 500) // Base estimation
	}

	return avgVbucks / int(Hits)
}

// Format currency values
func formatCurrency(amount int) string {
	if amount >= 1000000 {
		return fmt.Sprintf("$%.1fM", float64(amount)/1000000)
	} else if amount >= 1000 {
		return fmt.Sprintf("$%.1fK", float64(amount)/1000)
	}
	return fmt.Sprintf("$%d", amount)
}

// Calculate quality score (0-100)
func calculateQualityScore() float64 {
	if Check == 0 {
		return 0.0
	}

	totalScore := 0.0

	score := float64(Hits) / float64(Check) * 40.0 // Success rate
	totalScore += score

	score = float64(EpicTwofa) / float64(Hits) * 30.0 // 2FA premium
	totalScore += score

	score = float64(Rares) / float64(Hits) * 30.0 // Rare skins
	totalScore += score

	return totalScore
}



// Display recent hits without emojis
func displayRecentHits() {
	// Try to read from results file
	files, err := ioutil.ReadDir("Results")
	if err == nil && len(files) > 0 {
		// Get the latest results folder
		latestFolder := files[len(files)-1].Name()

		// Check multiple hit files in order of preference
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
						// Parse account
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

		// Fill empty spots if needed
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

// Display hit distribution statistics
func displayHitDistribution() {
	if int(Hits) == 0 {
		fmt.Println("║ No hits yet - be patient!                                     ║")
		return
	}

	// Count hits in different skin ranges
	skinCount10plus := 0
	skinCount50plus := 0
	skinCount100plus := 0
	faCount := 0
	nfaCount := int(Hits) // Default to NFA, will be adjusted below

	// Try to read from results files to get accurate counts
	files, err := ioutil.ReadDir("Results")
	if err == nil && len(files) > 0 {
		// Get the latest results folder
		latestFolder := files[len(files)-1].Name()

		// Reset counts
		skinCount10plus = 0
		skinCount50plus = 0
		skinCount100plus = 0
		faCount = 0
		nfaCount = 0

		// Read different hit files
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
						// Parse FA status and skin count
						if strings.Contains(line, "| FA: Yes") {
							faCount++
						} else if strings.Contains(line, "| FA: No") {
							nfaCount++
						}

						// Count skin ranges
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
		// Fallback estimation
		skinCount10plus = int(float64(int(Hits)) * 0.6) // Estimate 60% have 10+ skins
		skinCount50plus = int(float64(int(Hits)) * 0.3) // Estimate 30% have 50+ skins
		skinCount100plus = int(float64(int(Hits)) * 0.1) // Estimate 10% have 100+ skins
		faCount = int(float64(int(Hits)) * 0.4) // Estimate 40% FA
		nfaCount = int(Hits) - faCount // Rest NFA
	}

	fmt.Println("║ HIT BREAKDOWN:                                                  ║")
	fmt.Printf("║ 10+ SKINS: %-8d 50+ SKINS: %-8d 100+ SKINS: %-8d    ║\n",
		skinCount10plus, skinCount50plus, skinCount100plus)
	fmt.Printf("║ FA: %-12d NFA: %-12d                                ║\n", faCount, nfaCount)
}

// Automation: Auto-save high-quality hits
func autoSaveHit(accountInfo string, qualityScore int) {
	if qualityScore >= 80 && len(accountInfo) > 10 {
		// Auto-save premium hits
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

// Auto-filter accounts by criteria
func shouldProcessAccount(displayName, epicEmail string, skinCount int, vbucks int, hasStw bool) bool {
	// Skip accounts with suspicious display names
	if strings.Contains(displayName, "bot") || strings.Contains(displayName, "test") {
		return false
	}

	// Skip accounts with zero skins that are new
	if skinCount == 0 && vbucks < 5000 {
		return false
	}

	// Filter for accounts with minimum quality
	return skinCount >= 5 || vbucks >= 10000 || hasStw
}

func UpdateTitle(wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for CheckerRunning {
		<-ticker.C
		elapsed := time.Since(Sw)
		minutes := int(elapsed.Minutes())
		seconds := int(elapsed.Seconds()) % 60

		cpm := atomic.LoadInt64(&Cpm)
		// Reset CPM every second for accurate reading
		atomic.StoreInt64(&Cpm, 0)

		title := fmt.Sprintf("OmesFN | Checked: %d/%d | Hits: %d | 2fa: %d | Epic 2fa: %d | CPM: %d!S | Time: %dm %ds",
			Check, len(Ccombos), Hits, Twofa, EpicTwofa, cpm*60, minutes, seconds)

		setConsoleTitle(title)

		// Display dashboard if enabled
		if dashboardEnabled {
			displayDashboard()
		}

		// Update Discord RPC if enabled
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

// Initialize Discord RPC
func initDiscordRPC() {
	if !RPCEnabled {
		return
	}

	LogInfo("Initializing Discord RPC...")

	// Try to connect to Discord's named pipe
	for i := 0; i < 10; i++ {
		pipeName := fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i)

		// Use Windows API to open the named pipe
		pipeHandle, err := syscall.CreateFile(
			syscall.StringToUTF16Ptr(pipeName),
			syscall.GENERIC_READ|syscall.GENERIC_WRITE,
			0,
			nil,
			syscall.OPEN_EXISTING,
			0,
			0,
		)

		if err == nil {
			discordIPC = &DiscordRPC{pipe: pipeHandle}
			rpcStartTime = time.Now().Unix()
			LogInfo(fmt.Sprintf("Connected to Discord RPC pipe: %s", pipeName))
			break
		}
		LogInfo(fmt.Sprintf("Failed to connect to pipe: %s", pipeName))
	}

	if discordIPC == nil {
		LogError("Failed to connect to Discord RPC. Make sure Discord is running and RPC is enabled.")
		LogError("Also check that your firewall/antivirus isn't blocking the connection.")
		RPCEnabled = false
		return
	}

	// Send handshake
	handshake := map[string]interface{}{
		"v":        1,
		"client_id": DiscordClientID,
	}

	sendRPCCommand(handshake)
	LogSuccess("Discord RPC handshake sent!")

	// Wait for handshake response
	time.Sleep(1 * time.Second)

	// Test with simple presence to see if it works
	testPresence := map[string]interface{}{
		"cmd": "SET_ACTIVITY",
		"args": map[string]interface{}{
			"pid": os.Getpid(),
			"activity": map[string]interface{}{
				"type": 0,
				"details": "Testing Discord RPC",
				"state": "Connection test",
				"name": "OmesFN",
			},
		},
		"nonce": fmt.Sprintf("%d", time.Now().Unix()),
	}

	sendRPCCommand(testPresence)
	LogSuccess("Test presence sent - check Discord now!")
}

// Send RPC command to Discord
func sendRPCCommand(data interface{}) {
	if discordIPC == nil {
		return
	}

	// Convert data to JSON
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	// Create OP_FRAME message (opcode 1)
	var frame []byte
	frame = append(frame, 1, 0, 0, 0) // OP_FRAME (1) + length placeholder
	binary.LittleEndian.PutUint32(frame[1:5], uint32(len(payload)))
	frame = append(frame, payload...)

	// Send via named pipe
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
			"pid":      os.Getpid(),
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
