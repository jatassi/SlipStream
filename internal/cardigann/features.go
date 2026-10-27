package cardigann

import (
	"slices"
	"strings"
)

// Feature is a part of the Cardigann format that a definition uses, named
// as the conformance list shows it.
type Feature string

// The features, in the order Features lists them.
const (
	LoginForm              Feature = "login method form"
	LoginPost              Feature = "login method post"
	LoginCookie            Feature = "login method cookie"
	LoginGet               Feature = "login method get"
	LoginOneURL            Feature = "login method oneurl"
	LoginSelectorInputs    Feature = "login selectorinputs"
	LoginGetSelectorInputs Feature = "login getselectorinputs"
	LoginSubmitPath        Feature = "login submitpath"
	LoginCookies           Feature = "login cookies"
	LoginHeaders           Feature = "login headers"
	LoginError             Feature = "login error"
	LoginTest              Feature = "login test"
	LoginTestPath          Feature = "login test path"
	LoginCaptcha           Feature = "image captcha"
	ResponseHTML           Feature = "HTML responses"
	ResponseXML            Feature = "XML responses"
	ResponseJSON           Feature = "JSON responses"
	SearchPaths            Feature = "several search paths"
	SearchPathCategories   Feature = "search paths by category"
	SearchPost             Feature = "search method post"
	SearchHeaders          Feature = "search headers"
	SearchError            Feature = "search error"
	KeywordsFilters        Feature = "keywordsfilters"
	PreprocessingFilters   Feature = "preprocessingfilters"
	NoResultsMessage       Feature = "noResultsMessage"
	RowsAfter              Feature = "rows after"
	RowsDateHeaders        Feature = "rows dateheaders"
	RowsCount              Feature = "rows count"
	RowsMultiple           Feature = "rows multiple"
	RowsAttribute          Feature = "rows attribute"
	FollowRedirect         Feature = "followredirect"
	DownloadSelectors      Feature = "download selectors"
	DownloadBefore         Feature = "download before"
	DownloadInfohash       Feature = "download infohash"
	DownloadPost           Feature = "download method post"
	DownloadHeaders        Feature = "download headers"
	NoTestLinkTorrent      Feature = "testlinktorrent false"
	Certificates           Feature = "certificates"
	LegacyEncoding         Feature = "an encoding other than UTF-8"
)

// Features are the features a definition uses, in the order of the
// constants.
type Features []Feature

// Has reports whether f is among fs.
func (fs Features) Has(f Feature) bool { return slices.Contains(fs, f) }

// The methods as a definition writes them.
const (
	methodGet  = "get"
	methodPost = "post"
)

// unported are the features a definition may use that SlipStream can't run
// yet. Load refuses a definition that uses one.
var unported = []Feature{
}

type featureCheck struct {
	feature Feature
	uses    func(*cardigannDefinition) bool
}

