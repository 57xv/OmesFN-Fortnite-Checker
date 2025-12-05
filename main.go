package main

//go:generate goversioninfo

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
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
	"github.com/hugolgst/rich-go/client"
)

// Dashboard functionality removed for modern UI
var dashboardEnabled = false

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

	// Dashboard section - disabled for modern UI
	dashboardEnabled = false

	LogSuccess("Configuration and license validated successfully!")
	return true
}

func ClearConsole() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func PrintLogo() {
	fmt.Println()
	LogInfo(centerText("========================================", 80))
	LogInfo(centerText("OmesFN Fortnite Checker", 80))
	LogInfo(centerText(fmt.Sprintf("License Status: %s", LeftDays), 80))
	LogInfo(centerText("========================================", 80))
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
		updateDiscordPresence("Idle", "frozi.lol/r")
	}

	loadSkinsList()
	for {
		ClearConsole()
		PrintLogo()
		LogInfo(centerText("[1] FN Checker", 80))
		LogInfo(centerText("[2] 2FA Bypasser", 80))
		LogInfo(centerText("[3] Links", 80))
		LogInfo(centerText("[4] Bruter", 80))

		fmt.Println()
		fmt.Print(centerText("[>] ", 80))

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1", "4":
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
				LogError("No valid combos loaded. Please check combo.txt.")
				LogInfo("Press Enter to return to main menu...")
				reader.ReadString('\n')
				continue
			}

			// Discord Webhook Configuration
			LogInfo("Do you want to use webhook?")
			fmt.Print("[>] ")
			webhookChoice, _ := reader.ReadString('\n')
			webhookChoice = strings.TrimSpace(strings.ToLower(webhookChoice))

			if webhookChoice == "y" || webhookChoice == "yes" {
				if DiscordWebhookURL == "" {
					LogInfo("Enter your discord webhook URL:")
					fmt.Print("[>] ")
					webhookURL, _ := reader.ReadString('\n')
					webhookURL = strings.TrimSpace(webhookURL)
					if webhookURL != "" {
						DiscordWebhookURL = webhookURL
						SendAllHits = true
						// Save to config file
						cfg, err := ini.Load("config.ini")
						if err == nil {
							if !cfg.HasSection("Discord") {
								cfg.NewSection("Discord")
							}
							cfg.Section("Discord").Key("webhook_url").SetValue(webhookURL)
							cfg.Section("Discord").Key("send_all_hits").SetValue("true")
							cfg.SaveTo("config.ini")
						}
						LogInfo("Discord webhook configured and saved successfully!")
					} else {
						LogInfo("No webhook URL provided. Continuing without webhook.")
					}
				} else {
					LogInfo("Using saved Discord webhook from config.")
					SendAllHits = true
				}
			} else {
				DiscordWebhookURL = ""
				SendAllHits = false
				// Save the "no webhook" choice to config
				cfg, err := ini.Load("config.ini")
				if err == nil {
					if !cfg.HasSection("Discord") {
						cfg.NewSection("Discord")
					}
					cfg.Section("Discord").Key("webhook_url").SetValue("")
					cfg.Section("Discord").Key("send_all_hits").SetValue("false")
					cfg.SaveTo("config.ini")
				}
				LogInfo("Continuing without Discord webhook.")
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
							case <-time.After(60 * time.Second): // Reduced timeout to 1 minute to prevent hangs
								LogError(fmt.Sprintf("TIMEOUT: Module for combo %s took longer than 60s", combo))
								GetStats().ExportRetries(combo, "timeout", false)
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
				// Export seller logs for FN checker
				GetStats().ExportSellerLog()
			}

			LogInfo("\nAll checking completed!")
			LogInfo(fmt.Sprintf("MS: %d | Hits: %d | Epic 2FA: %d", MsHits, Hits, EpicTwofa))

			if len(FailureReasons) > 0 {
				LogInfo("\nFailure reasons:")
				for _, reason := range FailureReasons {
					LogError(reason)
				}
			}

			LogSuccess("\nPress Enter to return to main menu...")
			reader.ReadString('\n')
			continue // Return to main menu instead of exiting

		case "2":
			ClearConsole()
			PrintLogo()
			BypassCheck()

		case "3":
			LogInfo("Opening links...")
			exec.Command("cmd", "/c", "start", "https://frozi.lol/r").Run()
			time.Sleep(1 * time.Second)

		default:
			LogWarning("Invalid choice, please try again.")
			time.Sleep(1 * time.Second)
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

		title := fmt.Sprintf("OmesFN | Checked: %d/%d | Hits: %d | 2fa: %d | Epic 2fa: %d | CPM: %d | Time: %dm %ds",
			Check, len(Ccombos), Hits, Twofa, EpicTwofa, cpm*60, minutes, seconds)

		setConsoleTitle(title)

		// Display clean stats in console
		ClearConsole()
		PrintLogo()

		fmt.Println()
		LogInfo(centerText(fmt.Sprintf("Total Hits: %d", Hits), 80))
		LogInfo(centerText(fmt.Sprintf("Epic 2FA: %d", EpicTwofa), 80))
		LogInfo(centerText(fmt.Sprintf("2FA: %d", Twofa), 80))
		LogInfo(centerText(fmt.Sprintf("FA: %d", Sfa), 80))
		LogInfo(centerText(fmt.Sprintf("Headless: %d", Headless), 80))
		LogInfo(centerText(fmt.Sprintf("Rares: %d", Rares), 80))
		fmt.Println()

		LogInfo(centerText("0 Skins: "+fmt.Sprintf("%d", ZeroSkin), 80))
		LogInfo(centerText("1-9 Skins: "+fmt.Sprintf("%d", OnePlus), 80))
		LogInfo(centerText("10+ Skins: "+fmt.Sprintf("%d", TenPlus), 80))
		LogInfo(centerText("50+ Skins: "+fmt.Sprintf("%d", FiftyPlus), 80))
		LogInfo(centerText("100+ Skins: "+fmt.Sprintf("%d", HundredPlus), 80))
		LogInfo(centerText("300+ Skins: "+fmt.Sprintf("%d", ThreeHundredPlus), 80))
		fmt.Println()
		LogInfo(centerText("1K+ V-Bucks: "+fmt.Sprintf("%d", Vbucks1kPlus), 80))
		LogInfo(centerText("3K+ V-Bucks: "+fmt.Sprintf("%d", Vbucks3kPlus), 80))

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

