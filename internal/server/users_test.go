package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
)

func TestGenerateAccessToken(t *testing.T) {
	testCases := []struct {
		name                string
		user                store.User
		expires             time.Time
		secret              string
		expectedAccessToken string
		expectError         bool
	}{
		{
			name: "normal user",
			user: store.User{
				ID:      14,
				Roles:   store.Roles{IsSuperAdmin: true, IsAdmin: true, IsVerified: true},
				RealmID: 4,
			},
			expires:             time.Unix(1558045069, 0),
			secret:              "i-am-secret",
			expectedAccessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVSb2xlcyI6eyJpc1N1cGVyQWRtaW4iOnRydWUsImlzQWRtaW4iOnRydWUsImlzVmVyaWZpZWQiOnRydWV9LCJwZXJlZ3JpbmVSZWFsbSI6NCwiZXhwIjoxNTU4MDQ1MDY5LCJzdWIiOiIxNCJ9.sAPxwUZ8PvRlLDntuysXlAc1IFIaTUzYhjPbWf_fIsE",
			expectError:         false,
		},
		{
			name:                "zero value user",
			user:                store.User{},
			expires:             time.Unix(0, 0),
			secret:              "foo",
			expectedAccessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVSb2xlcyI6eyJpc1N1cGVyQWRtaW4iOmZhbHNlLCJpc0FkbWluIjpmYWxzZSwiaXNWZXJpZmllZCI6ZmFsc2V9LCJwZXJlZ3JpbmVSZWFsbSI6MCwic3ViIjoiMCJ9.x2IaaIUlD2SSvNKUwEAXo7JhpS1azpgfeptpPK6_u2A",
			expectError:         false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actualAccessToken, err := generateAccessToken(testCase.user, testCase.expires, testCase.secret)

			if !cmp.Equal(testCase.expectedAccessToken, actualAccessToken) {
				t.Errorf("expected actual access token to match expected access token, but got diff: %s", cmp.Diff(testCase.expectedAccessToken, actualAccessToken))
			}

			if !testCase.expectError && err != nil {
				t.Errorf("did not expect error but got: %v", err)
			} else if testCase.expectError && err == nil {
				t.Errorf("expected error but didn't get one")
			}
		})
	}
}

type mockGetUserByName struct {
	user store.User
	err  error
}

func (mgu mockGetUserByName) GetUserByUsername(ctx context.Context, username string) (store.User, error) {
	return mgu.user, mgu.err
}

func TestAuthenticateHandler(t *testing.T) {
	testCases := []struct {
		name                  string
		requestUser           baseUser
		returnedUser          store.User
		returnedError         error
		secret                string
		expectedStatusCode    int
		expectedPlainResponse string
		expectedResponse      map[string]interface{}
	}{
		{
			name: "user that fails to validate",
			requestUser: baseUser{
				Username: "foo",
				Password: "foobar",
			},
			secret:             "foobar",
			expectedStatusCode: http.StatusUnprocessableEntity,
			expectedResponse:   map[string]interface{}{"error": "Key: 'baseUser.Username' Error:Field validation for 'Username' failed on the 'gte' tag\nKey: 'baseUser.Password' Error:Field validation for 'Password' failed on the 'gte' tag"},
		},
		{
			name: "user not found",
			requestUser: baseUser{
				Username: "franklin",
				Password: "password1",
			},
			secret:                "foobar",
			returnedError:         store.ErrNoResults{},
			expectedStatusCode:    http.StatusUnauthorized,
			expectedPlainResponse: http.StatusText(http.StatusUnauthorized) + "\n",
		},
		{
			name: "couldnt get user from database",
			requestUser: baseUser{
				Username: "franklin",
				Password: "password1",
			},
			secret:                "foobar",
			returnedError:         errors.New("bla"),
			expectedStatusCode:    http.StatusInternalServerError,
			expectedPlainResponse: http.StatusText(http.StatusInternalServerError) + "\n",
		},
		{
			name: "password doesnt match",
			requestUser: baseUser{
				Username: "franklin",
				Password: "password1",
			},
			secret: "foobar",
			returnedUser: store.User{
				Username:        "franklin",
				HashedPassword:  "$2a$10$sAG1oh48UCBIx8lNLT/vUu5Ppjbl.XKE92.2Z5jabYSbmJ20lgxUS",
				PasswordChanged: store.UnixTime{Unix: 1000},
			},
			expectedStatusCode:    http.StatusUnauthorized,
			expectedPlainResponse: http.StatusText(http.StatusUnauthorized) + "\n",
		},
		{
			name: "normal valid auth",
			requestUser: baseUser{
				Username: "franklin",
				Password: "password1",
			},
			secret: "foobar",
			returnedUser: store.User{
				Username:        "franklin",
				HashedPassword:  "$2a$10$L.wVyII3NNQARQVXlKwV2e9cxJltqQHdyLoybFLK3LMQ4mtsZJt9.",
				PasswordChanged: store.UnixTime{Unix: 1000},
			},
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"accessToken":  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVSb2xlcyI6eyJpc1N1cGVyQWRtaW4iOmZhbHNlLCJpc0FkbWluIjpmYWxzZSwiaXNWZXJpZmllZCI6ZmFsc2V9LCJwZXJlZ3JpbmVSZWFsbSI6MCwiZXhwIjoxNTU4MTM2OTI4LCJzdWIiOiIwIn0.X7O8HOHrhfPXpSwxIHLALZAe_y5TXxkEq7iXvUSQX0I",
				"refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVQYXNzd29yZENoYW5nZWQiOiIxOTY5LTEyLTMxVDE2OjE2OjQwLTA4OjAwIiwiZXhwIjoxNTYwNDY5NzI4LCJzdWIiOiIwIn0.QM8UQV1NyR2OnyvUnjAGYkgd61y5YpmauCwyQ1RYz98",
			},
		},
	}

	now := func() time.Time {
		return time.Unix(1558050528, 0)
	}

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			requestBuffer := new(bytes.Buffer)
			if err := json.NewEncoder(requestBuffer).Encode(testCase.requestUser); err != nil {
				t.Errorf("did not expect error %v marshaling request user", err)
				t.FailNow()
			}

			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/", requestBuffer)
			if err != nil {
				t.Errorf("did not expect error %v setting up test", err)
				t.FailNow()
			}

			handler := authenticateHandler(logger, now, mockGetUserByName{testCase.returnedUser, testCase.returnedError}, testCase.secret)

			handler(rr, req)

			if rr.Code != testCase.expectedStatusCode {
				t.Errorf("expected status code %d but got %d", testCase.expectedStatusCode, rr.Code)
			}

			if testCase.expectedPlainResponse == "" {
				var actualResponse map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&actualResponse); err != nil {
					t.Errorf("did not expect error decoding response, but got: %v", err)
				}

				if !cmp.Equal(testCase.expectedResponse, actualResponse) {
					t.Errorf("expected actual response to equal expected response, but got diff: %v", cmp.Diff(testCase.expectedResponse, actualResponse))
				}
			} else {
				plainResponse := rr.Body.String()
				if !cmp.Equal(testCase.expectedPlainResponse, plainResponse) {
					t.Errorf("expected actual response to equal expected response, but got diff: %v", cmp.Diff(testCase.expectedPlainResponse, plainResponse))
				}
			}
		})
	}
}
