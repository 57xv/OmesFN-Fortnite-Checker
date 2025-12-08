package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	DiscordWebhookURL string
	SendAllHits       bool
	RPCEnabled        bool
	DiscordClientID   string = "1445192531511087235" // **REPLACE THIS** with your actual Discord Application ID
	FailureKeywords   = []string{
		"Your account or password is incorrect.",
		"That Microsoft account doesn't exist. Enter a different account",
		"Sign in to your Microsoft account",
		"const trackingBase=\"https://tracking.epicgames.com,https://tracking.unrealengine.com\"",
		"Please sign in with a Microsoft account or create a new account",
		"recover?mkt=",
		"account.live.com/identity/confirm?mkt",
		"Email/Confirm?mkt",
		"Help us protect your account",
		"abuse?mkt=",
	}

	Sw             = time.Now()
	CheckerRunning = false

	Cookie = "MicrosoftApplicationsTelemetryDeviceId=920e613f-effa-4c29-8f33-9b639c3b321b; MSFPC=GUID=1760ade1dcf744b88cec3dccf0c07f0d&HASH=1760&LV=202311&V=4&LU=1701108908489; mkt=ar-SA; IgnoreCAW=1; MUID=251A1E31369E6D281AED0DE737986C36; MSCC=197.33.70.230-EG; MSPBack=0; NAP=V=1.9&E=1cca&C=sD-vxVi5jYeyeMkwVA7dKII2IAq8pRAa4DmVKHoqD1M-tyafuCSd4w&W=2; ANON=A=D086BC080C843D7172138ECBFFFFFFFF&E=1d24&W=2; SDIDC=CVbyEkUg8GuRPdWN!EPGwsoa25DdTij5DNeTOr4FqnHvLfbt1MrJg5xnnJzsh!HecLu5ZypjM!sZ5TtKN5sdEd2rZ9rugezwzlcUIDU5Szgq7yMLIVdfna8dg3sFCj!kQaXy2pwx6TFwJ7ar63EdVIz*Z3I3yVzEpbDMlVRweAFmG1M54fOyH0tdFaXs5Mk*7WyS05cUa*oiyMjqGmeFcnE7wutZ2INRl6ESPNMi8l98WUFK3*IKKZgUCfuaNm8lWfbBzoWBy9F3hgwe9*QM1yi41O*rE0U0!V4SpmrIPRSGT5yKcYSEDu7TJOO1XXctcPAq21yk*MnNVrYYfibqZvnzRMvTwoNBPBKzrM6*EKQd6RKQyJrKVdEAnErMFjh*JKgS35YauzHTacSRH6ocroAYtB0eXehx5rdp2UyG5kTnd8UqA00JYvp4r1lKkX4Tv9yUb3tZ5vR7JTQLhoQpSblC4zSaT9R5AgxKW3coeXxqkz0Lbpz!7l9qEjO*SdOm*5LBfF2NZSLeXlhol**kM3DFdLVyFogVq0gl0wR52Y02; MSPSoftVis=@:@; MSPRequ=id=N&lt=1701944501&co=0; uaid=a7afddfca5ea44a8a2ee1bba76040b3c; ai_session=6FvJma4ss/5jbM3ZARR4JM|1701943445431|1701944504493; wlidperf=FR=L&ST=1701944522902; __Host-MSAAUTH=11-M.C513_BAY.0.U.CobRRcb!n3ZaE3R7CUG3Zxzz7S9vV6*He8j8ioTFhrnQhA0!EnthJ8gVyKWwD5BnEZ*6gAqKNh94mVlzaZJDD5XI5cahaiT6JvIOz1yjUEEjHui5gl5kGlFIIWknljQPcOrHfClmWwodrCLOj0nmiK4!0t8d7VdaNB5NWh5bf0t9r8NimP4yViKpZ9hF7mYSrnfdWWH73MAdL6DVAXAAxUwUdBhRQvB9BQtQOgP2r4we0Cg0pvxld3XcEX9lvnktJeC2WII2kq3ING4!WoPfpoM$; PPLState=1; MSPOK=$uuid-d9559e5d-eb3c-4862-aefb-702fdaaf8c62$uuid-d48f3872-ff6f-457e-acde-969d16a38c95$uuid-c227e203-c0b0-411f-9e65-01165bcbc281$uuid-98f882b7-0037-4de4-8f58-c8db795010f1$uuid-0454a175-8868-4a70-9822-8e509836a4ef$uuid-ce4db8a3-c655-4677-a457-c0b7ff81a02f$uuid-160e65e0-7703-4950-9154-67fd0829b36a$uuid-dd8bae77-7811-4d1e-82dc-011f340afefe; OParams=11O.DvvtyHP50fqvq59j32mzNPWlpClyJTWqtEnFckOG878YtkueiF73Cvxi6BlMgWNJNKdCyKeNbY24lUdywNUISg2RtJ5SC77Fvn4f*Vh*RWznkt!lqPvRN9zj4aHa8GF9aku7tD5if2G!InhlLvSIrUnY5UEL1A*3s!CwU1wt*FFwKuuucB0sIkjug6SL1M9oy6LDisVNTM0usbaIq3Dr2pBl90Sine1GXhAblh0LY*8vm5Ik3gZXHpGOyoMKIo7RVtKqjzW8h1Er5dVp3JPKKMBbXT461WCpU4!!np4luPty*aWvxjPgjEz1jhTBqd0pU!8Gtk6xatvjSvlWjhoLQ8WbssxgZnFb5xqPByDvbFnCtR7xyQLHDpPRKBaXFcxfpRIoea0tAUrKuoCS2YEJEVW7Rd7!z2w0kNeWeQ*Cz*i8V2DVgj36xG8Lyy7rvf9orWPMGO4!Y0GRHBil!UpRJz!pTfonpO35XFokDHLTqzwOx7JqbB!eEMOxm6RWg7rQv0!s1xiTfrpCnOzswvf84aKIX4*x!e!5w1NaXrE*IkRWiLnyWCL4kXeIRVz998WXwd9VeJYRP8B5WSdOp6KDA5p7HgJrxzsZZJrBFIJjjkpIafjj!Rl60EYYtgxxGRQYH1QlsUYiNRV24m39P7RgoXIzj3Rvi2zVFKM3UQ!KRXSN804cr0SdDWDjg8yBQRS2h5P5VkUdMBksGAO2i!vmn7BY!r2wJQCbDn7v2eGjjonkyuRwXgRKKw34ME0TEVe7N3iQdoKZQcjtDhP5zdgG98cN!r5kJqatFG94Y3UDOSkoMYQw*kHV!QYK!EY5Hzmu61oKfHOrC5Ktuav7zEeUkGkHHhWgUxVKhSLtXyZ*uYe8D23WwRLnWilblWCpUjZxwksIg!cAUSVGr*kRlv7U37ydY!9y2gMtfdsYAMdR!vJ0*D9LtUkoRF6awqsyTRFJRsHZBcBO3874czjjgHOdHwBSGAgii5UEet!IgcBD!c4lKR5AGNQVKN!ndSCBGqtJKkMhh4mvpXSXjmAbA3ozLW6NcHcmTN5Vp!VQIWTbt9Y5SGWXbmajaxw60n1nCzxSztzFX!P!vrs9XEyKBWt5**r6iIw82PN5EDm5m7l19kpQBptXjmrbvmDZcRyY8k1krKI!FTyKY*t6Q7K1EpBQDDZFVxMOkRnCWCp*IVLa64y43Fak8tN1jmimudCCn4GDGig!luwlx6c!eNcnZazT*XuiHoIOUUMOHDdutNmRjUWNyE3Wuv*AtuMG1AF3361YFiY97ASELBaZpqQerCDayTYagBi4oxCVLQAM1!5oIHBe8v2LsxMTFXPMTSaMg5IzBOL9Uf6g8gw2dY8qKusNAnKzCKRS*qUkG5TmonufMI1dK4NlGz0mDka0kgfqclFCAmoUEx1ERI*jvTP*euv*rZ2P4mVYkd0Qlc4H77R9Htp6fKFIXb4jwoFkjkhPOW9LYFhAQymJXOk9QlQhrHJQriIT2Pr9vX0kY1xewxkbfhTgBR0XeFzTP18ToQGBuuN3GSfwesKHNyBOu3SAqVXZf4HEVcWtkcRD2pjm6VeK5wIDZPQUvQb4GSiKoreJwuR892wAxrfMAhanReoAuTiXj9Zr*2kw5kMWGhZkjCV7RiBHmk0MGHkUv!0pKezhjJNraldK2UbWlhLSWRbl2VMKTiL*aRON!0CtDQdL6ALNB8*i!VTEuUrAfs2RQp7SBZDHUQsTdv84NHjDKS0J*A1!kNt2O6X*g5cUOhOy*km0UuuDEQVsX8TIzbkMiNN0Bgwi*ed*HtP58Q1c5klOMeLifqivHoC0PpX3H2rE*xQXKKjFZEn04aReazKJBgQeNxSin!NLnyQn6Q590ZMuhmqs4o16MDpmumpCLhnPOerpG*cXbp8Q81wXNM3K8oh9whlDx7bI*3!o7zFoVobCxwx*aN9JfVLohNNrdArOtfuCwCIFlLjrgdIxhbVPKBTYkcN5CXB4a8ETpWZ87aSpRmS6DUclEf1Grxl*qm*EWxSjSpHX3whSxokpXQhtX*F*WSuGxv552RORocIaurFPTBldAXqnD3ucKuvM4zpcJ3vxSY0vBq9l01*tvwy*Gp3LhLMls0Ecy4cKGAYLNn*RE7hQZAE$"
	PsRNGCSLK = "-DiygW3nqox0vvJ7dW44rE5gtFMCs15qempbazLM7SFt8rqzFPYiz07lngjQhCSJAvR432cnbv6uaSwnrXQRzFyhsGXlLUErzLrdZpblzzJQawycvgHoIN2D6CUMD9qwoIgRvIcvH3ARmKp1m44JQ6VmC6jLndxQadyaLe8Tb!ZLz59Te6lw6PshEEM54ry8FL2VM6aH5HPUv94uacHz!qunRagNYaNJax7vItu5KjQ"
	Ppft      = "-DjzN1eKq4VUaibJxOt7gxnW7oAY0R7jEm4DZ2KO3NyQh!VlvUxESE5N3*8O*fHxztUSA7UxqAc*jZ*hb9kvQ2F!iENLKBr0YC3T7a5RxFF7xUXJ7SyhDPND0W3rT1l7jl3pbUIO5v1LpacgUeHVyIRaVxaGUg*bQJSGeVs10gpBZx3SPwGatPXcPCofS!R7P0Q$$"

	Twofa          int64
	MsHits         int64
	Hits           int64
	Bad            int64
	Check          int64
	Frees          int64
	Flagged        int64
	ToCheck        int64
	Retries        int64
	Perrors        int64
	Custom         int64
	Cpm            int64
	Rares          int64
	Headless       int64
	JobsInProgress int64

	ThreadCount int  = 0
	DebugMode   bool = false
	ProxylessMaxThreads int = 10

	Username    string = ""
	LicenseKey  string = ""
	InboxWord   string = ""
	ProxyType   string = ""
	LeftDays    string = ""
	Level       string = ""
	Mode        string = "1"
	UiMode      string = "1"
	RaresList   string = "Black Knight,Excl Neo Versa,Sparkle Specialist,Blue Squire,The Reaper,Galaxy,IKONIK,Glow,Royale Bomber,Travis Scott,Astro Jack,Wonder,Wildcat,Chun-Li,Rose Team Leader,Eon,Dark Skully,Rogue Spider Knight,Omega,Blitz,Havoc,John Wick,Blue Striker,Prodigy,Blue Team Leader,Royal Knight,Stealth Reflex,Sub Commander,Huntmaster Saber,Royale Knight,World Cup,Rogue Agent,Elite Agent,Strong Guard,Warpaint,Eddie Brock,Master Chief,Fresh,Reflex"
	OGRaresList string = "OG Skull Trooper,OG Ghoul Trooper,OG Aerial Assault Trooper,OG Renegade Raider"

	Combos = make(chan string, 10000)
	Ccombos    []string
	UseProxies = true
	Modules    []func(string) bool

	IsInBox                = false
	ExportLock             sync.Mutex
	FailureReasonsMutex    sync.Mutex
	WorkWg                 sync.WaitGroup
	FailureReasons         []string
	EpicTwofa              int64
	Sfa                    int64
	Stw                    int64
	CurrentAccountHeadless = false

	ZeroSkin            int64
	OnePlus             int64
	TenPlus             int64
	TwentyFivePlus      int64
	FiftyPlus           int64
	HundredPlus         int64
	HundredFiftyPlus    int64
	TwoHundredPlus      int64
	TwoHundredFiftyPlus int64
	ThreeHundredPlus    int64

	Version       = 1.0
	SkinTransList = make(map[string]string)
	TypesHit      = 1
	// HeadlessKw    = []string{"errors.com.epicgames.accountportal.account_headless"}
	HeadlessKw    = []string{}
	Mapping       = make(map[string]string)

	// AsciiArt = []string{"¨OmesFN¨"}
	AsciiArt = []string{}
)

