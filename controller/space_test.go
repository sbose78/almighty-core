package controller_test

import (
	"context"
	"fmt"
	"testing"

	"time"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/auth"
	"github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var spaceConfiguration *configuration.ConfigurationData

type DummyResourceManager struct {
}

func (m *DummyResourceManager) CreateResource(ctx context.Context, request *goa.RequestData, name string, rType string, uri *string, scopes *[]string, userID string, policyName string) (*auth.Resource, error) {
	return &auth.Resource{ResourceID: uuid.NewV4().String(), PermissionID: uuid.NewV4().String(), PolicyID: uuid.NewV4().String()}, nil
}

func (m *DummyResourceManager) DeleteResource(ctx context.Context, request *goa.RequestData, resource auth.Resource) error {
	return nil
}

func init() {
	var err error
	spaceConfiguration, err = configuration.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

type TestSpaceREST struct {
	gormtestsupport.DBTestSuite
	db            *gormapplication.GormDB
	clean         func()
	iterationRepo iteration.Repository
}

func TestRunSpaceREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestSpaceREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestSpaceREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
	rest.iterationRepo = iteration.NewIterationRepository(rest.DB)
}

func (rest *TestSpaceREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceREST) SecuredController(identity account.Identity) (*goa.Service, *SpaceController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Space-Service", almtoken.NewManagerWithPrivateKey(priv), identity)
	return svc, NewSpaceController(svc, rest.db, spaceConfiguration, &DummyResourceManager{})
}

func (rest *TestSpaceREST) UnSecuredController() (*goa.Service, *SpaceController) {
	svc := goa.New("Space-Service")
	return svc, NewSpaceController(svc, rest.db, spaceConfiguration, &DummyResourceManager{})
}

func (rest *TestSpaceREST) TestFailCreateSpaceUnsecure() {
	// given
	p := minimumRequiredCreateSpace()
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.CreateSpaceUnauthorized(rest.T(), svc.Context, svc, ctrl, p)
}

func (rest *TestSpaceREST) TestFailCreateSpaceMissingName() {
	// given
	p := minimumRequiredCreateSpace()
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	// when/then
	test.CreateSpaceBadRequest(rest.T(), svc.Context, svc, ctrl, p)
}

func (rest *TestSpaceREST) TestSuccessCreateSpace() {
	// given
	name := "TestSuccessCreateSpace-" + uuid.NewV4().String()
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	// when
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// then
	require.NotNil(rest.T(), created.Data)
	require.NotNil(rest.T(), created.Data.Attributes)
	assert.NotNil(rest.T(), created.Data.Attributes.CreatedAt)
	assert.NotNil(rest.T(), created.Data.Attributes.UpdatedAt)
	require.NotNil(rest.T(), created.Data.Attributes.Name)
	assert.Equal(rest.T(), name, *created.Data.Attributes.Name)
	require.NotNil(rest.T(), created.Data.Links)
	assert.NotNil(rest.T(), created.Data.Links.Self)
}

func (rest *TestSpaceREST) SecuredSpaceAreaController(identity account.Identity) (*goa.Service, *SpaceAreasController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	svc := testsupport.ServiceAsUser("Area-Service", almtoken.NewManager(pub), identity)
	return svc, NewSpaceAreasController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceREST) SecuredSpaceIterationController(identity account.Identity) (*goa.Service, *SpaceIterationsController) {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	svc := testsupport.ServiceAsUser("Iteration-Service", almtoken.NewManager(pub), identity)
	return svc, NewSpaceIterationsController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceREST) TestSuccessCreateSpaceAndDefaultArea() {
	// given
	name := "TestSuccessCreateSpaceAndDefaultArea-" + uuid.NewV4().String()
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	// when
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	require.NotNil(rest.T(), created.Data)
	spaceAreaSvc, spaceAreaCtrl := rest.SecuredSpaceAreaController(testsupport.TestIdentity)
	createdID := created.Data.ID.String()
	_, areaList := test.ListSpaceAreasOK(rest.T(), spaceAreaSvc.Context, spaceAreaSvc, spaceAreaCtrl, createdID, nil, nil)
	// then
	// only 1 default gets created.
	assert.Len(rest.T(), areaList.Data, 1)
	assert.Equal(rest.T(), name, *areaList.Data[0].Attributes.Name)

	// verify if root iteration is created or not
	spaceIterationSvc, spaceIterationCtrl := rest.SecuredSpaceIterationController(testsupport.TestIdentity)
	_, iterationList := test.ListSpaceIterationsOK(rest.T(), spaceIterationSvc.Context, spaceIterationSvc, spaceIterationCtrl, createdID, nil, nil)
	require.Len(rest.T(), iterationList.Data, 1)
	assert.Equal(rest.T(), name, *iterationList.Data[0].Attributes.Name)

}

func (rest *TestSpaceREST) TestSuccessCreateSpaceWithDescription() {
	// given
	name := "TestSuccessCreateSpaceWithDescription-" + uuid.NewV4().String()
	description := "Space for TestSuccessCreateSpaceWithDescription"
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	// when
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// then
	assert.NotNil(rest.T(), created.Data)
	assert.NotNil(rest.T(), created.Data.Attributes)
	assert.NotNil(rest.T(), created.Data.Attributes.CreatedAt)
	assert.NotNil(rest.T(), created.Data.Attributes.UpdatedAt)
	assert.NotNil(rest.T(), created.Data.Attributes.Name)
	assert.Equal(rest.T(), name, *created.Data.Attributes.Name)
	assert.NotNil(rest.T(), created.Data.Attributes.Description)
	assert.Equal(rest.T(), description, *created.Data.Attributes.Description)
	assert.NotNil(rest.T(), created.Data.Links)
	assert.NotNil(rest.T(), created.Data.Links.Self)
}

func (rest *TestSpaceREST) TestSuccessUpdateSpace() {
	// given
	name := "TestSuccessUpdateSpace-" + uuid.NewV4().String()
	description := "Space for TestSuccessUpdateSpace"
	newName := "TestSuccessUpdateSpace" + uuid.NewV4().String()
	newDescription := "Space for TestSuccessUpdateSpace2"
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	u := minimumRequiredUpdateSpace()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Version = created.Data.Attributes.Version
	u.Data.Attributes.Name = &newName
	u.Data.Attributes.Description = &newDescription
	// when
	_, updated := test.UpdateSpaceOK(rest.T(), svc.Context, svc, ctrl, created.Data.ID.String(), u)
	// then
	assert.Equal(rest.T(), newName, *updated.Data.Attributes.Name)
	assert.Equal(rest.T(), newDescription, *updated.Data.Attributes.Description)
}

func (rest *TestSpaceREST) TestFailUpdateSpaceDifferentOwner() {
	// given
	name := "TestFailUpdateSpaceDifferentOwner-" + uuid.NewV4().String()
	description := "Space for TestFailUpdateSpaceDifferentOwner"
	newName := "TestFailUpdateSpaceDifferentOwner-" + uuid.NewV4().String()
	newDescription := "Space for TestFailUpdateSpaceDifferentOwner2"
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// when
	u := minimumRequiredUpdateSpace()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Version = created.Data.Attributes.Version
	u.Data.Attributes.Name = &newName
	u.Data.Attributes.Description = &newDescription
	svc2, ctrl2 := rest.SecuredController(testsupport.TestIdentity2)
	_, errors := test.UpdateSpaceForbidden(rest.T(), svc2.Context, svc2, ctrl2, created.Data.ID.String(), u)
	// then
	assert.NotEmpty(rest.T(), errors.Errors)
	assert.Contains(rest.T(), errors.Errors[0].Detail, "User is not the space owner")
}

func (rest *TestSpaceREST) TestFailUpdateSpaceUnSecure() {
	// given
	u := minimumRequiredUpdateSpace()
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.UpdateSpaceUnauthorized(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), u)
}

