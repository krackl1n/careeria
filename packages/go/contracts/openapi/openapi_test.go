package openapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSpecification(t *testing.T) {
	t.Parallel()

	var specification map[string]any
	if err := json.Unmarshal(Specification(), &specification); err != nil {
		t.Fatalf("decode specification: %v", err)
	}

	if got := specification["swagger"]; got != "2.0" {
		t.Fatalf("unexpected Swagger version: %v", got)
	}

	info, ok := specification["info"].(map[string]any)
	if !ok {
		t.Fatal("info is missing")
	}
	if got := info["title"]; got != "Careeria API" {
		t.Fatalf("unexpected API title: %v", got)
	}

	tags, ok := specification["tags"].([]any)
	if !ok {
		t.Fatal("tags are missing")
	}
	expectedTags := map[string]bool{
		"OAuth2 / OIDC":     false,
		"Authentication UI": false,
		"Identity session":  false,
		"Identity metadata": false,
		"Registration":      false,
	}
	for _, rawTag := range tags {
		tag, ok := rawTag.(map[string]any)
		if !ok {
			t.Fatalf("unexpected tag: %v", rawTag)
		}
		name, ok := tag["name"].(string)
		if !ok {
			t.Fatalf("tag name is missing: %v", tag)
		}
		if _, ok := expectedTags[name]; !ok {
			t.Errorf("unexpected generated tag %q", name)
			continue
		}
		expectedTags[name] = true
	}
	for name, found := range expectedTags {
		if !found {
			t.Errorf("tag %q is missing", name)
		}
	}

	paths, ok := specification["paths"].(map[string]any)
	if !ok {
		t.Fatal("paths are missing")
	}

	expectedOperations := map[string][]string{
		"/.well-known/jwks.json":                     {"get"},
		"/.well-known/openid-configuration":          {"get"},
		"/api/v1/authentication/consent:deny":        {"post"},
		"/api/v1/authentication/consent:grant":       {"post"},
		"/api/v1/authentication/context":             {"get"},
		"/api/v1/authentication/methods":             {"get"},
		"/api/v1/authentication/mfa/enrollment:skip": {"post"},
		"/api/v1/authentication/password":            {"post"},
		"/api/v1/authentication/verification/email":  {"post"},
		"/api/v1/authentication/verification/phone":  {"post"},
		"/api/v1/registration":                       {"post"},
		"/api/v1/session":                            {"get"},
		"/api/v1/session/logout":                     {"post"},
		"/oauth2/authorize":                          {"get"},
		"/oauth2/authorize/continue":                 {"post"},
		"/oauth2/revoke":                             {"post"},
		"/oauth2/token":                              {"post"},
		"/oauth2/userinfo":                           {"get"},
	}

	if len(paths) != len(expectedOperations) {
		t.Fatalf("unexpected number of paths: got %d, want %d", len(paths), len(expectedOperations))
	}
	for path, methods := range expectedOperations {
		pathItem, ok := paths[path].(map[string]any)
		if !ok {
			t.Errorf("path %q is missing", path)
			continue
		}
		for _, method := range methods {
			if _, ok := pathItem[method]; !ok {
				t.Errorf("operation %s %s is missing", method, path)
			}
		}
	}

	operationIDs := make(map[string]string)
	for path, rawPathItem := range paths {
		pathItem, ok := rawPathItem.(map[string]any)
		if !ok {
			t.Fatalf("unexpected path item %q: %v", path, rawPathItem)
		}
		for method, rawOperation := range pathItem {
			if method == "parameters" {
				continue
			}
			operation, ok := rawOperation.(map[string]any)
			if !ok {
				t.Errorf("unexpected operation %s %s: %v", method, path, rawOperation)
				continue
			}
			operationID, ok := operation["operationId"].(string)
			if !ok || operationID == "" {
				t.Errorf("operationId is missing for %s %s", method, path)
				continue
			}
			if previous, exists := operationIDs[operationID]; exists {
				t.Errorf("duplicate operationId %q for %s %s and %s", operationID, method, path, previous)
			}
			operationIDs[operationID] = method + " " + path

			operationTags, ok := operation["tags"].([]any)
			if !ok || len(operationTags) == 0 {
				t.Errorf("tags are missing for %s %s", method, path)
			}

			responses := mustObject(t, operation, "responses")
			defaultResponse := mustObject(t, responses, "default")
			if got := defaultResponse["$ref"]; got != "#/responses/Error" {
				t.Errorf("unexpected default error response for %s %s: %v", method, path, got)
			}
		}
	}

	responses := mustObject(t, specification, "responses")
	errorResponse := mustObject(t, responses, "Error")
	errorSchema := mustObject(t, errorResponse, "schema")
	if got := errorSchema["$ref"]; got != "#/definitions/v1ErrorResponse" {
		t.Fatalf("unexpected shared error schema: %v", got)
	}
}

