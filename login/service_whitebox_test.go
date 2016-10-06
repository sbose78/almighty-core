package login

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/token"
	"github.com/jinzhu/gorm"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var db *gorm.DB
var loginService *gitHubOAuth

func setup() {
	if _, c := os.LookupEnv(resource.Database); c == false {
		fmt.Printf(resource.StSkipReasonNotSet+"\n", resource.Database)
		return
	}

	dbhost := os.Getenv("ALMIGHTY_DB_HOST")
	if "" == dbhost {
		panic("The environment variable ALMIGHTY_DB_HOST is not specified or empty.")
	}
	var err error
	db, err = gorm.Open("postgres", fmt.Sprintf("host=%s database=postgres user=postgres password=mysecretpassword sslmode=disable", dbhost))
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	defer db.Close()

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
	loginService = &gitHubOAuth{
		config:       oauth,
		identities:   identityRepository,
		users:        userRepository,
		tokenManager: tokenManager,
	}
}

func TestValidOAuthAccessToken(t *testing.T) {
	resource.Require(t, resource.Database)
	setup()
	accessToken := &oauth2.Token{
		AccessToken: "ccd845b7499e20b6faaf4dc036845a12fd5d1ee6",
		TokenType:   "Bearer",
	}
	emails, err := loginService.getUserEmails(context.Background(), accessToken)
	t.Log(emails, err)
}

func TestInvalidOAuthAccessToken(t *testing.T) {
	resource.Require(t, resource.Database)
}

func TestGetUserEmails(t *testing.T) {

}

func TestGetUser(t *testing.T) {

}

func TestFilterPrimaryEmail(t *testing.T) {

}