func (rest *TestSpaceREST) TestFailUpdateSpaceNotFound() {
	// given
	name := "TestFailUpdateSpaceNotFound-" + uuid.NewV4().String()
	version := 0
	id := uuid.NewV4()
	u := minimumRequiredUpdateSpace()
	u.Data.Attributes.Name = &name
	u.Data.Attributes.Version = &version
	u.Data.ID = &id
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	// when/then
	test.UpdateSpaceNotFound(rest.T(), svc.Context, svc, ctrl, id.String(), u)
}

func (rest *TestSpaceREST) TestFailUpdateSpaceMissingName() {
	// given
	name := "TestFailUpdateSpaceMissingName-" + uuid.NewV4().String()
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	u := minimumRequiredUpdateSpace()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Version = created.Data.Attributes.Version
	// when/then
	test.UpdateSpaceBadRequest(rest.T(), svc.Context, svc, ctrl, created.Data.ID.String(), u)
}

func (rest *TestSpaceREST) TestFailUpdateSpaceMissingVersion() {
	// given
	name := "TestFailUpdateSpaceMissingVersion-" + uuid.NewV4().String()
	newName := "TestFailUpdateSpaceMissingVersion-" + uuid.NewV4().String()
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	u := minimumRequiredUpdateSpace()
	u.Data.ID = created.Data.ID
	u.Data.Attributes.Name = &newName
	// when/then
	test.UpdateSpaceBadRequest(rest.T(), svc.Context, svc, ctrl, created.Data.ID.String(), u)
}