func UsernameLine() string {
	return fmt.Sprintf("Current User: [%s]", Username)
}

func SubscriptionLine() string {
	return fmt.Sprintf("Subscription Ends In: [%s Days]", LeftDays)
}

func ExtractValues(obj interface{}, key string) []string {
	var result []string

	switch val := obj.(type) {
	case map[string]interface{}:
		for k, v := range val {
			if k == key {
				result = append(result, fmt.Sprintf("%v", v))
			}
			result = append(result, ExtractValues(v, key)...)
		}
	case []interface{}:
		for _, item := range val {
			result = append(result, ExtractValues(item, key)...)
		}
	}
	return result
}

func CountOccurrences(text, subString string) int {
	return strings.Count(text, subString)
}

func Parse(source, left, right string) string {
	parts := strings.SplitN(source, left, 2)
	if len(parts) > 1 {
		return strings.SplitN(parts[1], right, 2)[0]
	}
	return ""
}

func buildLrPattern(left, right string) string {
	leftPattern := "^"
	if left != "" {
		leftPattern = regexp.QuoteMeta(left)
	}
	rightPattern := "$"
	if right != "" {
		rightPattern = regexp.QuoteMeta(right)
	}
	return fmt.Sprintf("(?s)%s(.+?)%s", leftPattern, rightPattern)
}

