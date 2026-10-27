package cardigann

// cardigannDefinition is the YAML read as Prowlarr's CardigannDefinition
// reads it, with every template, filter and regex compiled once.
type cardigannDefinition struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Type        string   `yaml:"type"`
	Language    string   `yaml:"language"`
	Encoding    string   `yaml:"encoding"`
	Links       []string `yaml:"links"`
	LegacyLinks []string `yaml:"legacylinks"`
	// RequestDelay is in seconds; nil when the definition sets none.
	RequestDelay   *float64 `yaml:"requestDelay"`
	FollowRedirect bool     `yaml:"followredirect"`
	// TestLinkTorrent is read from the schema's testlinktorrent key, which
	// Prowlarr's camelCase reader never sees. It defaults to true.
	TestLinkTorrent bool              `yaml:"testlinktorrent"`
	Certificates    []string          `yaml:"certificates"`
	Caps            capabilitiesBlock `yaml:"caps"`
	Settings        []settingsField   `yaml:"settings"`
	Login           *loginBlock       `yaml:"login"`
	Search          searchBlock       `yaml:"search"`
	Download        *downloadBlock    `yaml:"download"`
}

func (d *cardigannDefinition) setDefaults() { d.TestLinkTorrent = true }

type settingsField struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Label    string   `yaml:"label"`
	Default  string   `yaml:"default"`
	Defaults []string `yaml:"defaults"`
	// Options keep their keys as written: an unquoted True stays True.
	Options keyedList[string] `yaml:"options"`
}

type capabilitiesBlock struct {
	Categories       keyedList[string]   `yaml:"categories"`
	CategoryMappings []categoryMapping   `yaml:"categorymappings"`
	Modes            keyedList[[]string] `yaml:"modes"`
	AllowRawSearch   bool                `yaml:"allowrawsearch"`
}

type categoryMapping struct {
	ID      string `yaml:"id"`
	Cat     string `yaml:"cat"`
	Desc    string `yaml:"desc"`
	Default bool   `yaml:"default"`
}

type loginBlock struct {
	Path              *template                `yaml:"path"`
	SubmitPath        string                   `yaml:"submitpath"`
	Cookies           []string                 `yaml:"cookies"`
	Method            string                   `yaml:"method"`
	Form              string                   `yaml:"form"`
	Selectors         bool                     `yaml:"selectors"`
	Inputs            keyedList[*template]     `yaml:"inputs"`
	SelectorInputs    keyedList[selectorBlock] `yaml:"selectorinputs"`
	GetSelectorInputs keyedList[selectorBlock] `yaml:"getselectorinputs"`
	Error             []errorBlock             `yaml:"error"`
	Test              *pageTestBlock           `yaml:"test"`
	Captcha           *captchaBlock            `yaml:"captcha"`
	Headers           keyedList[[]*template]   `yaml:"headers"`
}

type errorBlock struct {
	Path     string         `yaml:"path"`
	Selector string         `yaml:"selector"`
	Message  *selectorBlock `yaml:"message"`
}

// pageTestBlock is login.test. Prowlarr never reads Path.
type pageTestBlock struct {
	Path     string `yaml:"path"`
	Selector string `yaml:"selector"`
}

type captchaBlock struct {
	Type     string `yaml:"type"`
	Selector string `yaml:"selector"`
	Input    string `yaml:"input"`
}

// selectorBlock picks one value from a row, a page or a JSON object.
type selectorBlock struct {
	Selector  *template    `yaml:"selector"`
	Optional  bool         `yaml:"optional"`
	Default   *template    `yaml:"default"`
	Text      *template    `yaml:"text"`
	Attribute string       `yaml:"attribute"`
	Remove    string       `yaml:"remove"`
	Filters   filterBlocks `yaml:"filters"`
	// Case keys are selectors over HTML and literal values over JSON; the
	// values are templates.
	Case keyedList[*template] `yaml:"case"`
}

