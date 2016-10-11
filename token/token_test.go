package token_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/token"
	jwt "github.com/dgrijalva/jwt-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	manager := createManager(t)

	fullName := "Mr Test Case"

	tokenString := manager.Generate(account.Identity{
		ID:       uuid.NewV4(),
		FullName: fullName,
		ImageURL: "http://some.com/image",
		Emails:   []account.User{account.User{Email: "mr@test.com"}},
	})

	ident, err := manager.Extract(tokenString)
	if err != nil {
		t.Fatal("Could not extract Identity from generated token", err)
	}
	assert.Equal(t, fullName, ident.FullName)
}

func TestExtractWithInvalidToken(t *testing.T) {
	t.Skip("Not implemented, verify token.Valid")
}

func TestLocateTokenInContex(t *testing.T) {
	id := uuid.NewV4()

	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["uuid"] = id.String()
	ctx := goajwt.WithJWT(context.Background(), tk)

	manager := createManager(t)

	foundId, err := manager.Locate(ctx)
	if err != nil {
		t.Error("Failed not locate token in given context", err)
	}
	assert.Equal(t, id, foundId, "ID in created context not equal")
}

func TestLocateMissingTokenInContext(t *testing.T) {
	ctx := context.Background()

	manager := createManager(t)

	_, err := manager.Locate(ctx)
	if err == nil {
		t.Error("Should have returned error on missing token in contex", err)
	}
}

func TestLocateMissingUUIDInTokenInContext(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	ctx := goajwt.WithJWT(context.Background(), tk)

	manager := createManager(t)

	_, err := manager.Locate(ctx)
	if err == nil {
		t.Error("Should have returned error on missing token in contex", err)
	}
}

func TestLocateInvalidUUIDInTokenInContext(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["uuid"] = "131"
	ctx := goajwt.WithJWT(context.Background(), tk)

	manager := createManager(t)

	_, err := manager.Locate(ctx)
	if err == nil {
		t.Error("Should have returned error on missing token in contex", err)
	}
}

func createManager(t *testing.T) token.Manager {
	publicKey, err := token.ParsePublicKey(token.RSAPublicKey)
	if err != nil {
		t.Fatal("Could not parse public key")
	}

	privateKey, err := token.ParsePrivateKey(token.RSAPrivateKey)
	if err != nil {
		t.Fatal("Could not parse private key")
	}

	return token.NewManager(publicKey, privateKey)
}
