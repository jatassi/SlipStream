package indexersapi

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
)

var (
	captchaExpired  = api.NewCode(http.StatusGone, "captcha_expired", "The captcha expired. Load a new one and type its code.")
	captchaNotShown = api.NewCode(http.StatusUnprocessableEntity, "captcha_not_shown", "The Indexer's sign-in shows no captcha")
)

type captchaImage struct {
	Attempt   string    `json:"attempt" doc:"Send it back with the code the image shows"`
	Image     string    `json:"image" doc:"The captcha as a data URL"`
	ExpiresAt time.Time `json:"expiresAt" doc:"When the attempt closes. Load a new captcha after it."`
}

func captchaOf(c indexers.Captcha) captchaImage {
	return captchaImage{
		Attempt: c.Attempt, Image: "data:" + c.ContentType + ";base64," + base64.StdEncoding.EncodeToString(c.Image), ExpiresAt: c.Expires,
	}
}

type captchaAnswer struct {
	Attempt string `json:"attempt" minLength:"1" maxLength:"64" doc:"The attempt the captcha was loaded with"`
	Answer  string `json:"answer" maxLength:"64" doc:"The code the captcha shows"`
}

func answerIn(c *captchaAnswer) *indexers.CaptchaAnswer {
	if c == nil {
		return nil
	}
	return &indexers.CaptchaAnswer{Attempt: c.Attempt, Text: c.Answer}
}

type testBody struct {
	Captcha *captchaAnswer `json:"captcha,omitempty" doc:"The answer to a captcha loaded for this Indexer"`
}

type testIndexerInput struct {
	ID   string `path:"id" format:"uuid"`
	Body *testBody
}

type captchaOutput struct{ Body captchaImage }

type captchaDraftInput struct{ Body newIndexer }

func registerCaptcha(o *api.Owner, h handlers, drafted []*api.Code) {
	api.Register(o, api.Admin, api.Op{
		ID:      "getIndexerCaptchaDraft",
		Method:  http.MethodPost,
		Path:    "/indexers/captcha",
		Summary: "Load the captcha of an unsaved Indexer's sign-in, from the Node it would search from",
		Status:  http.StatusOK,
		Errors:  append([]*api.Code{captchaNotShown}, drafted...),
	}, h.captchaDraft)

	api.Register(o, api.Admin, api.Op{
		ID:      "getIndexerCaptcha",
		Method:  http.MethodPost,
		Path:    "/indexers/{id}/captcha",
		Summary: "Load the captcha of an Indexer's sign-in, from the Node it searches from",
		Status:  http.StatusOK,
		Errors:  []*api.Code{indexerNotFound, captchaNotShown, indexerTestFailed.Code, indexerUnavailable},
	}, h.captcha)
}

func (h handlers) captchaDraft(ctx context.Context, by identity.Admin, in *captchaDraftInput) (*captchaOutput, error) {
	if in.Body.Captcha != nil {
		return nil, api.Invalid.On("captcha", "Load a captcha before you answer one.")
	}
	d, err := in.Body.draft()
	if err != nil {
		return nil, err
	}
	xs := open(h.registry)
	c, err := xs.Captcha(ctx, by, indexers.CaptchaFor{Draft: &d})
	if err != nil {
		return nil, answer(xs, err)
	}
	return &captchaOutput{Body: captchaOf(c)}, nil
}

func (h handlers) captcha(ctx context.Context, by identity.Admin, in *idInput) (*captchaOutput, error) {
	id, err := parseID(in.ID)
	if err != nil {
		return nil, err
	}
	xs := open(h.registry)
	c, err := xs.Captcha(ctx, by, indexers.CaptchaFor{Indexer: id})
	if err != nil {
		return nil, answer(xs, err)
	}
	return &captchaOutput{Body: captchaOf(c)}, nil
}