func Lr(inputStr, left, right string, recursive, useRegex bool) []string {
	var result []string

	if left == "" && right == "" {
		return []string{inputStr}
	}

	if (left != "" && !strings.Contains(inputStr, left)) || (right != "" && !strings.Contains(inputStr, right)) {
		return nil
	}

	if recursive {
		if useRegex {
			pattern := buildLrPattern(left, right)
			re, err := regexp.Compile(pattern)
			if err == nil {
				matches := re.FindAllStringSubmatch(inputStr, -1)
				for _, match := range matches {
					if len(match) > 1 {
						result = append(result, match[1])
					}
				}
			}
		} else {
			partial := inputStr
			for {
				pFrom := 0
				if left != "" {
					idx := strings.Index(partial, left)
					if idx == -1 {
						break
					}
					pFrom = idx + len(left)
				}
				partial = partial[pFrom:]

				pTo := len(partial)
				if right != "" {
					idx := strings.Index(partial, right)
					if idx == -1 {
						break
					}
					pTo = idx
				}

				parsed := partial[:pTo]
				result = append(result, parsed)

				offset := pTo
				if right != "" {
					offset += len(right)
				}
				partial = partial[offset:]

				if !(left == "" || (strings.Contains(partial, left) && (right == "" || strings.Contains(partial, right)))) {
					break
				}
			}
		}
	} else {
		if useRegex {
			pattern := buildLrPattern(left, right)
			re, err := regexp.Compile(pattern)
			if err == nil {
				match := re.FindStringSubmatch(inputStr)
				if len(match) > 1 {
					result = append(result, match[1])
				}
			}
		} else {
			pFrom := 0
			if left != "" {
				idx := strings.Index(inputStr, left)
				if idx == -1 {
					return nil
				}
				pFrom = idx + len(left)
			}
			partial := inputStr[pFrom:]

			pTo := len(partial)
			if right != "" {
				idx := strings.Index(partial, right)
				if idx == -1 {
					return nil
				}
				pTo = idx
			}
			result = append(result, partial[:pTo])
		}
	}
	return result
}