func TestDocumentSchemas(t *testing.T) {
	t.Parallel()

	var specification map[string]any
	if err := json.Unmarshal(Specification(), &specification); err != nil {
		t.Fatalf("decode specification: %v", err)
	}

	definitions := mustObject(t, specification, "definitions")
	if _, ok := definitions["v1CreateDocumentRoleRequest"]; !ok {
		t.Skip("document schemas are not part of the active contracts")
	}
	if _, ok := definitions["v1VariableValue"]; ok {
		t.Fatal("VariableValue wrapper must not be exposed")
	}
	if _, ok := definitions["v1VariableSchema"]; ok {
		t.Fatal("VariableSchema wrapper must not be exposed")
	}

	roleRequest := mustObject(t, definitions, "v1CreateDocumentRoleRequest")
	roleProperties := mustObject(t, roleRequest, "properties")
	roleCode := mustObject(t, roleProperties, "code")
	if got := roleCode["maxLength"]; got != float64(64) {
		t.Fatalf("document role code maxLength: got %v, want 64", got)
	}
	isActive := mustObject(t, roleProperties, "isActive")
	if got := isActive["default"]; got != true {
		t.Fatalf("create role isActive default: got %v, want true", got)
	}

	variableRequest := mustObject(t, definitions, "v1CreateVariableDefinitionRequest")
	variableProperties := mustObject(t, variableRequest, "properties")
	if defaultValue := mustObject(t, variableProperties, "defaultValue"); defaultValue["$ref"] != nil {
		t.Fatalf("defaultValue must be a direct JSON value: %v", defaultValue)
	}
	variableSchema := mustObject(t, variableProperties, "schema")
	if got := variableSchema["type"]; got != "object" {
		t.Fatalf("variable schema must be a direct JSON object: %v", got)
	}
	sourceFile := mustObject(t, definitions, "v1SourceFileReference")
	sourceFileProperties := mustObject(t, sourceFile, "properties")
	if checksum := mustObject(t, sourceFileProperties, "checksum"); checksum["maxLength"] != float64(128) {
		t.Fatalf("source file checksum maxLength: got %v, want 128", checksum["maxLength"])
	}
	contextBinding := mustObject(t, definitions, "v1ContextBinding")
	contextBindingProperties := mustObject(t, contextBinding, "properties")
	if _, ok := contextBindingProperties["createdBy"]; !ok {
		t.Fatal("context binding createdBy read field is missing")
	}
	documentFile := mustObject(t, definitions, "v1DocumentFile")
	documentFileProperties := mustObject(t, documentFile, "properties")
	if checksum := mustObject(t, documentFileProperties, "checksum"); checksum["maxLength"] != float64(128) {
		t.Fatalf("document file checksum maxLength: got %v, want 128", checksum["maxLength"])
	}

	editorContent := mustObject(t, definitions, "v1EditorContent")
	editorProperties := mustObject(t, editorContent, "properties")
	editorData := mustObject(t, editorProperties, "data")
	if got := editorData["x-max-bytes"]; got != float64(16*1024*1024) {
		t.Fatalf("editor payload limit: got %v, want %d", got, 16*1024*1024)
	}
	editorFormat := mustObject(t, editorProperties, "format")
	if got := stringSet(editorFormat["enum"]); got["EDITOR_CONTENT_FORMAT_UNSPECIFIED"] || !got["EDITOR_CONTENT_FORMAT_JSON"] {
		t.Fatalf("editor format validation is not reflected in OpenAPI: %v", editorFormat["enum"])
	}

	createByBinding := mustObject(t, definitions, "DocumentServiceCreateDocumentByBindingBody")
	oneofGroups, ok := createByBinding["x-oneof"].([]any)
	if !ok || len(oneofGroups) != 1 {
		t.Fatalf("required document source oneof is missing: %v", createByBinding["x-oneof"])
	}
	oneof := oneofGroups[0].(map[string]any)
	if got := stringSet(oneof["fields"]); !got["templateId"] || !got["blank"] || !got["uploaded"] || len(got) != 3 {
		t.Fatalf("unexpected document source oneof fields: %v", oneof["fields"])
	}
	if got := oneof["required"]; got != true {
		t.Fatalf("document source oneof must be required: %v", got)
	}
	uploaded := mustObject(t, definitions, "v1UploadedDocumentDefinition")
	uploadedProperties := mustObject(t, uploaded, "properties")
	if got := mustObject(t, uploadedProperties, "data")["x-max-bytes"]; got != float64(16*1024*1024) {
		t.Fatalf("uploaded document payload limit: got %v, want %d", got, 16*1024*1024)
	}
	if got := stringSet(mustObject(t, uploadedProperties, "format")["enum"]); !got["DOCUMENT_FILE_FORMAT_PDF"] || !got["DOCUMENT_FILE_FORMAT_DOC"] || len(got) != 2 {
		t.Fatalf("unexpected uploaded document formats: %v", got)
	}

	preview := mustObject(t, definitions, "v1PreviewResult")
	previewProperties := mustObject(t, preview, "properties")
	if _, exists := previewProperties["data"]; exists {
		t.Fatal("preview must not embed base64 binary data")
	}
	for _, field := range []string{"fileId", "fileName", "downloadUrl", "expiresAt", "validation"} {
		if _, exists := previewProperties[field]; !exists {
			t.Errorf("preview field %q is missing", field)
		}
	}

	apiError := mustObject(t, definitions, "v1APIError")
	requiredErrorFields := stringSet(apiError["required"])
	if !requiredErrorFields["code"] || !requiredErrorFields["message"] {
		t.Fatalf("API error code and message must be required: %v", apiError["required"])
	}
	errorDetails := mustObject(t, definitions, "v1ErrorDetails")
	errorDetailsProperties := mustObject(t, errorDetails, "properties")
	fields := mustObject(t, errorDetailsProperties, "fields")
	if got := fields["maxItems"]; got != float64(1000) {
		t.Fatalf("validation error fields limit: got %v, want 1000", got)
	}
	if _, ok := errorDetailsProperties["metadata"]; !ok {
		t.Fatal("domain error metadata schema is missing")
	}
	errorEnvelope := mustObject(t, definitions, "v1ErrorResponse")
	errorEnvelopeProperties := mustObject(t, errorEnvelope, "properties")
	if _, ok := errorEnvelopeProperties["request_id"]; !ok {
		t.Fatal("error response request_id must match the API Gateway JSON envelope")
	}
	if _, ok := errorEnvelopeProperties["requestId"]; ok {
		t.Fatal("camelCase requestId does not match the API Gateway JSON envelope")
	}

	paths := mustObject(t, specification, "paths")
	templatesPath := mustObject(t, paths, "/v1/document-templates")
	createTemplate := mustObject(t, templatesPath, "post")
	description, _ := createTemplate["description"].(string)
	if !strings.Contains(description, "File Service") || !strings.Contains(description, "не поддерживает multipart/form-data") {
		t.Fatalf("template upload flow is not explicit: %q", description)
	}
}