var featureChecks = []featureCheck{
	{LoginForm, loginMethodIs("form")},
	{LoginPost, loginMethodIs("post")},
	{LoginCookie, loginMethodIs("cookie")},
	{LoginGet, loginMethodIs("get")},
	{LoginOneURL, loginMethodIs("oneurl")},
	{LoginSelectorInputs, loginHas(func(l *loginBlock) bool { return len(l.SelectorInputs) > 0 })},
	{LoginGetSelectorInputs, loginHas(func(l *loginBlock) bool { return len(l.GetSelectorInputs) > 0 })},
	{LoginSubmitPath, loginHas(func(l *loginBlock) bool { return l.SubmitPath != "" })},
	{LoginCookies, loginHas(func(l *loginBlock) bool { return len(l.Cookies) > 0 })},
	{LoginHeaders, loginHas(func(l *loginBlock) bool { return len(l.Headers) > 0 })},
	{LoginError, loginHas(func(l *loginBlock) bool { return len(l.Error) > 0 })},
	{LoginTest, loginHas(func(l *loginBlock) bool { return l.Test != nil })},
	{LoginTestPath, loginHas(func(l *loginBlock) bool { return l.Test != nil && l.Test.Path != "" })},
	{LoginCaptcha, loginHas(func(l *loginBlock) bool { return l.Captcha != nil })},
	{ResponseHTML, anyPath(func(p searchPathBlock) bool { return responseType(p) == "" })},
	{ResponseXML, anyPath(func(p searchPathBlock) bool { return responseType(p) == "xml" })},
	{ResponseJSON, anyPath(func(p searchPathBlock) bool { return responseType(p) == "json" })},
	{SearchPaths, func(d *cardigannDefinition) bool { return len(d.Search.Paths) > 1 }},
	{SearchPathCategories, anyPath(func(p searchPathBlock) bool { return len(p.Categories) > 0 })},
	{SearchPost, anyPath(func(p searchPathBlock) bool { return p.Method == methodPost })},
	{SearchHeaders, func(d *cardigannDefinition) bool { return len(d.Search.Headers) > 0 }},
	{SearchError, func(d *cardigannDefinition) bool { return len(d.Search.Error) > 0 }},
	{KeywordsFilters, func(d *cardigannDefinition) bool { return len(d.Search.KeywordsFilters) > 0 }},
	{PreprocessingFilters, func(d *cardigannDefinition) bool { return len(d.Search.PreprocessingFilters) > 0 }},
	{NoResultsMessage, anyPath(func(p searchPathBlock) bool { return p.Response != nil && p.Response.NoResultsMessage != nil })},
	{RowsAfter, func(d *cardigannDefinition) bool { return d.Search.Rows.After > 0 }},
	{RowsDateHeaders, func(d *cardigannDefinition) bool { return d.Search.Rows.DateHeaders != nil }},
	{RowsCount, func(d *cardigannDefinition) bool { return d.Search.Rows.Count != nil }},
	{RowsMultiple, func(d *cardigannDefinition) bool { return d.Search.Rows.Multiple }},
	{RowsAttribute, func(d *cardigannDefinition) bool { return d.Search.Rows.Attribute != "" }},
	{FollowRedirect, func(d *cardigannDefinition) bool {
		return d.FollowRedirect || slices.ContainsFunc(d.Search.Paths, func(p searchPathBlock) bool { return p.FollowRedirect })
	}},
	{DownloadSelectors, downloadHas(func(b *downloadBlock) bool { return len(b.Selectors) > 0 })},
	{DownloadBefore, downloadHas(func(b *downloadBlock) bool { return b.Before != nil })},
	{DownloadInfohash, downloadHas(func(b *downloadBlock) bool { return b.Infohash != nil })},
	{DownloadPost, downloadHas(func(b *downloadBlock) bool { return b.Method == methodPost })},
	{DownloadHeaders, downloadHas(func(b *downloadBlock) bool { return len(b.Headers) > 0 })},
	{NoTestLinkTorrent, func(d *cardigannDefinition) bool { return !d.TestLinkTorrent }},
	{Certificates, func(d *cardigannDefinition) bool { return len(d.Certificates) > 0 }},
	{LegacyEncoding, func(d *cardigannDefinition) bool { return !strings.EqualFold(d.Encoding, "UTF-8") }},
}

func usedFeatures(def *cardigannDefinition) Features {
	var out Features
	for _, c := range featureChecks {
		if c.uses(def) {
			out = append(out, c.feature)
		}
	}
	return out
}

func loginMethodIs(method string) func(*cardigannDefinition) bool {
	return loginHas(func(l *loginBlock) bool { return l.Method == method })
}

func loginHas(f func(*loginBlock) bool) func(*cardigannDefinition) bool {
	return func(d *cardigannDefinition) bool { return d.Login != nil && f(d.Login) }
}

func downloadHas(f func(*downloadBlock) bool) func(*cardigannDefinition) bool {
	return func(d *cardigannDefinition) bool { return d.Download != nil && f(d.Download) }
}

func anyPath(f func(searchPathBlock) bool) func(*cardigannDefinition) bool {
	return func(d *cardigannDefinition) bool { return slices.ContainsFunc(d.Search.Paths, f) }
}

func responseType(p searchPathBlock) string {
	if p.Response == nil || p.Response.Type == "html" {
		return ""
	}
	return p.Response.Type
}
