package atscfg

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type mapperRule struct {
	DSXMLID              string
	RequestScheme        string
	RequestPort          string
	OriginScheme		 string
	OriginURL			 string
	OriginURLNoPlaylist  string
	OriginFQDN           string
	OriginPath           string
	OriginPathNoPlaylist string
	Backups              []string // each element is a hostname
	InsertionRing        []string // each element is a hostname
	Insertion            bool
	BackupIsProxy        bool
	InserterIsProxy      bool
}

func BuildMapperRules(mapperMode string, mapperMap string) (map[string][]mapperRule, []string) {
	warnings := []string{}
	mapperRules := map[string][]mapperRule{}
	if mapperMode != "" && mapperMap != "" {
		lines := strings.Split(mapperMap, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 10 {
				warnings = append(warnings, "mapper rule line has fewer than 10 fields, skipping: "+line)
				continue
			}
			// Fields: 0=unused, 1=XMLID, 2=RequestScheme, 3=RequestPORT, 4=OriginURL, 5=Backups(csv), 6=Insertion(bool), 7=InsertionServers(csv), 8=BackupIsProxy(bool), 9=InserterIsProxy(bool)
			xmlid := fields[1]
			requestScheme := fields[2]
			requestPort := fields[3]
			originURL := fields[4]
			Backups := fields[5]
			Insertion := fields[6]
			InsertionServers := fields[7]
			BackupIsProxy := fields[8]
			InserterIsProxy := fields[9]

			// RequestScheme: 1=HTTP, 2=HTTPS
			if requestScheme == "1" {
				requestScheme = "http"
			} else if requestScheme == "2" {
				requestScheme = "https"
			}

			// Standard ports are omitted
			if requestPort == "80" && requestScheme == "http" {
				warnings = append(warnings, "mapper rule has RequestPort set to 80 for scheme '"+requestScheme+"', defaulting to empty")
				requestPort = ""
			} else if requestPort == "443" && requestScheme == "https" {
				warnings = append(warnings, "mapper rule has RequestPort set to 443 for scheme '"+requestScheme+"', defaulting to empty")
				requestPort = ""
			}

			// Parse the Origin URL to extract FQDN and Path
			parsedOrigin, err := url.Parse(originURL)
			var originFQDN, originPath string
			if err != nil || parsedOrigin.Host == "" {
				warnings = append(warnings, "mapper rule has invalid origin URL '"+originURL+"', using as-is")
				originFQDN = originURL
				originPath = ""
			} else {
				originFQDN = parsedOrigin.Hostname()
				originPath = parsedOrigin.Path
			}

			// OriginPathNoPlaylist: remove the first path segment matching *.m3u8 and everything after it
			reM3u8 := regexp.MustCompile(`/[^/]*\.m3u8.*$`)
			originPathNoPlaylist := reM3u8.ReplaceAllString(originPath, "")

			originScheme := parsedOrigin.Scheme
			originURL = originScheme + "://" + originFQDN + originPath
			originURLNoPlaylist := originScheme + "://" + originFQDN + originPathNoPlaylist + "/"

			// Backups: comma-separated list, extract hostnames only
			backupParts := strings.Split(Backups, ",")
			backups := make([]string, 0, len(backupParts))
			for _, b := range backupParts {
				backupURL, err := url.Parse(b)
				if err != nil {
					warnings = append(warnings, "mapper rule has invalid backup URL '"+b+"', skipping")
					continue
				}
				host := backupURL.Hostname()
				var port string
				if backupURL.Port() == "" {
					port = "80"
				} else {
					port = backupURL.Port()
				}
				backups = append(backups, host+":"+port)
			}

			// Insertion boolean
			insertion := strings.EqualFold(Insertion, "true")
			backupIsProxy := true
			if len(fields) >= 8 {
				parsedBackupIsProxy, err := strconv.ParseBool(BackupIsProxy)
				if err != nil {
					warnings = append(warnings, "mapper rule has invalid BackupIsProxy value '"+BackupIsProxy+"', defaulting to true")
				} else {
					backupIsProxy = parsedBackupIsProxy
				}
			}
			inserterIsProxy := true
			if len(fields) >= 9 {
				parsedInserterIsProxy, err := strconv.ParseBool(InserterIsProxy)
				if err != nil {
					warnings = append(warnings, "mapper rule has invalid InserterIsProxy value '"+InserterIsProxy+"', defaulting to true")
				} else {
					inserterIsProxy = parsedInserterIsProxy
				}
			}

			// InsertionRing: comma-separated list, extract hostnames only
			insertionParts := strings.Split(InsertionServers, ",")
			insertionRing := make([]string, 0, len(insertionParts))
			for _, b := range insertionParts {
				insertionURL, err := url.Parse(b)
				if err != nil {
					warnings = append(warnings, "mapper rule has invalid insertion URL '"+b+"', skipping")
					continue
				}
				host := insertionURL.Hostname()
				var port string
				if insertionURL.Port() == "" {
					port = "80"
				} else {
					port = insertionURL.Port()
				}
				insertionRing = append(insertionRing, host+":"+port)
			}

			mapperRules[xmlid] = append(mapperRules[xmlid], mapperRule{
				DSXMLID:              xmlid,
				RequestScheme:        requestScheme,
				RequestPort:          requestPort,
				OriginScheme:		  originScheme,
				OriginURL:			  originURL,
				OriginURLNoPlaylist:  originURLNoPlaylist,
				OriginFQDN:           originFQDN,
				OriginPath:           originPath,
				OriginPathNoPlaylist: originPathNoPlaylist,
				Backups:              backups,
				InsertionRing:        insertionRing,
				Insertion:            insertion,
				BackupIsProxy:        backupIsProxy,
				InserterIsProxy:      inserterIsProxy,
			})
		}
	}
	return mapperRules, warnings
}
