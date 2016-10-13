package login_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	. "github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var db *gorm.DB
var loginService Service

func TestMain(m *testing.M) {
	if _, c := os.LookupEnv(resource.Database); c != false {

		dbhost := os.Getenv("ALMIGHTY_DB_HOST")
		db, err := gorm.Open("postgres", fmt.Sprintf("host=%s database=postgres user=postgres password=mysecretpassword sslmode=disable", dbhost))

		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
		defer db.Close()
	}

	oauth := &oauth2.Config{
		ClientID:     "875da0d2113ba0a6951d",
		ClientSecret: "2fe6736e90a9283036a37059d75ac0c82f4f5288",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}

	publicKey, err := token.ParsePublicKey(token.RSAPublicKey)
	if err != nil {
		panic(err)
	}

	privateKey, err := token.ParsePrivateKey(token.RSAPrivateKey)
	if err != nil {
		panic(err)
	}

	tokenManager := token.NewManager(publicKey, privateKey)
	userRepository := account.NewUserRepository(db)
	identityRepository := account.NewIdentityRepository(db)
	loginService = NewGitHubOAuth(oauth, identityRepository, userRepository, tokenManager)

	os.Exit(m.Run())
}

func TestGithubOAuthAuthorizationRedirect(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	rw := httptest.NewRecorder()
	u := &url.URL{
		Path: fmt.Sprintf("/api/login/authorize"),
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	req.Header.Add("referer", "https://localhost/path")
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, goa.New("LoginService"))
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = loginService.Perform(authorizeCtx)

	assert.Equal(t, 307, rw.Code)
	assert.Contains(t, rw.Header().Get("Location"), "https://github.com/login/oauth/authorize")
}

func TestValidOAuthAuthorizationCode(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	// Current the OAuth code is generated as part of a UI workflow.
	// Yet to figure out how to mock.
	t.Skip("Authorization Code not avaiable")

}

func TestValidState(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	// We do not have a test for a valid
	// authorization code because it needs a
	// user UI workflow. Furthermore, the code can be used
	// only once. https://tools.ietf.org/html/rfc6749#section-4.1.2
	t.Skip("Authorization Code not avaiable")
}

func TestInvalidState(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	// Setup request context
	rw := httptest.NewRecorder()
	u := &url.URL{
		Path: fmt.Sprintf("/api/login/authorize"),
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{
		"state": {},
		"code":  {"doesnt_matter_what_is_here"},
	}
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, goa.New("LoginService"))
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = loginService.Perform(authorizeCtx)
	assert.Equal(t, 401, rw.Code)
}

func TestInvalidOAuthAuthorizationCodeWithReferer(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	// We do not have a test for a valid
	// authorization code because it needs a
	// user UI workflow. Furthermore, the code can be used
	// only once. https://tools.ietf.org/html/rfc6749#section-4.1.2

	// Setup request context
	rw := httptest.NewRecorder()
	u := &url.URL{
		Path: fmt.Sprintf("/api/login/authorize"),
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	refererUrl := "https://localhost/path"
	req.Header.Add("referer", refererUrl)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, goa.New("LoginService"))
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = loginService.Perform(authorizeCtx)

	assert.Equal(t, 307, rw.Code)

	locationString := rw.HeaderMap["Location"][0]
	locationUrl, err := url.Parse(locationString)
	if err != nil {
		t.Fatal("Redirect URL is in a wrong format ", err)
	}

	t.Log(locationString)
	allQueryParameters := locationUrl.Query()

	// Avoiding panics.
	assert.NotNil(t, allQueryParameters)
	assert.NotNil(t, allQueryParameters["state"][0])

	returnedState := allQueryParameters["state"][0]

	prms = url.Values{
		"state": {returnedState},
		"code":  {"INVALID_OAUTH2.0_CODE"},
	}
	ctx = context.Background()
	rw = httptest.NewRecorder()
	goaCtx = goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err = app.NewAuthorizeLoginContext(goaCtx, goa.New("LoginService"))

	err = loginService.Perform(authorizeCtx)

	locationString = rw.HeaderMap["Location"][0]
	locationUrl, err = url.Parse(locationString)
	if err != nil {
		t.Fatal("Redirect URL is in a wrong format ", err)
	}

	t.Log(locationString)
	allQueryParameters = locationUrl.Query()
	assert.Equal(t, 307, rw.Code)
	// Avoiding panics.
	assert.NotNil(t, allQueryParameters)
	assert.NotNil(t, allQueryParameters["error"])
	assert.Equal(t, allQueryParameters["error"][0], InvalidCodeError)

	returnedErrorReason := allQueryParameters["error"][0]
	assert.NotEmpty(t, returnedErrorReason)
	assert.Contains(t, locationString, refererUrl)
}

func TestInvalidOAuthAuthorizationCodeWithoutReferer(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	// We do not have a test for a valid
	// authorization code because it needs a
	// user UI workflow. Furthermore, the code can be used
	// only once. https://tools.ietf.org/html/rfc6749#section-4.1.2

	// Setup request context
	rw := httptest.NewRecorder()
	u := &url.URL{
		Path: fmt.Sprintf("/api/login/authorize"),
	}
	req, err := http.NewRequest("GET", u.String(), nil)

	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, goa.New("LoginService"))
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = loginService.Perform(authorizeCtx)

	assert.Equal(t, 307, rw.Code)

	locationString := rw.HeaderMap["Location"][0]
	t.Log(locationString)

	locationUrl, err := url.Parse(locationString)
	if err != nil {
		t.Fatal("Redirect URL is in a wrong format ", err)
	}

	allQueryParameters := locationUrl.Query()

	// Avoiding panics.
	assert.NotNil(t, allQueryParameters)
	assert.NotNil(t, allQueryParameters["state"])

	returnedState := allQueryParameters["state"][0]

	prms = url.Values{
		"state": {returnedState},
		"code":  {"INVALID_OAUTH2.0_CODE"},
	}
	ctx = context.Background()
	rw = httptest.NewRecorder()
	goaCtx = goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err = app.NewAuthorizeLoginContext(goaCtx, goa.New("LoginService"))

	err = loginService.Perform(authorizeCtx)
	assert.Equal(t, 401, rw.Code)

}