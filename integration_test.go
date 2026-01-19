package go_salesforce_api_client_test

import (
	"strings"
	"testing"

	go_salesforce_api_client "github.com/MASA-JAPAN/go-salesforce-api-client"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/auth"
	sfemulator "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/emulator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticatePassword_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New(
		sfemulator.WithCredentials(auth.Credential{
			ClientID:     "test_client_id",
			ClientSecret: "test_client_secret",
			Username:     "test@example.com",
			Password:     "test_password",
		}),
	)
	emu.Start()
	defer emu.Stop()

	authConfig := go_salesforce_api_client.Auth{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		Username:     "test@example.com",
		Password:     "test_password",
		TokenURL:     emu.URL() + "/services/oauth2/token",
	}

	client, err := authConfig.AuthenticatePassword()
	require.NoError(t, err)
	assert.NotEmpty(t, client.AccessToken, "Access token should be set")
	assert.NotEmpty(t, client.InstanceURL, "Instance URL should be set")
	assert.True(t, strings.HasPrefix(client.InstanceURL, "http"), "Instance URL should be a valid URL")
}

func TestAuthenticatePassword_InvalidCredentials(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New(
		sfemulator.WithCredentials(auth.Credential{
			ClientID:     "valid_client_id",
			ClientSecret: "valid_client_secret",
			Username:     "valid@example.com",
			Password:     "valid_password",
		}),
	)
	emu.Start()
	defer emu.Stop()

	authConfig := go_salesforce_api_client.Auth{
		ClientID:     "wrong_client_id",
		ClientSecret: "wrong_client_secret",
		Username:     "wrong@example.com",
		Password:     "wrong_password",
		TokenURL:     emu.URL() + "/services/oauth2/token",
	}

	client, err := authConfig.AuthenticatePassword()
	require.Error(t, err, "Should return error for invalid credentials")
	assert.Nil(t, client, "Client should be nil on authentication failure")
}

func TestAuthenticateClientCredentials_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New(
		sfemulator.WithCredentials(auth.Credential{
			ClientID:     "cc_client_id",
			ClientSecret: "cc_client_secret",
		}),
	)
	emu.Start()
	defer emu.Stop()

	authConfig := go_salesforce_api_client.Auth{
		ClientID:     "cc_client_id",
		ClientSecret: "cc_client_secret",
		TokenURL:     emu.URL() + "/services/oauth2/token",
	}

	client, err := authConfig.AuthenticateClientCredentials()
	require.NoError(t, err)
	assert.NotEmpty(t, client.AccessToken, "Access token should be set")
	assert.NotEmpty(t, client.InstanceURL, "Instance URL should be set")
}

func TestQuery_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	_, _ = store.CreateRecord("Account", map[string]interface{}{
		"Name":     "Acme Corporation",
		"Industry": "Technology",
	})
	_, _ = store.CreateRecord("Account", map[string]interface{}{
		"Name":     "Global Industries",
		"Industry": "Manufacturing",
	})

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	resp, err := client.Query("SELECT Id, Name, Industry FROM Account")
	require.NoError(t, err)
	assert.Equal(t, 2, resp.TotalSize, "Should return 2 records")
	assert.True(t, resp.Done, "Query should be done")
	assert.Len(t, resp.Records, 2, "Should have 2 records")

	names := make([]string, len(resp.Records))
	for i, r := range resp.Records {
		names[i] = r["Name"].(string)
		assert.NotEmpty(t, r["Id"], "Each record should have an Id")
	}
	assert.Contains(t, names, "Acme Corporation")
	assert.Contains(t, names, "Global Industries")
}

func TestQuery_WithWhereClause(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	_, _ = store.CreateRecord("Account", map[string]interface{}{
		"Name":     "Tech Corp",
		"Industry": "Technology",
	})
	_, _ = store.CreateRecord("Account", map[string]interface{}{
		"Name":     "Manufacturing Inc",
		"Industry": "Manufacturing",
	})

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	resp, err := client.Query("SELECT Id, Name FROM Account WHERE Industry = 'Technology'")
	require.NoError(t, err)
	assert.Equal(t, 1, resp.TotalSize, "Should filter to 1 record")
	require.Len(t, resp.Records, 1)
	assert.Equal(t, "Tech Corp", resp.Records[0]["Name"])
}

func TestQuery_EmptyResult(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	resp, err := client.Query("SELECT Id, Name FROM Account")
	require.NoError(t, err)
	assert.Equal(t, 0, resp.TotalSize, "Should return 0 for empty result")
	assert.Empty(t, resp.Records, "Records should be empty")
	assert.True(t, resp.Done, "Query should be done even with no results")
}

