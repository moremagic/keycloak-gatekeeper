/*
Copyright 2015 All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestEntrypointHandlerSecure(t *testing.T) {
	proxy := newFakeKeycloakProxyWithResources(t, []*Resource{
		{
			URL:         "/admin/white_listed",
			WhiteListed: true,
		},
		{
			URL:     "/admin",
			Methods: []string{"ANY"},
		},
		{
			URL:          "/",
			Methods:      []string{"POST"},
			RolesAllowed: []string{"test"},
		},
	})

	handler := proxy.entrypointHandler()

	tests := []struct {
		Context *gin.Context
		Secure  bool
	}{
		{Context: newFakeGinContext("GET", "/")},
		{Context: newFakeGinContext("GET", "/admin"), Secure: true},
		{Context: newFakeGinContext("GET", "/admin/white_listed")},
		{Context: newFakeGinContext("GET", "/admin/white"), Secure: true},
		{Context: newFakeGinContext("GET", "/not_secure")},
		{Context: newFakeGinContext("POST", "/"), Secure: true},
	}

	for i, c := range tests {
		handler(c.Context)
		_, found := c.Context.Get(cxEnforce)
		if c.Secure && !found {
			t.Errorf("test case %d should have been set secure", i)
		}
		if !c.Secure && found {
			t.Errorf("test case %d should not have been set secure", i)
		}
	}
}

func TestEntrypointMethods(t *testing.T) {
	proxy := newFakeKeycloakProxyWithResources(t, []*Resource{
		{
			URL:     "/u0",
			Methods: []string{"GET", "POST"},
		},
		{
			URL:     "/u1",
			Methods: []string{"ANY"},
		},
		{
			URL:     "/u2",
			Methods: []string{"POST", "PUT"},
		},
	})

	handler := proxy.entrypointHandler()

	tests := []struct {
		Context *gin.Context
		Secure  bool
	}{
		{Context: newFakeGinContext("GET", "/u0"), Secure: true},
		{Context: newFakeGinContext("POST", "/u0"), Secure: true},
		{Context: newFakeGinContext("PUT", "/u0"), Secure: false},
		{Context: newFakeGinContext("GET", "/u1"), Secure: true},
		{Context: newFakeGinContext("POST", "/u1"), Secure: true},
		{Context: newFakeGinContext("PATCH", "/u1"), Secure: true},
		{Context: newFakeGinContext("POST", "/u2"), Secure: true},
		{Context: newFakeGinContext("PUT", "/u2"), Secure: true},
		{Context: newFakeGinContext("DELETE", "/u2"), Secure: false},
	}

	for i, c := range tests {
		handler(c.Context)
		_, found := c.Context.Get(cxEnforce)
		if c.Secure && !found {
			t.Errorf("test case %d should have been set secure", i)
		}
		if !c.Secure && found {
			t.Errorf("test case %d should not have been set secure", i)
		}
	}
}

func TestEntrypointWhiteListing(t *testing.T) {
	proxy := newFakeKeycloakProxyWithResources(t, []*Resource{
		{
			URL:         "/admin/white_listed",
			WhiteListed: true,
		},
		{
			URL:     "/admin",
			Methods: []string{"ANY"},
		},
	})
	handler := proxy.entrypointHandler()

	tests := []struct {
		Context *gin.Context
		Secure  bool
	}{
		{Context: newFakeGinContext("GET", "/")},
		{Context: newFakeGinContext("GET", "/admin"), Secure: true},
		{Context: newFakeGinContext("GET", "/admin/white_listed")},
	}

	for i, c := range tests {
		handler(c.Context)
		_, found := c.Context.Get(cxEnforce)
		if c.Secure && !found {
			t.Errorf("test case %d should have been set secure", i)
		}
		if !c.Secure && found {
			t.Errorf("test case %d should not have been set secure", i)
		}
	}

}

func TestEntrypointHandler(t *testing.T) {
	proxy := newFakeKeycloakProxy(t)
	handler := proxy.entrypointHandler()

	tests := []struct {
		Context *gin.Context
		Secure  bool
	}{
		{Context: newFakeGinContext("GET", fakeAdminRoleURL), Secure: true},
		{Context: newFakeGinContext("GET", fakeAdminRoleURL+"/sso"), Secure: true},
		{Context: newFakeGinContext("GET", fakeAdminRoleURL+"/../sso"), Secure: true},
		{Context: newFakeGinContext("GET", "/not_secure")},
		{Context: newFakeGinContext("GET", fakeTestWhitelistedURL)},
		{Context: newFakeGinContext("GET", oauthURL)},
		{Context: newFakeGinContext("GET", faketestListenOrdered), Secure: true},
	}

	for i, c := range tests {
		handler(c.Context)
		_, found := c.Context.Get(cxEnforce)
		if c.Secure && !found {
			t.Errorf("test case %d should have been set secure", i)
		}
		if !c.Secure && found {
			t.Errorf("test case %d should not have been set secure", i)
		}
	}
}

func TestAdmissionHandlerRoles(t *testing.T) {
	proxy := newFakeKeycloakProxyWithResources(t, []*Resource{
		{
			URL:          "/admin",
			Methods:      []string{"ANY"},
			RolesAllowed: []string{"admin"},
		},
		{
			URL:          "/test",
			Methods:      []string{"GET"},
			RolesAllowed: []string{"test"},
		},
		{
			URL:          "/either",
			Methods:      []string{"ANY"},
			RolesAllowed: []string{"admin", "test"},
		},
		{
			URL:     "/",
			Methods: []string{"ANY"},
		},
	})
	handler := proxy.admissionHandler()

	tests := []struct {
		Context     *gin.Context
		UserContext *userContext
		HTTPCode    int
	}{
		{
			Context:     newFakeGinContext("GET", "/admin"),
			UserContext: &userContext{},
			HTTPCode:    http.StatusForbidden,
		},
		{
			Context:  newFakeGinContext("GET", "/admin"),
			HTTPCode: http.StatusOK,
			UserContext: &userContext{
				roles: []string{"admin"},
			},
		},
		{
			Context:  newFakeGinContext("GET", "/test"),
			HTTPCode: http.StatusOK,
			UserContext: &userContext{
				roles: []string{"test"},
			},
		},
		{
			Context:  newFakeGinContext("GET", "/either"),
			HTTPCode: http.StatusOK,
			UserContext: &userContext{
				roles: []string{"test", "admin"},
			},
		},
		{
			Context:  newFakeGinContext("GET", "/either"),
			HTTPCode: http.StatusForbidden,
			UserContext: &userContext{
				roles: []string{"no_roles"},
			},
		},
		{
			Context:     newFakeGinContext("GET", "/"),
			HTTPCode:    http.StatusOK,
			UserContext: &userContext{},
		},
	}

	for i, c := range tests {
		// step: find the resource and inject into the context
		for _, r := range proxy.config.Resources {
			if strings.HasPrefix(c.Context.Request.RequestURI, r.URL) {
				c.Context.Set(cxEnforce, r)
				break
			}
		}
		if _, found := c.Context.Get(cxEnforce); !found {
			t.Errorf("test case %d unable to find a resource for context", i)
			continue
		}

		c.Context.Set(userContextName, c.UserContext)

		handler(c.Context)
		if c.Context.Writer.Status() != c.HTTPCode {
			t.Errorf("test case %d should have recieved code: %d, got %d", i, c.HTTPCode, c.Context.Writer.Status())
		}
	}
}

func TestSecurityHandler(t *testing.T) {
	kc := newFakeKeycloakProxy(t)
	handler := kc.securityHandler()
	context := newFakeGinContext("GET", "/")
	handler(context)
	if context.Writer.Status() != http.StatusOK {
		t.Errorf("we should have received a 200")
	}

	kc = newFakeKeycloakProxy(t)
	kc.config.Hostnames = []string{"127.0.0.1"}
	handler = kc.securityHandler()
	handler(context)
	if context.Writer.Status() != http.StatusOK {
		t.Errorf("we should have received a 200 not %d", context.Writer.Status())
	}

	kc = newFakeKeycloakProxy(t)
	kc.config.Hostnames = []string{"127.0.0.2"}
	handler = kc.securityHandler()
	handler(context)
	handler(context)
	if context.Writer.Status() != http.StatusInternalServerError {
		t.Errorf("we should have received a 500 not %d", context.Writer.Status())
	}
}

func TestHealthHandler(t *testing.T) {
	proxy := newFakeKeycloakProxy(t)
	context := newFakeGinContext("GET", healthURL)
	proxy.healthHandler(context)
	if context.Writer.Status() != http.StatusOK {
		t.Errorf("we should have recieved a 200 response")
	}
}
