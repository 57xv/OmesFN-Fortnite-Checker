package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

func TranslateList(skinList string) string {
	idsArray := strings.Split(skinList, "@")
	var translatedIds []string

	for _, id := range idsArray {
		trimmedID := strings.TrimSpace(id)
		if trimmedID != "" && trimmedID != "character_plasticfork" && trimmedID != "character_corkfloor" && trimmedID != "character_greatpool" {
			if val, ok := Mapping[trimmedID]; ok {
				translatedIds = append(translatedIds, val)
			}
		}
	}

	return strings.Join(translatedIds, ", ")
}

func GrabCosmetics(items string, itemType string) string {
	var totalItemsFound int
	var itemNames []string
	var translated string
	var wg sync.WaitGroup
	var mutex sync.Mutex

	itemIds := strings.Split(items, ",")

	for _, itemId := range itemIds {
		trimmedItemId := strings.TrimSpace(itemId)
		if trimmedItemId == "" {
			continue
		}

		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			resp, err := http.Get(fmt.Sprintf("https://fortnite-api.com/v2/cosmetics/br/%s", id))
			if err != nil {
				logError(err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				logError(fmt.Errorf("bad status: %s", resp.Status))
				return
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logError(err)
				return
			}

			name := ExtractNameFromResponse(string(body))
			if name != "" {
				mutex.Lock()
				totalItemsFound++
				itemNames = append(itemNames, name)
				mutex.Unlock()
			}
		}(trimmedItemId)
	}

	wg.Wait()

	translated = strings.Join(itemNames, ", ")
	return translated
}

func ExtractNameFromResponse(response string) string {
	re := regexp.MustCompile(`"name":"(.*?)"`)
	matches := re.FindStringSubmatch(response)

	if len(matches) > 1 {
		return strings.Replace(matches[1], "\\u0027", "'", -1)
	}

	return ""
}

func logError(err error) {
	fmt.Printf("Error: %s\n", err.Error())
	f, fileErr := os.OpenFile("Trans error.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if fileErr != nil {
		fmt.Printf("Failed to open log file: %s\n", fileErr.Error())
		return
	}
	defer f.Close()

	if _, writeErr := f.WriteString(err.Error() + "\n"); writeErr != nil {
		fmt.Printf("Failed to write to log file: %s\n", writeErr.Error())
	}
}
