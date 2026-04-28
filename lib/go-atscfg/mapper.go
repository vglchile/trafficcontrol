package atscfg

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type mapperRule struct {
	DSXMLID              string
	RuleType             string
	RequestScheme        string
	RequestPort          string
	RequestPath          string //Only Valid for redirect type
	OriginScheme         string
	OriginURL            string
	OriginURLNoPlaylist  string
	OriginFQDN           string
	OriginPath           string
	OriginPathNoPlaylist string
	OriginPort           string
	Backups              []string // each element is a hostname
	InsertionRing        []string // each element is a hostname
	Insertion            bool
	BackupIsProxy        bool
	InserterIsProxy      bool
	LegacyShortFormat    bool
}

func parseMapperMode(mapperMode string) (string, bool) {
	modeFields := strings.Fields(mapperMode)
	if len(modeFields) == 0 {
		return "", false
	}
	return modeFields[0], len(modeFields) == 2 && modeFields[1] == "1"
}

func normalizeMapperRuleType(ruleType string) string {
	switch strings.ToLower(strings.TrimSpace(ruleType)) {
	case "redirect":
		return "redirect"
	default:
		return "map"
	}
}

func normalizeRequestScheme(requestScheme string) string {
	switch strings.ToLower(strings.TrimSpace(requestScheme)) {
	case "1", "http":
		return "http"
	case "2", "https":
		return "https"
	default:
		return requestScheme
	}
}

func buildOriginURLs(originScheme string, originFQDN string, originPort string, originPath string, originPathNoPlaylist string) (string, string) {
	originHost := originFQDN
	if originPort != "" {
		originHost += ":" + originPort
	}
	return originScheme + "://" + originHost + originPath, originScheme + "://" + originHost + originPathNoPlaylist + "/"
}