func TestCreateRecord_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	record := map[string]interface{}{
		"Name":     "New Account",
		"Industry": "Healthcare",
	}

	resp, err := client.CreateRecord("Account", record)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ID, "Created record should have an ID")
	assert.True(t, resp.Success, "Create should be successful")
	assert.Empty(t, resp.Errors, "Should have no errors")

	// Verify the record was actually created
	created, err := client.GetRecord("Account", resp.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Account", created["Name"])
	assert.Equal(t, "Healthcare", created["Industry"])
}

func TestGetRecord_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	id, _ := store.CreateRecord("Account", map[string]interface{}{
		"Name":     "Test Account",
		"Industry": "Finance",
	})

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	record, err := client.GetRecord("Account", id)
	require.NoError(t, err)
	assert.Equal(t, "Test Account", record["Name"])
	assert.Equal(t, "Finance", record["Industry"])
	assert.Equal(t, id, record["Id"], "Record ID should match")
}

func TestGetRecord_NotFound(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	record, err := client.GetRecord("Account", "001NONEXISTENT")
	require.Error(t, err, "Should return error for non-existent record")
	assert.Nil(t, record, "Record should be nil when not found")
}

func TestUpdateRecord_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	id, _ := store.CreateRecord("Account", map[string]interface{}{
		"Name":     "Original Name",
		"Industry": "Tech",
	})

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	updates := map[string]interface{}{
		"Name": "Updated Name",
	}

	err := client.UpdateRecord("Account", id, updates)
	require.NoError(t, err)

	record, err := client.GetRecord("Account", id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", record["Name"], "Name should be updated")
	assert.Equal(t, "Tech", record["Industry"], "Industry should remain unchanged")
}

func TestDeleteRecord_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	id, _ := store.CreateRecord("Account", map[string]interface{}{
		"Name": "To Be Deleted",
	})

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	// Verify record exists before deletion
	_, err := client.GetRecord("Account", id)
	require.NoError(t, err, "Record should exist before deletion")

	err = client.DeleteRecord("Account", id)
	require.NoError(t, err)

	// Verify record no longer exists
	_, err = client.GetRecord("Account", id)
	assert.Error(t, err, "Record should not exist after deletion")
}

func TestDescribeSObject_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	describe, err := client.DescribeSObject("Account")
	require.NoError(t, err)
	assert.Equal(t, "Account", describe["name"], "Object name should be Account")
	assert.NotNil(t, describe["fields"], "Should include fields metadata")
}

func TestCreateRecords_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	records := []map[string]interface{}{
		{"Name": "Bulk Account 1"},
		{"Name": "Bulk Account 2"},
		{"Name": "Bulk Account 3"},
	}

	resp, err := client.CreateRecords("Account", records)
	require.NoError(t, err)
	require.Len(t, resp, 3, "Should return 3 responses")

	createdIDs := make([]string, len(resp))
	for i, r := range resp {
		assert.True(t, r.Success, "Record %d should be successful", i)
		assert.NotEmpty(t, r.ID, "Record %d should have an ID", i)
		assert.Empty(t, r.Errors, "Record %d should have no errors", i)
		createdIDs[i] = r.ID
	}

	// Verify all records exist
	queryResp, err := client.Query("SELECT Id, Name FROM Account")
	require.NoError(t, err)
	assert.Equal(t, 3, queryResp.TotalSize, "Should have 3 records in database")
}

func TestUpdateRecords_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	id1, _ := store.CreateRecord("Account", map[string]interface{}{"Name": "Account 1"})
	id2, _ := store.CreateRecord("Account", map[string]interface{}{"Name": "Account 2"})

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	updates := []map[string]interface{}{
		{"Id": id1, "Name": "Updated Account 1"},
		{"Id": id2, "Name": "Updated Account 2"},
	}

	err := client.UpdateRecords("Account", updates)
	require.NoError(t, err)

	record1, err := client.GetRecord("Account", id1)
	require.NoError(t, err)
	assert.Equal(t, "Updated Account 1", record1["Name"])

	record2, err := client.GetRecord("Account", id2)
	require.NoError(t, err)
	assert.Equal(t, "Updated Account 2", record2["Name"])
}

func TestDeleteRecords_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	id1, _ := store.CreateRecord("Account", map[string]interface{}{"Name": "Delete Me 1"})
	id2, _ := store.CreateRecord("Account", map[string]interface{}{"Name": "Delete Me 2"})

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	// Verify records exist
	queryResp, _ := client.Query("SELECT Id FROM Account")
	assert.Equal(t, 2, queryResp.TotalSize, "Should have 2 records before deletion")

	err := client.DeleteRecords("Account", []string{id1, id2})
	require.NoError(t, err)

	queryResp, err = client.Query("SELECT Id FROM Account")
	require.NoError(t, err)
	assert.Equal(t, 0, queryResp.TotalSize, "Should have 0 records after deletion")
}