// Initialize Discord RPC
func initDiscordRPC() {
	if !RPCEnabled {
		return
	}

	LogInfo("Initializing Discord RPC...")

	// Try to login to Discord RPC with a working client ID
	err := client.Login(DiscordClientID) // Use custom client ID for OmesFN branding
	if err != nil {
		LogError(fmt.Sprintf("Failed to login to Discord RPC: %v", err))
		LogError("Make sure Discord is running and RPC is enabled in User Settings > Activity Status")
		RPCEnabled = false
		return
	}

	LogSuccess("Discord RPC login successful!")

	// Set initial presence with minimal info first
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

	// Start a goroutine to keep the RPC connection alive and handle updates
	go func() {
		ticker := time.NewTicker(15 * time.Second) // Update every 15 seconds to keep alive
		defer ticker.Stop()

		for RPCEnabled {
			select {
			case <-ticker.C:
				// Keep the connection alive by re-setting the activity
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

func updateDiscordPresence(details, state string) {
	if !RPCEnabled {
		return
	}

	// Try with images first
	now := time.Now()
	err := client.SetActivity(client.Activity{
		State:      state,
		Details:    details,
		LargeImage: "fortnite-png-27062",
		LargeText:  "OmesFN",
		SmallImage: "checking",
		SmallText:  "Active",
		Timestamps: &client.Timestamps{
			Start: &now,
		},
	})

	if err != nil {
		// If images fail, try without images
		LogInfo("Trying Discord RPC without images...")
		err = client.SetActivity(client.Activity{
			State:   state,
			Details: details,
			Timestamps: &client.Timestamps{
				Start: &now,
			},
		})

		if err != nil {
			LogError(fmt.Sprintf("Failed to update Discord presence: %v", err))
			RPCEnabled = false
		}
	}
}

func shutdownDiscordRPC() {
	if RPCEnabled {
		client.Logout()
		RPCEnabled = false
	}
	LogInfo("Discord RPC shutdown")
}