func BuildMapperRules(mapperMode string, mapperMap string) (map[string][]mapperRule, []string) {
	warnings := []string{}
	mapperRules := map[string][]mapperRule{}
	_, legacyShortMode := parseMapperMode(mapperMode)
	if mapperMode != "" && mapperMap != "" {
		lines := strings.Split(mapperMap, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			legacyShortFormat := legacyShortMode && len(fields) == 5
			if !legacyShortFormat && len(fields) < 3 {
				warnings = append(warnings, "mapper rule line has fewer than 3 fields, skipping: "+line)
				continue
			}
			// Fields (map): 0=unused, 1=XMLID, 2=Type, 3=RequestScheme, 4=RequestPORT, 5=OriginURL, 6=Backups(csv), 7=Insertion(bool), 8=InsertionServers(csv), 9=BackupIsProxy(bool), 10=InserterIsProxy(bool)
			// Fields (redirect): 0=unused, 1=XMLID, 2=Type, 3=RequestScheme, 4=RequestPORT, 5=RequestPath, 6=OriginURL
			// Legacy short format when mapper_mode has two fields and the second is 1: 0=unused, 1=XMLID, 2=OriginURL, 3=Backups(csv), 4=unused
			xmlid := fields[1]
			ruleType := "map"
			requestScheme := ""
			requestPort := ""
			requestPath := ""
			originURL := ""
			originURLNoPlaylist := ""
			backupsCSV := ""
			insertionCSV := ""
			insertion := false
			backupIsProxy := false
			inserterIsProxy := false

			if legacyShortFormat {
				originURL = fields[2]
				backupsCSV = fields[3]
			} else {
				ruleType = normalizeMapperRuleType(fields[2])
				if ruleType == "redirect" {
					if len(fields) < 7 {
						warnings = append(warnings, "redirect mapper rule line has fewer than 7 fields, skipping: "+line)
						continue
					}
					requestScheme = normalizeRequestScheme(fields[3])
					requestPort = fields[4]
					requestPath = fields[5]
					originURL = fields[6]
				} else {
					if len(fields) < 11 {
						warnings = append(warnings, "mapper rule line has fewer than 11 fields, skipping: "+line)
						continue
					}
					requestScheme = normalizeRequestScheme(fields[3])
					requestPort = fields[4]
					originURL = fields[5]
					backupsCSV = fields[6]
					insertionCSV = fields[8]

					parsedInsertion, err := strconv.ParseBool(fields[7])
					if err != nil {
						warnings = append(warnings, "mapper rule has invalid Insertion value '"+fields[7]+"', skipping line...")
						continue
					}
					insertion = parsedInsertion

					parsedBackupIsProxy, err := strconv.ParseBool(fields[9])
					if err != nil {
						warnings = append(warnings, "mapper rule has invalid BackupIsProxy value '"+fields[9]+"', skipping line...")
						continue
					}
					backupIsProxy = parsedBackupIsProxy

					parsedInserterIsProxy, err := strconv.ParseBool(fields[10])
					if err != nil {
						warnings = append(warnings, "mapper rule has invalid InserterIsProxy value '"+fields[10]+"', skipping line...")
						continue
					}
					inserterIsProxy = parsedInserterIsProxy
				}

				// Standard ports are omitted
				if requestPort == "80" && requestScheme == "http" {
					warnings = append(warnings, "mapper rule has RequestPort set to 80 for scheme '"+requestScheme+"', defaulting to empty")
					requestPort = ""
				} else if requestPort == "443" && requestScheme == "https" {
					warnings = append(warnings, "mapper rule has RequestPort set to 443 for scheme '"+requestScheme+"', defaulting to empty")
					requestPort = ""
				}
			}

			// Parse the Origin URL to extract FQDN and Path
			parsedOrigin, err := url.Parse(originURL)
			if err != nil || parsedOrigin.Host == "" {
				warnings = append(warnings, "mapper rule has invalid origin URL '"+originURL+"', skipping line...")
				continue
			}
			originFQDN := parsedOrigin.Hostname()
			originPath := parsedOrigin.Path
			originScheme := parsedOrigin.Scheme

			originPort := ""
			if parsedOrigin.Port() == "80" && originScheme == "http" {
				warnings = append(warnings, "mapper rule has OriginPort set to 80 for scheme '"+originScheme+"', defaulting to empty")
				originPort = ""
			} else if parsedOrigin.Port() == "443" && originScheme == "https" {
				warnings = append(warnings, "mapper rule has OriginPort set to 443 for scheme '"+originScheme+"', defaulting to empty")
				originPort = ""
			} else {
				originPort = parsedOrigin.Port()
			}

			originPortForRings := ""
			if parsedOrigin.Port() == "" && originScheme == "http" {
				originPortForRings = "80"
			} else if parsedOrigin.Port() == "" && originScheme == "https" {
				originPortForRings = "443"
			} else {
				originPortForRings = parsedOrigin.Port()
			}

			// OriginPathNoPlaylist: remove the first path segment matching *.m3u8 and everything after it
			reM3u8 := regexp.MustCompile(`/[^/]*\.m3u8.*$`)
			originPathNoPlaylist := reM3u8.ReplaceAllString(originPath, "")
			originURL, originURLNoPlaylist = buildOriginURLs(originScheme, originFQDN, originPort, originPath, originPathNoPlaylist)

			// Backups: comma-separated list, extract hostnames only
			backupParts := strings.Split(backupsCSV, ",")
			backups := make([]string, 0, len(backupParts))
			for _, b := range backupParts {
				if strings.TrimSpace(b) == "" {
					continue
				}
				backupURL, err := url.Parse(b)
				if err != nil {
					warnings = append(warnings, "mapper rule has invalid backup URL '"+b+"', skipping")
					continue
				}
				host := backupURL.Hostname()
				var port string
				if backupURL.Port() == "" && backupURL.Scheme == "http" {
					port = "80"
				} else if backupURL.Port() == "" && backupURL.Scheme == "https" {
					port = "443"
				} else {
					port = backupURL.Port()
				}
				backups = append(backups, host+":"+port)
			}

			if !backupIsProxy {
				backups = append(backups, originFQDN+":"+originPortForRings)
			}

			// InsertionRing: comma-separated list, extract hostnames only
			insertionParts := strings.Split(insertionCSV, ",")
			insertionRing := make([]string, 0, len(insertionParts))
			for _, b := range insertionParts {
				if strings.TrimSpace(b) == "" {
					continue
				}
				insertionURL, err := url.Parse(b)
				if err != nil {
					warnings = append(warnings, "mapper rule has invalid insertion URL '"+b+"', skipping")
					continue
				}
				host := insertionURL.Hostname()
				var port string
				if insertionURL.Port() == "" && insertionURL.Scheme == "http" {
					port = "80"
				} else if insertionURL.Port() == "" && insertionURL.Scheme == "https" {
					port = "443"
				} else {
					port = insertionURL.Port()
				}
				insertionRing = append(insertionRing, host+":"+port)
			}

			// if !legacyShortFormat && !inserterIsProxy {
			// 	insertionRing = append(insertionRing, originFQDN+":"+originPortForRings)
			// }

			mapperRules[xmlid] = append(mapperRules[xmlid], mapperRule{
				DSXMLID:              xmlid,
				RuleType:             ruleType,
				RequestScheme:        requestScheme,
				RequestPort:          requestPort,
				RequestPath:          requestPath,
				OriginScheme:         originScheme,
				OriginURL:            originURL,
				OriginURLNoPlaylist:  originURLNoPlaylist,
				OriginFQDN:           originFQDN,
				OriginPath:           originPath,
				OriginPathNoPlaylist: originPathNoPlaylist,
				OriginPort:           originPort,
				Backups:              backups,
				InsertionRing:        insertionRing,
				Insertion:            insertion,
				BackupIsProxy:        backupIsProxy,
				InserterIsProxy:      inserterIsProxy,
				LegacyShortFormat:    legacyShortFormat,
			})
		}
	}
	return mapperRules, warnings
}