func TestCreateJobQuery_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	_, _ = store.CreateRecord("Account", map[string]interface{}{"Name": "Bulk Test Account"})

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	resp, err := client.CreateJobQuery("SELECT Id, Name FROM Account")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ID, "Job ID should be set")
	assert.Equal(t, "Account", resp.Object, "Object should be Account")
	assert.NotEmpty(t, resp.State, "State should be set")
}

func TestGetJobQuery_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	createResp, err := client.CreateJobQuery("SELECT Id FROM Account")
	require.NoError(t, err)

	resp, err := client.GetJobQuery(createResp.ID)
	require.NoError(t, err)
	assert.Equal(t, createResp.ID, resp.ID, "Job ID should match")
	assert.NotEmpty(t, resp.State, "State should be set")
	assert.Equal(t, "Account", resp.Object, "Object should match")
}

func TestAbortJobQuery_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	createResp, err := client.CreateJobQuery("SELECT Id FROM Account")
	require.NoError(t, err)

	err = client.AbortJobQuery(createResp.ID)
	require.NoError(t, err)

	jobResp, err := client.GetJobQuery(createResp.ID)
	require.NoError(t, err)
	assert.Equal(t, "Aborted", jobResp.State, "Job state should be Aborted")
}

func TestDeleteJobQuery_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	createResp, err := client.CreateJobQuery("SELECT Id FROM Account")
	require.NoError(t, err)

	err = client.DeleteJobQuery(createResp.ID)
	require.NoError(t, err)

	// Verify job is deleted by trying to get it
	_, err = client.GetJobQuery(createResp.ID)
	assert.Error(t, err, "Should return error for deleted job")
}

func TestQueryToolingAPI_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	resp, err := client.QueryToolingAPI("SELECT Id, Name FROM ApexClass")
	require.NoError(t, err)
	assert.True(t, resp.Done, "Tooling API query should be done")
	assert.NotNil(t, resp.Records, "Records should not be nil")
}

func TestGetLimits_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	limits, err := client.GetLimits()
	require.NoError(t, err)
	require.NotNil(t, limits, "Limits should not be nil")

	dailyRequests, ok := limits["DailyApiRequests"]
	require.True(t, ok, "DailyApiRequests should be present")

	dailyMap, ok := dailyRequests.(map[string]interface{})
	require.True(t, ok, "DailyApiRequests should be a map")

	_, hasMax := dailyMap["Max"]
	assert.True(t, hasMax, "Should have Max field")

	_, hasRemaining := dailyMap["Remaining"]
	assert.True(t, hasRemaining, "Should have Remaining field")
}

func TestSObjectCRUD_FullWorkflow(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	// CREATE
	createResp, err := client.CreateRecord("Account", map[string]interface{}{
		"Name":     "Workflow Test Account",
		"Industry": "Technology",
	})
	require.NoError(t, err, "Create should succeed")
	assert.True(t, createResp.Success)
	accountID := createResp.ID
	assert.NotEmpty(t, accountID)

	// READ
	record, err := client.GetRecord("Account", accountID)
	require.NoError(t, err, "Read should succeed")
	assert.Equal(t, "Workflow Test Account", record["Name"])
	assert.Equal(t, "Technology", record["Industry"])

	// UPDATE
	err = client.UpdateRecord("Account", accountID, map[string]interface{}{
		"Industry": "Finance",
	})
	require.NoError(t, err, "Update should succeed")

	record, err = client.GetRecord("Account", accountID)
	require.NoError(t, err)
	assert.Equal(t, "Finance", record["Industry"], "Industry should be updated")
	assert.Equal(t, "Workflow Test Account", record["Name"], "Name should remain unchanged")

	// QUERY
	queryResp, err := client.Query("SELECT Id, Name, Industry FROM Account WHERE Id = '" + accountID + "'")
	require.NoError(t, err, "Query should succeed")
	assert.Equal(t, 1, queryResp.TotalSize)
	assert.Equal(t, accountID, queryResp.Records[0]["Id"])

	// DELETE
	err = client.DeleteRecord("Account", accountID)
	require.NoError(t, err, "Delete should succeed")

	// VERIFY DELETION
	_, err = client.GetRecord("Account", accountID)
	assert.Error(t, err, "Record should not exist after deletion")
}

