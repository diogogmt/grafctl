package grafsdk

import (
	"github.com/diogogmt/grafctl/pkg/simplejson"
)

type SearchType string

const (
	DashHitDB     SearchType = "dash-db"
	DashHitHome   SearchType = "dash-home"
	DashHitFolder SearchType = "dash-folder"
)

type DashboardSavePayload struct {
	Dashboard *simplejson.Json `json:"dashboard"`
	Overwrite bool             `json:"overwrite"`
	FolderID  int64            `json:"folderId"`
	FolderUID string           `json:"folderUid"`
}

type DashboardWithMeta struct {
	Meta      *simplejson.Json `json:"meta"`
	Dashboard *simplejson.Json `json:"dashboard"`
}

type SearchResult struct {
	ID           int64      `json:"id"`
	UID          string     `json:"uid"`
	Title        string     `json:"title"`
	URI          string     `json:"uri"`
	URL          string     `json:"url"`
	Slug         string     `json:"slug"`
	Type         SearchType `json:"type"`
	Tags         []string   `json:"tags"`
	IsStarred    bool       `json:"isStarred"`
	FolderID     int64      `json:"folderId,omitempty"`
	FolderUID    string     `json:"folderUid,omitempty"`
	FolderTitle  string     `json:"folderTitle,omitempty"`
	FolderURL    string     `json:"folderUrl,omitempty"`
	SortMeta     int64      `json:"sortMeta"`
	SortMetaName string     `json:"sortMetaName,omitempty"`
}

type Folder struct {
	ID       int64  `json:"id"`
	UID      string `json:"uid"`
	Title    string `json:"title"`
	Url      string `json:"url"`
	HasACL   bool   `json:"hasAcl"`
	CanSave  bool   `json:"canSave"`
	CanEdit  bool   `json:"canEdit"`
	CanAdmin bool   `json:"canAdmin"`
	Version  int    `json:"version"`
}

type Datasource struct {
	ID                int64            `json:"id"`
	UID               string           `json:"uid"`
	OrgID             int64            `json:"orgId"`
	Name              string           `json:"name"`
	Type              string           `json:"type"`
	TypeLogoURL       string           `json:"typeLogoUrl"`
	Access            string           `json:"access"`
	URL               string           `json:"url"`
	Password          string           `json:"password"`
	User              string           `json:"user"`
	Database          string           `json:"database"`
	BasicAuth         bool             `json:"basicAuth"`
	BasicAuthUser     string           `json:"basicAuthUser"`
	BasicAuthPassword string           `json:"basicAuthPassword"`
	WithCredentials   bool             `json:"withCredentials"`
	IsDefault         bool             `json:"isDefault"`
	JSONData          *simplejson.Json `json:"jsonData,omitempty"`
	SecureJsonFields  map[string]bool  `json:"secureJsonFields"`
	Version           int              `json:"version"`
	ReadOnly          bool             `json:"readOnly"`
}

type PromQLQuery struct {
	Expression  string `json:"expr"`
	ProjectName string `json:"projectName"`
	Step        string `json:"step"`
}
