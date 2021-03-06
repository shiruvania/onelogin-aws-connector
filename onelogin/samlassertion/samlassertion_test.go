package samlassertion

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/lifull-dev/onelogin-aws-connector/onelogin"
	"github.com/lifull-dev/onelogin-aws-connector/onelogin/credentials"
)

func TestSAMLAssertion_Generate(t *testing.T) {
	type fields struct {
		config *onelogin.Config
	}
	type args struct {
		input *GenerateRequest
	}
	type response struct {
		code int
		body string
	}
	config := &onelogin.Config{
		Endpoint:     "",
		ClientToken:  "client-token",
		ClientSecret: "client-secret",
		Credentials: credentials.New(nil, &credentials.Value{
			AccessToken:      "access-token",
			RefreshToken:     "refresh-token",
			CreatedAt:        time.Now().UTC(),
			AccessExpiresAt:  time.Now().UTC().Add(time.Second),
			RefreshExpiresAt: time.Now().UTC().Add(time.Second),
		}),
	}
	request := &GenerateRequest{
		UsernameOrEmail: "username-or-email",
		Password:        "password",
		AppID:           "app-id",
		Subdomain:       "subdomain",
		IPAddress:       "ip-address",
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		req     *GenerateRequest
		res     *response
		want    *GenerateResponse
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				config: config,
			},
			args: args{
				input: request,
			},
			req: request,
			res: &response{
				code: 200,
				body: `{
					"status": {
						"type":    "success",
						"message": "Success",
						"error":   false,
						"code":    200
					},
					"data": "Base64 Encoded SAML Data"
				}`,
			},
			want: &GenerateResponse{
				Status: &GenerateResponseStatus{
					Type:    "success",
					Message: "Success",
					Error:   false,
					Code:    200,
				},
				SAML: "Base64 Encoded SAML Data",
			},
			wantErr: false,
		},
		{
			name: "MFA Required",
			fields: fields{
				config: config,
			},
			args: args{
				input: request,
			},
			req: request,
			res: &response{
				code: 400,
				body: `{
					"status": {
						"type":    "success",
						"message": "MFA is required for this user",
						"error":   false,
						"code":    200
					},
					"data": [
						{
							"state_token": "5xxx604x8xx9x694xx860173xxx3x78x3x870x56",
							"devices": [
								{
									"device_id": 666666,
									"device_type": "Google Authenticator"
								}
							],
							"callback_url": "https://api.us.onelogin.com/api/1/saml_assertion/verify_factor",
							"user": {
								"lastname": "姓",
								"username": "username",
								"email": "username@example.com",
								"firstname": "名",
								"id": 12345678
							}
						}
					]
				}`,
			},
			want: &GenerateResponse{
				Status: &GenerateResponseStatus{
					Type:    "success",
					Message: "MFA is required for this user",
					Error:   false,
					Code:    200,
				},
				Factors: []GenerateResponseFactor{
					{
						StateToken: "5xxx604x8xx9x694xx860173xxx3x78x3x870x56",
						Devices: []GenerateResponseFactorDevice{
							{
								DeviceID:   666666,
								DeviceType: "Google Authenticator",
								RequireOTPToken: true,
							},
						},
						CallbackURL: "https://api.us.onelogin.com/api/1/saml_assertion/verify_factor",
						User: &GenerateResponseFactorUser{
							LastName:  "姓",
							UserName:  "username",
							Email:     "username@example.com",
							FirstName: "名",
							ID:        12345678,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "MFA Required with OneLogin Protect",
			fields: fields{
				config: config,
			},
			args: args{
				input: request,
			},
			req: request,
			res: &response{
				code: 400,
				body: `{
					"status": {
						"type":    "success",
						"message": "MFA is required for this user",
						"error":   false,
						"code":    200
					},
					"data": [
						{
							"state_token": "5xxx604x8xx9x694xx860173xxx3x78x3x870x56",
							"devices": [
								{
									"device_id": 666666,
									"device_type": "OneLogin Protect"
								}
							],
							"callback_url": "https://api.us.onelogin.com/api/1/saml_assertion/verify_factor",
							"user": {
								"lastname": "姓",
								"username": "username",
								"email": "username@example.com",
								"firstname": "名",
								"id": 12345678
							}
						}
					]
				}`,
			},
			want: &GenerateResponse{
				Status: &GenerateResponseStatus{
					Type:    "success",
					Message: "MFA is required for this user",
					Error:   false,
					Code:    200,
				},
				Factors: []GenerateResponseFactor{
					{
						StateToken: "5xxx604x8xx9x694xx860173xxx3x78x3x870x56",
						Devices: []GenerateResponseFactorDevice{
							{
								DeviceID:   666666,
								DeviceType: "OneLogin Protect",
								RequireOTPToken: true,
							},
							{
								DeviceID:   666666,
								DeviceType: "Notify to OneLogin Protect",
								RequireOTPToken: false,
							},
						},
						CallbackURL: "https://api.us.onelogin.com/api/1/saml_assertion/verify_factor",
						User: &GenerateResponseFactorUser{
							LastName:  "姓",
							UserName:  "username",
							Email:     "username@example.com",
							FirstName: "名",
							ID:        12345678,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			fields: fields{
				config: config,
			},
			args: args{
				input: request,
			},
			req: request,
			res: &response{
				code: 200,
				body: `invalid`,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error 40x",
			fields: fields{
				config: config,
			},
			args: args{
				input: request,
			},
			req: request,
			res: &response{
				code: 400,
				body: `{
					"status": {
						"type":    "bad request",
						"message": "Authorization Information is incorrect",
						"error":   true,
						"code":    400
					}
				}`,
			},
			want:    nil,
			wantErr: true,
		},
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Errorf("%v", err)
				}
				var input GenerateRequest
				if err := json.Unmarshal(body, &input); err != nil {
					t.Errorf("%v", err)
				}
				if !reflect.DeepEqual(&input, tt.req) {
					t.Errorf("Tokens.Generate() = %#v, want %#v", &input, tt.req)
				}
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.WriteHeader(tt.res.code)
				fmt.Fprintln(w, bytes.NewBuffer([]byte(tt.res.body)))
			}))
			defer ts.Close()
			u, _ := url.Parse(ts.URL)
			endpoint := fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
			tt.fields.config.Endpoint = endpoint
			s := &SAMLAssertion{
				config:     tt.fields.config,
				HTTPClient: httpClient,
			}
			got, err := s.Generate(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SAMLAssertion.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SAMLAssertion.Generate() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSAMLAssertion_VerifyFactor(t *testing.T) {
	type fields struct {
		config *onelogin.Config
	}
	type args struct {
		input *VerifyFactorRequest
	}
	type response struct {
		code int
		body string
	}
	config := &onelogin.Config{
		Endpoint:     "",
		ClientToken:  "client-token",
		ClientSecret: "client-secret",
		Credentials: credentials.New(nil, &credentials.Value{
			AccessToken:      "access-token",
			RefreshToken:     "refresh-token",
			CreatedAt:        time.Now().UTC(),
			AccessExpiresAt:  time.Now().UTC().Add(time.Second),
			RefreshExpiresAt: time.Now().UTC().Add(time.Second),
		}),
	}
	request := &VerifyFactorRequest{
		AppID:       "app-id",
		DeviceID:    "device_id",
		StateToken:  "state_token",
		OtpToken:    "otp_token",
		DoNotNotify: false,
	}
	notifyRequest := &VerifyFactorRequest{
		AppID:       "app-id",
		DeviceID:    "device_id",
		StateToken:  "state_token",
		OtpToken:    "",
		DoNotNotify: true,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		req     *VerifyFactorRequest
		res     *response
		want    *VerifyFactorResponse
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				config: config,
			},
			args: args{
				input: request,
			},
			req: request,
			res: &response{
				code: 200,
				body: `{
					"status": {
						"type":    "success",
						"message": "Success",
						"error":   false,
						"code":    200
					},
					"data": "Base64 Encoded SAML Data"
				}`,
			},
			want: &VerifyFactorResponse{
				Status: &VerifyFactorResponseStatus{
					Type:    "success",
					Message: "Success",
					Error:   false,
					Code:    200,
				},
				SAML: "Base64 Encoded SAML Data",
			},
			wantErr: false,
		},
		{
			name: "notify success",
			fields: fields{
				config: config,
			},
			args: args{
				input: notifyRequest,
			},
			req: notifyRequest,
			res: &response{
				code: 200,
				body: `{
					"status": {
						"type":    "success",
						"message": "Success",
						"error":   false,
						"code":    200
					},
					"data": "Base64 Encoded SAML Data"
				}`,
			},
			want: &VerifyFactorResponse{
				Status: &VerifyFactorResponseStatus{
					Type:    "success",
					Message: "Success",
					Error:   false,
					Code:    200,
				},
				SAML: "Base64 Encoded SAML Data",
			},
			wantErr: false,
		},
		{
			name: "notify error",
			fields: fields{
				config: config,
			},
			args: args{
				input: notifyRequest,
			},
			req: notifyRequest,
			res: &response{
				code: 200,
				body: `{
					"status": {
						"message": "Authentication pending on OL Protect",
						"error": false,
						"type": "pending",
						"code": 200
					}
				}`,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error 40x",
			fields: fields{
				config: config,
			},
			args: args{
				input: request,
			},
			req: request,
			res: &response{
				code: 400,
				body: `{
					"status": {
						"type":    "bad request",
						"message": "Authorization Information is incorrect",
						"error":   true,
						"code":    400
					}
				}`,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid JSON",
			fields: fields{
				config: config,
			},
			args: args{
				input: request,
			},
			req: request,
			res: &response{
				code: 200,
				body: `invalid`,
			},
			want:    nil,
			wantErr: true,
		},
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Errorf("%v", err)
				}
				var input VerifyFactorRequest
				if err := json.Unmarshal(body, &input); err != nil {
					t.Errorf("%v", err)
				}
				if !reflect.DeepEqual(&input, tt.req) {
					t.Errorf("Tokens.Generate() = %#v, want %#v", &input, tt.req)
				}
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.WriteHeader(tt.res.code)
				fmt.Fprintln(w, bytes.NewBuffer([]byte(tt.res.body)))
			}))
			defer ts.Close()
			u, _ := url.Parse(ts.URL)
			endpoint := fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
			tt.fields.config.Endpoint = endpoint
			s := &SAMLAssertion{
				config:                   tt.fields.config,
				HTTPClient:               httpClient,
				verifyFactorLoopMax:      2,
				verifyFactorLoopDuration: 100,
			}
			got, err := s.VerifyFactor(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SAMLAssertion.VerifyFactor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SAMLAssertion.VerifyFactor() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