func TestIdentitySchemasAndProtocolOperations(t *testing.T) {
	t.Parallel()

	var specification map[string]any
	if err := json.Unmarshal(Specification(), &specification); err != nil {
		t.Fatalf("decode specification: %v", err)
	}

	definitions := mustObject(t, specification, "definitions")
	if _, ok := definitions["v1CreateInvitationRequest"]; !ok {
		t.Skip("organization schemas are not part of the active contracts")
	}
	tokenRequest := mustObject(t, definitions, "v1ExchangeTokenRequest")
	tokenProperties := mustObject(t, tokenRequest, "properties")
	if _, exists := tokenProperties["scope"]; exists {
		t.Fatal("token exchange request must derive scopes from the authorization code or session")
	}
	grantType := mustObject(t, tokenProperties, "grant_type")
	grantTypes := stringSet(grantType["enum"])
	if !grantTypes["authorization_code"] || !grantTypes["refresh_token"] || len(grantTypes) != 2 {
		t.Fatalf("unexpected token grant types: %v", grantType["enum"])
	}
	codeVerifier := mustObject(t, tokenProperties, "code_verifier")
	if codeVerifier["minLength"] != float64(43) || codeVerifier["maxLength"] != float64(128) {
		t.Fatalf("PKCE verifier limits are missing: %v", codeVerifier)
	}
	tokenResponse := mustObject(t, definitions, "v1ExchangeTokenResponse")
	tokenResponseProperties := mustObject(t, tokenResponse, "properties")
	expiresIn := mustObject(t, tokenResponseProperties, "expires_in")
	if expiresIn["type"] != "integer" {
		t.Fatalf("OAuth expires_in must be a JSON integer: %v", expiresIn)
	}

	passwordRequest := mustObject(t, definitions, "v1AuthenticateWithPasswordRequest")
	oneofGroups, ok := passwordRequest["x-oneof"].([]any)
	if !ok || len(oneofGroups) != 1 {
		t.Fatalf("password login oneof is missing: %v", passwordRequest["x-oneof"])
	}
	loginOneof := oneofGroups[0].(map[string]any)
	if got := stringSet(loginOneof["fields"]); !got["email"] || !got["phoneNumber"] || len(got) != 2 {
		t.Fatalf("unexpected password login fields: %v", loginOneof["fields"])
	}
	authenticationContext := mustObject(t, definitions, "v1AuthenticationContext")
	if required := stringSet(authenticationContext["required"]); !required["nextStep"] {
		t.Fatalf("authentication context next step is not required: %v", authenticationContext["required"])
	}
	authenticationStep := mustObject(t, definitions, "v1AuthenticationStep")
	stepOneofs, ok := authenticationStep["x-oneof"].([]any)
	if !ok || len(stepOneofs) != 1 {
		t.Fatalf("authentication next step oneof is missing: %v", authenticationStep["x-oneof"])
	}
	stepOneof := stepOneofs[0].(map[string]any)
	if fields := stringSet(stepOneof["fields"]); !fields["mfaEnrollment"] {
		t.Fatalf("MFA enrollment is missing from authentication steps: %v", stepOneof["fields"])
	}
	mfaEnrollmentStep := mustObject(t, definitions, "v1MFAEnrollmentStep")
	if required := stringSet(mfaEnrollmentStep["required"]); !required["availableMethods"] {
		t.Fatalf("MFA enrollment methods are not required: %v", mfaEnrollmentStep["required"])
	}
	emailVerificationStep := mustObject(t, definitions, "v1EmailVerificationStep")
	emailVerificationProperties := mustObject(t, emailVerificationStep, "properties")
	if _, exists := emailVerificationProperties["emailId"]; exists {
		t.Fatal("email verification step must resolve its target from the authorization transaction")
	}
	if _, exists := emailVerificationProperties["challengeId"]; exists {
		t.Fatal("email verification challenge must remain internal")
	}
	if required := stringSet(emailVerificationStep["required"]); !required["maskedEmail"] || !required["challengeExpiresAt"] {
		t.Fatalf("email verification presentation data is incomplete: %v", emailVerificationStep["required"])
	}
	phoneVerificationStep := mustObject(t, definitions, "v1PhoneVerificationStep")
	phoneVerificationProperties := mustObject(t, phoneVerificationStep, "properties")
	if _, exists := phoneVerificationProperties["phoneId"]; exists {
		t.Fatal("phone verification step must resolve its target from the authorization transaction")
	}
	if _, exists := phoneVerificationProperties["challengeId"]; exists {
		t.Fatal("phone verification challenge must remain internal")
	}
	for _, name := range []string{"v1VerifyEmailRequest", "v1VerifyPhoneRequest"} {
		verificationRequest := mustObject(t, definitions, name)
		if required := stringSet(verificationRequest["required"]); !required["authorizationTransactionId"] || !required["code"] {
			t.Fatalf("%s fields are not required: %v", name, verificationRequest["required"])
		}
		verificationProperties := mustObject(t, verificationRequest, "properties")
		verificationCode := mustObject(t, verificationProperties, "code")
		if verificationCode["pattern"] != "^[0-9]{6}$" {
			t.Fatalf("%s code format is invalid: %v", name, verificationCode)
		}
	}
	registrationRequest := mustObject(t, definitions, "v1RegisterIdentityRequest")
	registrationProperties := mustObject(t, registrationRequest, "properties")
	registrationPassword := mustObject(t, registrationProperties, "password")
	if registrationPassword["x-min-bytes"] != float64(8) || registrationPassword["x-max-bytes"] != float64(1024) {
		t.Fatalf("registration password limits are missing: %v", registrationPassword)
	}
	registrationOneofs, ok := registrationRequest["x-oneof"].([]any)
	if !ok || len(registrationOneofs) != 1 {
		t.Fatalf("registration contact oneof is missing: %v", registrationRequest["x-oneof"])
	}
	contactOneof := registrationOneofs[0].(map[string]any)
	if got := stringSet(contactOneof["fields"]); !got["email"] || !got["phoneNumber"] || len(got) != 2 {
		t.Fatalf("unexpected registration contact fields: %v", contactOneof["fields"])
	}
	if required := stringSet(registrationRequest["required"]); !required["authorizationTransactionId"] || !required["password"] || !required["profile"] {
		t.Fatalf("registration authorization transaction, password and profile are not required: %v", registrationRequest["required"])
	}
	registrationResponse := mustObject(t, definitions, "v1RegisterIdentityResponse")
	registrationResponseProperties := mustObject(t, registrationResponse, "properties")
	registrationContext := mustObject(t, registrationResponseProperties, "context")
	if registrationContext["$ref"] != "#/definitions/v1AuthenticationContext" {
		t.Fatalf("registration response does not continue authentication context: %v", registrationContext)
	}
	registrationProfile := mustObject(t, definitions, "v1RegistrationProfile")
	if required := stringSet(registrationProfile["required"]); !required["familyName"] || !required["givenName"] {
		t.Fatalf("registration profile names are not required: %v", registrationProfile["required"])
	}
	profileProperties := mustObject(t, registrationProfile, "properties")
	for _, name := range []string{"familyName", "givenName", "middleName"} {
		property := mustObject(t, profileProperties, name)
		if property["maxLength"] != float64(255) {
			t.Errorf("registration profile field %q has no length limit: %v", name, property)
		}
	}

	identitySummary := mustObject(t, definitions, "v1IdentitySummary")
	if required := stringSet(identitySummary["required"]); !required["subject"] || !required["profile"] {
		t.Fatalf("identity summary does not require profile: %v", identitySummary["required"])
	}

	discovery := mustObject(t, definitions, "v1GetOpenIDConfigurationResponse")
	discoveryProperties := mustObject(t, discovery, "properties")
	responseTypes := mustObject(t, discoveryProperties, "response_types_supported")
	responseTypeItems := mustObject(t, responseTypes, "items")
	if got := stringSet(responseTypeItems["enum"]); !got["code"] || len(got) != 1 {
		t.Fatalf("discovery response type is not constrained: %v", responseTypeItems)
	}
	challengeMethods := mustObject(t, discoveryProperties, "code_challenge_methods_supported")
	challengeMethodItems := mustObject(t, challengeMethods, "items")
	if got := stringSet(challengeMethodItems["enum"]); !got["S256"] || len(got) != 1 {
		t.Fatalf("discovery PKCE method is not constrained: %v", challengeMethodItems)
	}

	userinfo := mustObject(t, definitions, "v1GetUserInfoResponse")
	userinfoProperties := mustObject(t, userinfo, "properties")
	for _, name := range []string{
		"sub",
		"name",
		"family_name",
		"given_name",
		"middle_name",
		"email_verified",
		"phone_number",
		"phone_number_verified",
		"updated_at",
	} {
		if _, ok := userinfoProperties[name]; !ok {
			t.Errorf("OIDC UserInfo field %q is missing", name)
		}
	}
	if _, ok := userinfoProperties["subject"]; ok {
		t.Fatal("UserInfo subject must be serialized as sub")
	}
	if updatedAt := mustObject(t, userinfoProperties, "updated_at"); updatedAt["type"] != "number" {
		t.Fatalf("OIDC updated_at must be a JSON number: %v", updatedAt)
	}

	paths := mustObject(t, specification, "paths")
	publicOperations := map[string]string{
		"/.well-known/jwks.json":                     "get",
		"/.well-known/openid-configuration":          "get",
		"/api/v1/authentication/context":             "get",
		"/api/v1/authentication/consent:deny":        "post",
		"/api/v1/authentication/consent:grant":       "post",
		"/api/v1/authentication/methods":             "get",
		"/api/v1/authentication/mfa/enrollment:skip": "post",
		"/api/v1/authentication/password":            "post",
		"/api/v1/authentication/verification/email":  "post",
		"/api/v1/authentication/verification/phone":  "post",
		"/api/v1/registration":                       "post",
		"/oauth2/authorize":                          "get",
		"/oauth2/authorize/continue":                 "post",
		"/oauth2/revoke":                             "post",
		"/oauth2/token":                              "post",
	}
	for path, method := range publicOperations {
		operation := mustObject(t, mustObject(t, paths, path), method)
		security, ok := operation["security"].([]any)
		if !ok || len(security) != 0 {
			t.Errorf("public operation %s %s inherits BearerAuth: %v", method, path, operation["security"])
		}
	}

	for _, path := range []string{"/oauth2/token", "/oauth2/revoke"} {
		operation := mustObject(t, mustObject(t, paths, path), "post")
		if got := stringSet(operation["consumes"]); !got["application/x-www-form-urlencoded"] {
			t.Errorf("operation POST %s does not consume form data: %v", path, operation["consumes"])
		}
		parameters, _ := operation["parameters"].([]any)
		hasBody := false
		hasFormToken := false
		for _, rawParameter := range parameters {
			parameter, _ := rawParameter.(map[string]any)
			if parameter["in"] == "body" {
				hasBody = true
			}
			if parameter["in"] == "formData" && (parameter["name"] == "grant_type" || parameter["name"] == "token") {
				hasFormToken = true
			}
		}
		if hasBody || !hasFormToken {
			t.Errorf("operation POST %s has invalid form parameters: %v", path, parameters)
		}
	}
}