func AddToTwofa(n int64) {
	atomic.AddInt64(&Twofa, n)
}

func AddToMsHits(n int64) {
	atomic.AddInt64(&MsHits, n)
}

func AddToHits(n int64) {
	atomic.AddInt64(&Hits, n)
}

func AddToBad(n int64) {
	atomic.AddInt64(&Bad, n)
}

func AddToCheck(n int64) {
	atomic.AddInt64(&Check, n)
}

func AddToFrees(n int64) {
	atomic.AddInt64(&Frees, n)
}

func AddToFlagged(n int64) {
	atomic.AddInt64(&Flagged, n)
}

func AddToToCheck(n int64) {
	atomic.AddInt64(&ToCheck, n)
}

func AddToRetries(n int64) {
	atomic.AddInt64(&Retries, n)
}

func AddToPerrors(n int64) {
	atomic.AddInt64(&Perrors, n)
}

func AddToCustom(n int64) {
	atomic.AddInt64(&Custom, n)
}

func AddToCpm(n int64) {
	atomic.AddInt64(&Cpm, n)
}

func AddToRares(n int64) {
	atomic.AddInt64(&Rares, n)
}

func AddToHeadless(n int64) {
	atomic.AddInt64(&Headless, n)
}

func DecrementJobs(n int64) {
	atomic.AddInt64(&JobsInProgress, -n)
}

func AddToZeroSkin(n int64) {
	atomic.AddInt64(&ZeroSkin, n)
}

func AddToOnePlus(n int64) {
	atomic.AddInt64(&OnePlus, n)
}

func AddToTenPlus(n int64) {
	atomic.AddInt64(&TenPlus, n)
}

func AddToFiftyPlus(n int64) {
	atomic.AddInt64(&FiftyPlus, n)
}

func AddToHundredPlus(n int64) {
	atomic.AddInt64(&HundredPlus, n)
}

func AddToTwoHundredPlus(n int64) {
	atomic.AddInt64(&TwoHundredPlus, n)
}

func AddToThreeHundredPlus(n int64) {
	atomic.AddInt64(&ThreeHundredPlus, n)
}

func AddToEpicTwofa(n int64) {
	atomic.AddInt64(&EpicTwofa, n)
}

func AddToFA(n int64) {
	atomic.AddInt64(&Sfa, n)
}

func AddToNFA(n int64) {
	atomic.AddInt64(&Twofa, n)
}