func TestContactRelationship_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	// Create parent Account
	accountResp, err := client.CreateRecord("Account", map[string]interface{}{
		"Name": "Parent Account",
	})
	require.NoError(t, err)
	assert.True(t, accountResp.Success)

	// Create child Contact with relationship
	contactResp, err := client.CreateRecord("Contact", map[string]interface{}{
		"FirstName": "John",
		"LastName":  "Doe",
		"AccountId": accountResp.ID,
	})
	require.NoError(t, err)
	assert.True(t, contactResp.Success)

	// Verify relationship
	contact, err := client.GetRecord("Contact", contactResp.ID)
	require.NoError(t, err)
	assert.Equal(t, accountResp.ID, contact["AccountId"], "Contact should be linked to Account")
	assert.Equal(t, "John", contact["FirstName"])
	assert.Equal(t, "Doe", contact["LastName"])
}

func TestQuery_WithOrderBy(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	_, _ = store.CreateRecord("Account", map[string]interface{}{"Name": "Zebra Corp"})
	_, _ = store.CreateRecord("Account", map[string]interface{}{"Name": "Alpha Inc"})
	_, _ = store.CreateRecord("Account", map[string]interface{}{"Name": "Beta LLC"})

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	resp, err := client.Query("SELECT Id, Name FROM Account ORDER BY Name ASC")
	require.NoError(t, err)
	require.Len(t, resp.Records, 3)

	assert.Equal(t, "Alpha Inc", resp.Records[0]["Name"], "First record should be Alpha Inc")
	assert.Equal(t, "Beta LLC", resp.Records[1]["Name"], "Second record should be Beta LLC")
	assert.Equal(t, "Zebra Corp", resp.Records[2]["Name"], "Third record should be Zebra Corp")
}

func TestQuery_WithLimit(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	for range 10 {
		_, _ = store.CreateRecord("Account", map[string]interface{}{"Name": "Account"})
	}

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	resp, err := client.Query("SELECT Id, Name FROM Account LIMIT 5")
	require.NoError(t, err)
	assert.Len(t, resp.Records, 5, "Should return exactly 5 records with LIMIT")
}

func TestQuery_WithLikeOperator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	store := emu.Store()
	_, _ = store.CreateRecord("Account", map[string]interface{}{"Name": "Acme Corporation"})
	_, _ = store.CreateRecord("Account", map[string]interface{}{"Name": "Acme Industries"})
	_, _ = store.CreateRecord("Account", map[string]interface{}{"Name": "Global Tech"})

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	resp, err := client.Query("SELECT Id, Name FROM Account WHERE Name LIKE 'Acme%'")
	require.NoError(t, err)
	assert.Len(t, resp.Records, 2, "Should find 2 records matching 'Acme%%'")

	for _, r := range resp.Records {
		name := r["Name"].(string)
		assert.True(t, strings.HasPrefix(name, "Acme"), "All results should start with 'Acme'")
	}
}

func TestMultipleObjectTypes_WithEmulator(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: emu.CreateTestSession(),
		InstanceURL: emu.URL(),
	}

	// Create different object types
	accountResp, err := client.CreateRecord("Account", map[string]interface{}{"Name": "Test Account"})
	require.NoError(t, err)
	assert.True(t, accountResp.Success)

	contactResp, err := client.CreateRecord("Contact", map[string]interface{}{
		"FirstName": "Jane",
		"LastName":  "Smith",
	})
	require.NoError(t, err)
	assert.True(t, contactResp.Success)

	leadResp, err := client.CreateRecord("Lead", map[string]interface{}{
		"FirstName": "Bob",
		"LastName":  "Johnson",
		"Company":   "Test Company",
	})
	require.NoError(t, err)
	assert.True(t, leadResp.Success)

	// Verify each object type is stored separately
	accountQuery, err := client.Query("SELECT Id FROM Account")
	require.NoError(t, err)
	assert.Equal(t, 1, accountQuery.TotalSize, "Should have 1 Account")

	contactQuery, err := client.Query("SELECT Id FROM Contact")
	require.NoError(t, err)
	assert.Equal(t, 1, contactQuery.TotalSize, "Should have 1 Contact")

	leadQuery, err := client.Query("SELECT Id FROM Lead")
	require.NoError(t, err)
	assert.Equal(t, 1, leadQuery.TotalSize, "Should have 1 Lead")
}

func TestQuery_MissingAuth(t *testing.T) {
	t.Parallel()

	emu := sfemulator.New()
	emu.Start()
	defer emu.Stop()

	client := &go_salesforce_api_client.Client{
		AccessToken: "",
		InstanceURL: emu.URL(),
	}

	_, err := client.Query("SELECT Id FROM Account")
	assert.Error(t, err, "Should return error when access token is missing")
}

func TestCreateRecord_MissingAuth(t *testing.T) {
	t.Parallel()

	client := &go_salesforce_api_client.Client{
		AccessToken: "",
		InstanceURL: "",
	}

	_, err := client.CreateRecord("Account", map[string]interface{}{"Name": "Test"})
	assert.Error(t, err, "Should return error when authentication is missing")
}