func (rest *TestSpaceREST) TestShowSpaceOK() {
	// given
	name := "TestShowSpaceOK-" + uuid.NewV4().String()
	description := "Space for TestShowSpaceOK"
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// when
	res, fetched := test.ShowSpaceOK(rest.T(), svc.Context, svc, ctrl, created.Data.ID.String(), nil, nil)
	// then
	assert.Equal(rest.T(), created.Data.ID, fetched.Data.ID)
	assert.Equal(rest.T(), *created.Data.Attributes.Name, *fetched.Data.Attributes.Name)
	assert.Equal(rest.T(), *created.Data.Attributes.Description, *fetched.Data.Attributes.Description)
	assert.Equal(rest.T(), *created.Data.Attributes.Version, *fetched.Data.Attributes.Version)
	require.NotNil(rest.T(), res.Header()[app.LastModified])
	assert.Equal(rest.T(), app.ToHTTPTime(getSpaceUpdatedAt(*created)), res.Header()[app.LastModified][0])
	require.NotNil(rest.T(), res.Header()[app.CacheControl])
	assert.NotNil(rest.T(), res.Header()[app.CacheControl][0])
	require.NotNil(rest.T(), res.Header()[app.ETag])
	assert.Equal(rest.T(), app.GenerateEntityTag(ConvertSpaceToModel(*created.Data)), res.Header()[app.ETag][0])
	// Test that it contains the right link for backlog items
	subStringBacklogUrl := fmt.Sprintf("/%s/backlog", fetched.Data.ID.String())
	assert.Contains(rest.T(), *fetched.Data.Links.Backlog.Self, subStringBacklogUrl)
	assert.Equal(rest.T(), fetched.Data.Links.Backlog.Meta.TotalCount, 0)

	// Test that it contains the right relationship values
	subString := fmt.Sprintf("/%s/iterations", fetched.Data.ID.String())
	assert.Contains(rest.T(), *fetched.Data.Relationships.Iterations.Links.Related, subString)
	subStringAreaUrl := fmt.Sprintf("/%s/areas", fetched.Data.ID.String())
	assert.Contains(rest.T(), *fetched.Data.Relationships.Areas.Links.Related, subStringAreaUrl)
}

