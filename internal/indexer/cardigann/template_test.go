package cardigann

import (
	"testing"
)

func TestTemplateEngine_Evaluate(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		ctx      *TemplateContext
		want     string
		wantErr  bool
	}{
		{
			name:     "no template markers",
			template: "plain text",
			ctx:      NewTemplateContext(),
			want:     "plain text",
		},
		{
			name:     "config value",
			template: "{{ .Config.username }}",
			ctx: func() *TemplateContext {
				c := NewTemplateContext()
				c.Config["username"] = "testuser"
				return c
			}(),
			want: "testuser",
		},
		{
			name:     "query keywords",
			template: "search={{ .Query.Keywords }}",
			ctx: func() *TemplateContext {
				c := NewTemplateContext()
				c.Query.Keywords = "test query"
				return c
			}(),
			want: "search=test query",
		},
		{
			name:     "keywords shortcut",
			template: "{{ .Keywords }}",
			ctx: func() *TemplateContext {
				c := NewTemplateContext()
				c.Query.Keywords = "shortcut test"
				return c
			}(),
			want: "shortcut test",
		},
		{
			name:     "IMDB ID shortcut",
			template: "imdb={{ .IMDBID }}",
			ctx: func() *TemplateContext {
				c := NewTemplateContext()
				c.Query.IMDBID = "tt1234567"
				return c
			}(),
			want: "imdb=tt1234567",
		},
		{
			name:     "season and episode",
			template: "S{{ .Season }}E{{ .Episode }}",
			ctx: func() *TemplateContext {
				c := NewTemplateContext()
				c.Query.Season = 1
				c.Query.Episode = 5
				return c
			}(),
			want: "S1E5",
		},
		{
			name:     "today context",
			template: "year={{ .Today.Year }}",
			ctx:      NewTemplateContext(),
			want:     "", // Will contain current year
			wantErr:  false,
		},
		{
			name:     "invalid template",
			template: "{{ .Invalid.Nested.Field }}",
			ctx:      NewTemplateContext(),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Evaluate(tt.template, tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.want != "" && got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_Functions(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := NewTemplateContext()

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "join function",
			template: `{{ join .Categories "," }}`,
			want:     "a,b,c",
		},
		{
			name:     "replace function",
			template: `{{ replace "hello world" "world" "there" }}`,
			want:     "hello there",
		},
		{
			name:     "re_replace function",
			template: `{{ re_replace "test123test" "[0-9]+" "X" }}`,
			want:     "testXtest",
		},
		{
			name:     "trim function",
			template: `{{ trim "  hello  " }}`,
			want:     "hello",
		},
		{
			name:     "tolower function",
			template: `{{ tolower "HELLO" }}`,
			want:     "hello",
		},
		{
			name:     "toupper function",
			template: `{{ toupper "hello" }}`,
			want:     "HELLO",
		},
		{
			name:     "prepend function",
			template: `{{ prepend "world" "hello " }}`,
			want:     "hello world",
		},
		{
			name:     "append function",
			template: `{{ append "hello " "world" }}`,
			want:     "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up context for join test
			if tt.name == "join function" {
				ctx.Categories = []string{"a", "b", "c"}
			}

			got, err := engine.Evaluate(tt.template, ctx)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateContext_Creation(t *testing.T) {
	ctx := NewTemplateContext()

	if ctx.Config == nil {
		t.Error("Config map should be initialized")
	}
	if ctx.Categories == nil {
		t.Error("Categories slice should be initialized")
	}
	if ctx.Result == nil {
		t.Error("Result map should be initialized")
	}
	if ctx.Today.Year == 0 {
		t.Error("Today.Year should be set")
	}
}

func TestPreprocessTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "keywords shortcut with spaces",
			input:    "{{ .Keywords }}",
			expected: "{{ .Query.Keywords }}",
		},
		{
			name:     "keywords shortcut no spaces",
			input:    "{{.Keywords}}",
			expected: "{{.Query.Keywords}}",
		},
		{
			name:     "IMDBID shortcut",
			input:    "{{ .IMDBID }}",
			expected: "{{ .Query.IMDBID }}",
		},
		{
			name:     "Season shortcut",
			input:    "{{ .Season }}",
			expected: "{{ .Query.Season }}",
		},
		{
			name:     "already qualified reference preserved",
			input:    "{{ .Query.Keywords }}",
			expected: "{{ .Query.Keywords }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.preprocessTemplate(tt.input)
			if got != tt.expected {
				t.Errorf("preprocessTemplate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEvaluateAll(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := NewTemplateContext()
	ctx.Config["user"] = "testuser"
	ctx.Config["pass"] = "testpass"

	templates := map[string]string{
		"username": "{{ .Config.user }}",
		"password": "{{ .Config.pass }}",
		"static":   "plain",
	}

	result, err := engine.EvaluateAll(templates, ctx)
	if err != nil {
		t.Fatalf("EvaluateAll() error = %v", err)
	}

	if result["username"] != "testuser" {
		t.Errorf("username = %v, want testuser", result["username"])
	}
	if result["password"] != "testpass" {
		t.Errorf("password = %v, want testpass", result["password"])
	}
	if result["static"] != "plain" {
		t.Errorf("static = %v, want plain", result["static"])
	}
}