func TestOrganizationSchemasAndContextHeaders(t *testing.T) {
	t.Parallel()

	var specification map[string]any
	if err := json.Unmarshal(Specification(), &specification); err != nil {
		t.Fatalf("decode specification: %v", err)
	}

	definitions := mustObject(t, specification, "definitions")
	if _, ok := definitions["v1CreateInvitationRequest"]; !ok {
		t.Skip("organization schemas are not part of the active contracts")
	}
	createInvitation := mustObject(t, definitions, "v1CreateInvitationRequest")
	createInvitationProperties := mustObject(t, createInvitation, "properties")
	presentationMethods := mustObject(t, createInvitationProperties, "presentationMethods")
	presentationItems := mustObject(t, presentationMethods, "items")
	if got := stringSet(presentationItems["enum"]); !got["INVITATION_PRESENTATION_METHOD_LINK"] || !got["INVITATION_PRESENTATION_METHOD_CODE"] || len(got) != 2 {
		t.Fatalf("unexpected invitation presentation methods: %v", presentationItems["enum"])
	}
	if required := stringSet(createInvitation["required"]); !required["presentationMethods"] || !required["expiresAt"] || !required["maxUses"] || !required["roleIds"] {
		t.Fatalf("invitation creation fields are not required: %v", createInvitation["required"])
	}

	invitation := mustObject(t, definitions, "v1OrganizationInvitation")
	invitationProperties := mustObject(t, invitation, "properties")
	if _, exists := invitationProperties["secrets"]; exists {
		t.Fatal("stored invitation must not expose secrets")
	}
	createInvitationResponse := mustObject(t, definitions, "v1CreateInvitationResponse")
	if _, exists := mustObject(t, createInvitationResponse, "properties")["secrets"]; !exists {
		t.Fatal("invitation creation response must expose one-time secrets")
	}
	previewOrganization := mustObject(t, mustObject(t, definitions, "v1InvitationOrganizationPreview"), "properties")
	if _, exists := previewOrganization["id"]; exists {
		t.Fatal("public invitation preview must not expose organization id")
	}
	if _, exists := previewOrganization["slug"]; exists {
		t.Fatal("public invitation preview must not expose organization slug")
	}
	if _, exists := previewOrganization["status"]; exists {
		t.Fatal("public invitation preview must not expose organization status")
	}
	if _, exists := previewOrganization["name"]; !exists {
		t.Fatal("public invitation preview must expose organization name")
	}

	membership := mustObject(t, definitions, "v1OrganizationMembership")
	if required := stringSet(membership["required"]); required["identityProfile"] {
		t.Fatal("identity profile projection must tolerate asynchronous delivery")
	}

	updateOrganization := mustObject(t, definitions, "v1UpdateOrganizationRequest")
	updateOneofs, ok := updateOrganization["x-oneof"].([]any)
	if !ok || len(updateOneofs) != 1 {
		t.Fatalf("organization logo update oneof is missing: %v", updateOrganization["x-oneof"])
	}
	logoOneof := updateOneofs[0].(map[string]any)
	if fields := stringSet(logoOneof["fields"]); !fields["logoFileId"] || !fields["clearLogoFile"] || len(fields) != 2 {
		t.Fatalf("unexpected organization logo update fields: %v", logoOneof["fields"])
	}
	if logoOneof["required"] != false {
		t.Fatalf("organization logo update must remain optional: %v", logoOneof)
	}

	acceptBody := mustObject(t, definitions, "OrganizationInvitationServiceAcceptInvitationBody")
	acceptOneofs, ok := acceptBody["x-oneof"].([]any)
	if !ok || len(acceptOneofs) != 1 {
		t.Fatalf("accept invitation credential oneof is missing: %v", acceptBody["x-oneof"])
	}
	credentialOneof := acceptOneofs[0].(map[string]any)
	if fields := stringSet(credentialOneof["fields"]); !fields["token"] || !fields["code"] || len(fields) != 2 || credentialOneof["required"] != true {
		t.Fatalf("unexpected invitation credential oneof: %v", credentialOneof)
	}

	paths := mustObject(t, specification, "paths")
	for _, path := range []string{
		"/api/v1/organization",
		"/api/v1/me/membership",
		"/api/v1/members",
		"/api/v1/roles",
		"/api/v1/invitations",
	} {
		pathItem := mustObject(t, paths, path)
		for method, rawOperation := range pathItem {
			operation, ok := rawOperation.(map[string]any)
			if !ok {
				continue
			}
			if !hasRequiredHeader(operation, "X-Tenant-Id") {
				t.Errorf("required tenant header is missing for %s %s", method, path)
			}
		}
	}

	for _, path := range []string{
		"/api/v1/organizations",
		"/api/v1/public/invitations/resolve-token",
		"/api/v1/public/invitations/resolve-code",
		"/api/v1/public/invitations/{invitationId}/accept",
	} {
		operation := mustObject(t, mustObject(t, paths, path), firstOperationMethod(t, mustObject(t, paths, path)))
		if hasRequiredHeader(operation, "X-Tenant-Id") {
			t.Errorf("tenant header must not be required for %s", path)
		}
	}

	for _, path := range []string{
		"/api/v1/public/invitations/resolve-token",
		"/api/v1/public/invitations/resolve-code",
	} {
		operation := mustObject(t, mustObject(t, paths, path), "post")
		security, ok := operation["security"].([]any)
		if !ok || len(security) != 0 {
			t.Errorf("public preview operation inherits BearerAuth for %s: %v", path, operation["security"])
		}
	}
	accept := mustObject(t, mustObject(t, paths, "/api/v1/public/invitations/{invitationId}/accept"), "post")
	if _, explicitlyPublic := accept["security"]; explicitlyPublic {
		t.Fatalf("invitation acceptance must inherit BearerAuth: %v", accept["security"])
	}
}

func hasRequiredHeader(operation map[string]any, name string) bool {
	parameters, _ := operation["parameters"].([]any)
	for _, rawParameter := range parameters {
		parameter, _ := rawParameter.(map[string]any)
		if parameter["in"] == "header" && parameter["name"] == name && parameter["required"] == true {
			return true
		}
	}
	return false
}

func firstOperationMethod(t *testing.T, pathItem map[string]any) string {
	t.Helper()
	for _, method := range []string{"get", "post", "put", "patch", "delete"} {
		if _, ok := pathItem[method]; ok {
			return method
		}
	}
	t.Fatal("path has no operation")
	return ""
}

func mustObject(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%q is missing or is not an object: %v", key, parent[key])
	}
	return value
}

func stringSet(value any) map[string]bool {
	result := make(map[string]bool)
	values, _ := value.([]any)
	for _, value := range values {
		if text, ok := value.(string); ok {
			result[text] = true
		}
	}
	return result
}