func (rest *TestSpaceREST) TestShowSpaceOKUsingExpiredIfModifiedSinceHeader() {
	// given
	name := "TestShowSpaceOKUsingExpiredIfModifiedSinceHeader-" + uuid.NewV4().String()
	description := "Space for TestShowSpaceOKUsingExpiredIfModifiedSinceHeader"
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// when
	ifModifiedSince := app.ToHTTPTime(created.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
	res, fetched := test.ShowSpaceOK(rest.T(), svc.Context, svc, ctrl, created.Data.ID.String(), &ifModifiedSince, nil)
	// then
	assert.Equal(rest.T(), created.Data.ID, fetched.Data.ID)
	assert.Equal(rest.T(), *created.Data.Attributes.Name, *fetched.Data.Attributes.Name)
	assert.Equal(rest.T(), *created.Data.Attributes.Description, *fetched.Data.Attributes.Description)
	assert.Equal(rest.T(), *created.Data.Attributes.Version, *fetched.Data.Attributes.Version)
	require.NotNil(rest.T(), res.Header()[app.LastModified])
	assert.Equal(rest.T(), app.ToHTTPTime(getSpaceUpdatedAt(*created)), res.Header()[app.LastModified][0])
	require.NotNil(rest.T(), res.Header()[app.CacheControl])
	assert.NotNil(rest.T(), res.Header()[app.CacheControl][0])
	require.NotNil(rest.T(), res.Header()[app.ETag])
	assert.Equal(rest.T(), generateSpaceTag(*created), res.Header()[app.ETag][0])
}

func (rest *TestSpaceREST) TestShowSpaceOKUsingExpiredIfNoneMatchHeader() {
	// given
	name := "TestShowSpaceOKUsingExpiredIfNoneMatchHeader-" + uuid.NewV4().String()
	description := "Space for TestShowSpaceOKUsingExpiredIfNoneMatchHeader"
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// when
	ifNoneMatch := "foo_etag"
	res, fetched := test.ShowSpaceOK(rest.T(), svc.Context, svc, ctrl, created.Data.ID.String(), nil, &ifNoneMatch)
	// then
	assert.Equal(rest.T(), created.Data.ID, fetched.Data.ID)
	assert.Equal(rest.T(), *created.Data.Attributes.Name, *fetched.Data.Attributes.Name)
	assert.Equal(rest.T(), *created.Data.Attributes.Description, *fetched.Data.Attributes.Description)
	assert.Equal(rest.T(), *created.Data.Attributes.Version, *fetched.Data.Attributes.Version)
	require.NotNil(rest.T(), res.Header()[app.LastModified])
	assert.Equal(rest.T(), app.ToHTTPTime(getSpaceUpdatedAt(*created)), res.Header()[app.LastModified][0])
	require.NotNil(rest.T(), res.Header()[app.CacheControl])
	assert.NotNil(rest.T(), res.Header()[app.CacheControl][0])
	require.NotNil(rest.T(), res.Header()[app.ETag])
	assert.Equal(rest.T(), generateSpaceTag(*created), res.Header()[app.ETag][0])
}

func (rest *TestSpaceREST) TestShowSpaceNotModifiedUsingIfModifiedSinceHeader() {
	// given
	name := "TestShowSpaceNotModifiedUsingIfModifiedSinceHeader-" + uuid.NewV4().String()
	description := "Space for TestShowSpaceNotModifiedUsingIfModifiedSinceHeader"
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// when/then
	ifModifiedSince := app.ToHTTPTime(getSpaceUpdatedAt(*created))
	test.ShowSpaceNotModified(rest.T(), svc.Context, svc, ctrl, created.Data.ID.String(), &ifModifiedSince, nil)
}

func (rest *TestSpaceREST) TestShowSpaceNotModifiedUsingIfNoneMatchHeader() {
	// given
	name := "TestShowSpaceNotModifiedUsingIfNoneMatchHeader-" + uuid.NewV4().String()
	description := "Space for TestShowSpaceNotModifiedUsingIfNoneMatchHeader"
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	p.Data.Attributes.Description = &description
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, created := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// when/then
	ifNoneMatch := generateSpaceTag(*created)
	// test.ShowSpaceNotModified(rest.T(), svc.Context, svc, ctrl, created.Data.ID.String(), nil, &ifNoneMatch)
	test.ShowSpaceNotModified(rest.T(), svc.Context, svc, ctrl, created.Data.ID.String(), nil, &ifNoneMatch)

	t := rest.T()
	_, fetched := test.ShowSpaceOK(t, svc.Context, svc, ctrl, created.Data.ID.String(), nil, nil)
	assert.Equal(t, created.Data.ID, fetched.Data.ID)
	assert.Equal(t, *created.Data.Attributes.Name, *fetched.Data.Attributes.Name)
	assert.Equal(t, *created.Data.Attributes.Description, *fetched.Data.Attributes.Description)
	assert.Equal(t, *created.Data.Attributes.Version, *fetched.Data.Attributes.Version)

	// verify list-WI URL exists in Relationships.Links
	require.NotNil(t, *fetched.Data.Relationships.Workitems)
	require.NotNil(t, *fetched.Data.Relationships.Workitems.Links)
	require.NotNil(t, *fetched.Data.Relationships.Workitems.Links.Related)
	subStringWI := fmt.Sprintf("/%s/workitems", created.Data.ID.String())
	assert.Contains(t, *fetched.Data.Relationships.Workitems.Links.Related, subStringWI)

	// verify list-WIT URL exists in Relationships.Links
	require.NotNil(t, *fetched.Data.Links)
	require.NotNil(t, fetched.Data.Links.Workitemtypes)
	subStringWIL := fmt.Sprintf("/%s/workitemtypes", created.Data.ID.String())
	assert.Contains(t, *fetched.Data.Links.Workitemtypes, subStringWIL)

	// verify list-WILT URL exists in Relationships.Links
	require.NotNil(t, *fetched.Data.Links)
	require.NotNil(t, fetched.Data.Links.Workitemlinktypes)
	subStringWILT := fmt.Sprintf("/%s/workitemlinktypes", created.Data.ID.String())
	assert.Contains(t, *fetched.Data.Links.Workitemlinktypes, subStringWILT)

	// verify list-Collaborators URL exists in Relationships.Links
	require.NotNil(t, fetched.Data.Relationships.Collaborators)
	require.NotNil(t, fetched.Data.Relationships.Collaborators.Links)
	require.NotNil(t, fetched.Data.Relationships.Collaborators.Links.Related)
	subStringCollaborators := fmt.Sprintf("/%s/collaborators", created.Data.ID.String())
	assert.Contains(t, *fetched.Data.Relationships.Collaborators.Links.Related, subStringCollaborators)
}

func (rest *TestSpaceREST) TestFailShowSpaceNotFound() {
	// given
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.ShowSpaceNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
}

func (rest *TestSpaceREST) TestFailShowSpaceNotFoundBadID() {
	// given
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.ShowSpaceNotFound(rest.T(), svc.Context, svc, ctrl, "asfasfsaf", nil, nil)
}

func (rest *TestSpaceREST) TestListSpacesOK() {
	// given
	name := "TestListSpacesOK-" + uuid.NewV4().String()
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// when
	_, list := test.ListSpaceOK(rest.T(), svc.Context, svc, ctrl, nil, nil, nil, nil)
	// then
	require.NotNil(rest.T(), list)
	require.NotEmpty(rest.T(), list.Data)
}

func (rest *TestSpaceREST) TestListSpacesOKUsingExpiredIfModifiedSinceHeader() {
	// given
	name := "TestListSpacesOKUsingExpiredIfModifiedSinceHeader-" + uuid.NewV4().String()
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// when
	ifModifiedSince := app.ToHTTPTime(time.Now().Add(-1 * time.Hour))
	_, list := test.ListSpaceOK(rest.T(), svc.Context, svc, ctrl, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(rest.T(), list)
	require.NotEmpty(rest.T(), list.Data)
}

func (rest *TestSpaceREST) TestListSpacesOKUsingExpiredIfNoneMatchHeader() {
	// given
	name := "TestListSpacesOKUsingExpiredIfNoneMatchHeader-" + uuid.NewV4().String()
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// when
	ifNoneMatch := "fooo-spaces"
	_, list := test.ListSpaceOK(rest.T(), svc.Context, svc, ctrl, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(rest.T(), list)
	require.NotEmpty(rest.T(), list.Data)
}

func (rest *TestSpaceREST) TestListSpacesNotModifiedUsingIfModifiedSinceHeader() {
	// given
	name := "TestListSpacesNotModifiedUsingIfModifiedSinceHeader-" + uuid.NewV4().String()
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	_, createdSpace := test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	// when/then
	ifModifiedSince := app.ToHTTPTime(*createdSpace.Data.Attributes.UpdatedAt)
	test.ListSpaceNotModified(rest.T(), svc.Context, svc, ctrl, nil, nil, &ifModifiedSince, nil)
}

func (rest *TestSpaceREST) TestListSpacesNotModifiedUsingIfNoneMatchHeader() {
	// given
	name := "TestListSpacesNotModifiedUsingIfNoneMatchHeader-" + uuid.NewV4().String()
	p := minimumRequiredCreateSpace()
	p.Data.Attributes.Name = &name
	svc, ctrl := rest.SecuredController(testsupport.TestIdentity)
	test.CreateSpaceCreated(rest.T(), svc.Context, svc, ctrl, p)
	_, spaceList := test.ListSpaceOK(rest.T(), svc.Context, svc, ctrl, nil, nil, nil, nil)
	// when/then
	ifNoneMatch := generateSpacesTag(*spaceList)
	test.ListSpaceNotModified(rest.T(), svc.Context, svc, ctrl, nil, nil, nil, &ifNoneMatch)
}

func minimumRequiredCreateSpace() *app.CreateSpacePayload {
	return &app.CreateSpacePayload{
		Data: &app.Space{
			Type:       "spaces",
			Attributes: &app.SpaceAttributes{},
		},
	}
}

func CreateSpacePayload(name, description string) *app.CreateSpacePayload {
	return &app.CreateSpacePayload{
		Data: &app.Space{
			Type: "spaces",
			Attributes: &app.SpaceAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
}

func minimumRequiredUpdateSpace() *app.UpdateSpacePayload {
	return &app.UpdateSpacePayload{
		Data: &app.Space{
			Type:       "spaces",
			Attributes: &app.SpaceAttributes{},
		},
	}
}

func generateSpacesTag(entities app.SpaceList) string {
	modelEntities := make([]app.ConditionalResponseEntity, len(entities.Data))
	for i, entityData := range entities.Data {
		modelEntities[i] = ConvertSpaceToModel(*entityData)
	}
	return app.GenerateEntitiesTag(modelEntities)
}

func generateSpaceTag(entity app.SpaceSingle) string {
	return app.GenerateEntityTag(ConvertSpaceToModel(*entity.Data))
}

func convertSpacesToConditionalEntities(spaceList app.SpaceList) []app.ConditionalResponseEntity {
	conditionalSpaces := make([]app.ConditionalResponseEntity, len(spaceList.Data))
	for i, spaceData := range spaceList.Data {
		conditionalSpaces[i] = ConvertSpaceToModel(*spaceData)
	}
	return conditionalSpaces
}

func getSpaceUpdatedAt(appSpace app.SpaceSingle) time.Time {
	return appSpace.Data.Attributes.UpdatedAt.Truncate(time.Second).UTC()
}
