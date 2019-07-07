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
	jwt "github.com/dgrijalva/jwt-go"
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

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actualAccessToken, err := generateAccessToken(tt.user, tt.expires, tt.secret)

			if !cmp.Equal(tt.expectedAccessToken, actualAccessToken) {
				t.Errorf("expected actual access token to match expected access token, but got diff: %s", cmp.Diff(tt.expectedAccessToken, actualAccessToken))
			}

			if !tt.expectError && err != nil {
				t.Errorf("did not expect error but got: %v", err)
			} else if tt.expectError && err == nil {
				t.Errorf("expected error but didn't get one")
			}
		})
	}
}

type mockGetUserByName struct {
	user     store.User
	err      error
	username string
}

func (mgu *mockGetUserByName) GetUserByUsername(ctx context.Context, username string) (store.User, error) {
	mgu.username = username
	return mgu.user, mgu.err
}

func TestAuthenticateHandler(t *testing.T) {
	testCases := []struct {
		name                  string
		requestUser           baseUser
		returnedUser          store.User
		returnedError         error
		secret                string
		expectedUsername      string // expected username passed to the mock store (tests that it was called and with the right params)
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
			expectedUsername:      "franklin",
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
			expectedUsername:      "franklin",
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
				PasswordChanged: time.Unix(1558054459, 0),
			},
			expectedUsername:      "franklin",
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
				PasswordChanged: time.Unix(1558054459, 0),
			},
			expectedUsername:   "franklin",
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"accessToken":  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVSb2xlcyI6eyJpc1N1cGVyQWRtaW4iOmZhbHNlLCJpc0FkbWluIjpmYWxzZSwiaXNWZXJpZmllZCI6ZmFsc2V9LCJwZXJlZ3JpbmVSZWFsbSI6MCwiZXhwIjoxNTU4MTM2OTI4LCJzdWIiOiIwIn0.X7O8HOHrhfPXpSwxIHLALZAe_y5TXxkEq7iXvUSQX0I",
				"refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVQYXNzd29yZENoYW5nZWQiOjE1NTgwNTQ0NTksImV4cCI6MTU2MDQ2OTcyOCwic3ViIjoiMCJ9.vJGdYkRAU8tOaGnuywHSsDpW5UnypmE4BBhaP_MpjE0",
			},
		},
	}

	mockNow := func() time.Time {
		return time.Unix(1558050528, 0)
	}

	jwt.TimeFunc = mockNow // hate this

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			requestBuffer := new(bytes.Buffer)
			if err := json.NewEncoder(requestBuffer).Encode(tt.requestUser); err != nil {
				t.Errorf("did not expect error %v marshaling request user", err)
				t.FailNow()
			}

			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/", requestBuffer)
			if err != nil {
				t.Errorf("did not expect error %v setting up test", err)
				t.FailNow()
			}

			mgu := &mockGetUserByName{user: tt.returnedUser, err: tt.returnedError}
			handler := authenticateHandler(logger, mockNow, mgu, tt.secret)

			handler(rr, req)

			if mgu.username != tt.expectedUsername {
				t.Errorf("expected username %s but got username %s", tt.expectedUsername, mgu.username)
			}

			if rr.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d but got %d", tt.expectedStatusCode, rr.Code)
			}

			if tt.expectedPlainResponse == "" {
				var actualResponse map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&actualResponse); err != nil {
					t.Errorf("did not expect error decoding response, but got: %v", err)
				}

				if !cmp.Equal(tt.expectedResponse, actualResponse) {
					t.Errorf("expected actual response to equal expected response, but got diff: %v", cmp.Diff(tt.expectedResponse, actualResponse))
				}
			} else {
				plainResponse := rr.Body.String()
				if !cmp.Equal(tt.expectedPlainResponse, plainResponse) {
					t.Errorf("expected actual response to equal expected response, but got diff: %v", cmp.Diff(tt.expectedPlainResponse, plainResponse))
				}
			}
		})
	}
}

type mockGetUserByID struct {
	user store.User
	err  error
	id   int64
}

func (mgu *mockGetUserByID) GetUserByID(ctx context.Context, id int64) (store.User, error) {
	mgu.id = id
	return mgu.user, mgu.err
}

