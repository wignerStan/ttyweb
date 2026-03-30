package server

import (
	"net/http"
	"runtime/debug"
)

// VersionInfo holds build version metadata.
type VersionInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
	Commit    string `json:"commit,omitempty"`
}

// appVersion is set via ldflags: -ldflags "-X ttyweb/server.appVersion=x.y.z"
var appVersion = "dev"

func getAppVersion() VersionInfo {
	info, _ := debug.ReadBuildInfo()
	goVer := "unknown"
	if info != nil {
		goVer = info.GoVersion
	}
	var commit string
	if info != nil {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				if len(s.Value) >= 8 {
					commit = s.Value[:8]
				}
				break
			}
		}
	}
	return VersionInfo{Version: appVersion, GoVersion: goVer, Commit: commit}
}

func (*Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeAPISuccess(w, getAppVersion())
}