// searchBlock.Path is appended to Paths when the definition loads.
type searchBlock struct {
	Path                 *template              `yaml:"path"`
	Paths                []searchPathBlock      `yaml:"paths"`
	Headers              keyedList[[]*template] `yaml:"headers"`
	KeywordsFilters      filterBlocks           `yaml:"keywordsfilters"`
	AllowEmptyInputs     bool                   `yaml:"allowEmptyInputs"`
	Inputs               keyedList[*template]   `yaml:"inputs"`
	Error                []errorBlock           `yaml:"error"`
	PreprocessingFilters filterBlocks           `yaml:"preprocessingfilters"`
	Rows                 rowsBlock              `yaml:"rows"`
	Fields               fieldList              `yaml:"fields"`
}

type rowsBlock struct {
	selectorBlock                   `yaml:",inline"`
	After                           int            `yaml:"after"`
	DateHeaders                     *selectorBlock `yaml:"dateheaders"`
	Count                           *selectorBlock `yaml:"count"`
	Multiple                        bool           `yaml:"multiple"`
	MissingAttributeEqualsNoResults bool           `yaml:"missingAttributeEqualsNoResults"`
}

type searchPathBlock struct {
	Path           *template            `yaml:"path"`
	Method         string               `yaml:"method"`
	Inputs         keyedList[*template] `yaml:"inputs"`
	QuerySeparator string               `yaml:"queryseparator"`
	// Categories are tracker category ids. A first entry of "!" negates
	// the rest.
	Categories     []string       `yaml:"categories"`
	InheritInputs  bool           `yaml:"inheritinputs"`
	FollowRedirect bool           `yaml:"followredirect"`
	Response       *responseBlock `yaml:"response"`
}

func (p *searchPathBlock) setDefaults() {
	p.QuerySeparator = "&"
	p.InheritInputs = true
}

// responseBlock is a search path's response block. NoResultsMessage is nil when
// the definition sets none, which differs from an empty message.
type responseBlock struct {
	Type             string  `yaml:"type"`
	NoResultsMessage *string `yaml:"noResultsMessage"`
}

type downloadBlock struct {
	Selectors []selectorField        `yaml:"selectors"`
	Method    string                 `yaml:"method"`
	Before    *beforeBlock           `yaml:"before"`
	Infohash  *infohashBlock         `yaml:"infohash"`
	Headers   keyedList[[]*template] `yaml:"headers"`
}

type beforeBlock struct {
	Path           *template            `yaml:"path"`
	Method         string               `yaml:"method"`
	Inputs         keyedList[*template] `yaml:"inputs"`
	QuerySeparator string               `yaml:"queryseparator"`
	PathSelector   *selectorField       `yaml:"pathselector"`
}

func (b *beforeBlock) setDefaults() { b.QuerySeparator = "&" }

type infohashBlock struct {
	Hash              *selectorField `yaml:"hash"`
	Title             *selectorField `yaml:"title"`
	UseBeforeResponse bool           `yaml:"usebeforeresponse"`
}

// selectorField is a download selector, matched with real CSS :root.
type selectorField struct {
	Selector          *template    `yaml:"selector"`
	Attribute         string       `yaml:"attribute"`
	UseBeforeResponse bool         `yaml:"usebeforeresponse"`
	Filters           filterBlocks `yaml:"filters"`
}

// keyedList is a YAML mapping in the order written, each key kept as its raw
// text: an unquoted True stays True and 1 stays 1. A repeated key takes the
// later value in the first key's place, as a .NET Dictionary does.
type keyedList[T any] []entry[T]

type entry[T any] struct {
	Key   string
	Value T
}

func (m keyedList[T]) Get(key string) (T, bool) {
	for _, e := range m {
		if e.Key == key {
			return e.Value, true
		}
	}
	var zero T
	return zero, false
}

// fieldList are search.fields in the order written. Unlike a keyedList, a key may
// repeat, as title and title|append do (Prowlarr's KeyValuePairList).
type fieldList []entry[selectorBlock]