func TestRefreshHandler(t *testing.T) {
	testCases := []struct {
		name                  string
		requestRefreshToken   string
		returnedUser          store.User
		returnedError         error
		secret                string // expected ID passed to the mock store (tests that it was called and with the right params)
		expectedID            int64
		expectedStatusCode    int
		expectedPlainResponse string
		expectedResponse      map[string]interface{}
	}{
		{
			name:                "normal refresh token",
			requestRefreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVQYXNzd29yZENoYW5nZWQiOjE1NTgwNTQ0NTksImV4cCI6MTU2MDQ2OTcyOCwic3ViIjoiMCJ9.vJGdYkRAU8tOaGnuywHSsDpW5UnypmE4BBhaP_MpjE0",
			returnedUser: store.User{
				Username:        "fharding1",
				PasswordChanged: time.Unix(1558054459, 0),
				ID:              6,
			},
			secret:             "foobar",
			expectedID:         6,
			expectedStatusCode: http.StatusOK,
			expectedResponse: map[string]interface{}{
				"accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVSb2xlcyI6eyJpc1N1cGVyQWRtaW4iOmZhbHNlLCJpc0FkbWluIjpmYWxzZSwiaXNWZXJpZmllZCI6ZmFsc2V9LCJwZXJlZ3JpbmVSZWFsbSI6MCwiZXhwIjoxNTU4NDAxODEwLCJzdWIiOiI2In0.nMepjyDJQvKQiRPCfisXLTYS4lrQJlJJIxlDD2StrQY",
			},
		},
		{
			name:                "password has changed",
			requestRefreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVQYXNzd29yZENoYW5nZWQiOjE1NTgwNTQ0NTksImV4cCI6MTU2MDQ2OTcyOCwic3ViIjoiMCJ9.vJGdYkRAU8tOaGnuywHSsDpW5UnypmE4BBhaP_MpjE0",
			returnedUser: store.User{
				Username:        "fharding1",
				PasswordChanged: time.Unix(1558054490, 0),
				ID:              6,
			},
			secret:                "foobar",
			expectedID:            6,
			expectedPlainResponse: http.StatusText(http.StatusForbidden) + "\n",
			expectedStatusCode:    http.StatusForbidden,
		},
		{
			name:                "invalid signature",
			requestRefreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVQYXNzd29yZENoYW5nZWQiOjE1NTgwNTQ0NTksImV4cCI6MTU2MDQ2OTcyOCwic3ViIjoiMCJ9.vJGdYkRAU8tOaGnuywHSsDpW5UnypmE4BBhaP0",
			returnedUser: store.User{
				Username:        "fharding1",
				PasswordChanged: time.Unix(1558054490, 0),
				ID:              6,
			},
			secret:                "foobar",
			expectedID:            6,
			expectedPlainResponse: http.StatusText(http.StatusUnauthorized) + "\n",
			expectedStatusCode:    http.StatusUnauthorized,
		},
		{
			name:                "none signing method",
			requestRefreshToken: "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0=.eyJwZXJlZ3JpbmVQYXNzd29yZENoYW5nZWQiOjE1NTgwNTQ0NTksImV4cCI6MTU2MDQ2OTcyOCwic3ViIjoiMCIsImJzaWRlc19pc19hZG1pbiI6dHJ1ZX0=.",
			returnedUser: store.User{
				Username:        "fharding1",
				PasswordChanged: time.Unix(1558054490, 0),
				ID:              6,
			},
			secret:                "foobar",
			expectedID:            6,
			expectedPlainResponse: http.StatusText(http.StatusUnauthorized) + "\n",
			expectedStatusCode:    http.StatusUnauthorized,
		},
		{
			name:                  "could not fetch user from store",
			requestRefreshToken:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVQYXNzd29yZENoYW5nZWQiOjE1NTgwNTQ0NTksImV4cCI6MTU2MDQ2OTcyOCwic3ViIjoiMCJ9.vJGdYkRAU8tOaGnuywHSsDpW5UnypmE4BBhaP_MpjE0",
			secret:                "foobar",
			returnedError:         errors.New("foo"),
			expectedID:            6,
			expectedPlainResponse: http.StatusText(http.StatusInternalServerError) + "\n",
			expectedStatusCode:    http.StatusInternalServerError,
		},
		{
			name:                  "user not found",
			requestRefreshToken:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJlZ3JpbmVQYXNzd29yZENoYW5nZWQiOjE1NTgwNTQ0NTksImV4cCI6MTU2MDQ2OTcyOCwic3ViIjoiMCJ9.vJGdYkRAU8tOaGnuywHSsDpW5UnypmE4BBhaP_MpjE0",
			returnedError:         store.ErrNoResults{},
			secret:                "foobar",
			expectedID:            6,
			expectedPlainResponse: http.StatusText(http.StatusUnauthorized) + "\n",
			expectedStatusCode:    http.StatusUnauthorized,
		},
	}

	mockNow := func() time.Time {
		return time.Unix(1558315410, 0)
	}

	jwt.TimeFunc = mockNow // hate this

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			requestBuffer := new(bytes.Buffer)
			if err := json.NewEncoder(requestBuffer).Encode(refreshRequest{tt.requestRefreshToken}); err != nil {
				t.Errorf("did not expect error %v marshaling refresh request", err)
				t.FailNow()
			}

			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/", requestBuffer)
			if err != nil {
				t.Errorf("did not expect error %v setting up test", err)
				t.FailNow()
			}

			mgu := &mockGetUserByID{err: tt.returnedError, user: tt.returnedUser}
			handler := refreshHandler(logger, mockNow, mgu, tt.secret)

			handler(rr, req)

			if rr.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d but got %d", tt.expectedStatusCode, rr.Code)
			}

			if tt.expectedPlainResponse == "" {
				var actualResponse map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&actualResponse); err != nil {
					t.Errorf("did not expect error decoding response, but got: %v", err)
				}

				if !cmp.Equal(tt.expectedResponse, actualResponse) {
					t.Errorf("expected actual response to equal expected response, but got diff: %v", cmp.Diff(tt.expectedResponse, actualResponse))
				}
			} else {
				plainResponse := rr.Body.String()
				if !cmp.Equal(tt.expectedPlainResponse, plainResponse) {
					t.Errorf("expected actual response to equal expected response, but got diff: %v", cmp.Diff(tt.expectedPlainResponse, plainResponse))
				}
			}

		})
	}
}
