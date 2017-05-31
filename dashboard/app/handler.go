// Copyright 2017 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

// +build appengine

package dash

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/user"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
)

type contextHandler func(c context.Context, w http.ResponseWriter, r *http.Request) error

func handlerWrapper(fn contextHandler) http.Handler {
	return handleContext(handleAuth(handleNamespace(fn)))
}

func handleContext(fn contextHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		if err := fn(c, w, r); err != nil {
			log.Errorf(c, "%v", err)
			if err1 := templates.ExecuteTemplate(w, "error.html", err.Error()); err1 != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	})
}

func handleAuth(fn contextHandler) contextHandler {
	return func(c context.Context, w http.ResponseWriter, r *http.Request) error {
		u := user.Current(c)
		if !u.Admin && (u.AuthDomain != "gmail.com" || !strings.HasSuffix(u.Email, "@google.com")) {
			log.Errorf(c, "Error: unauthorized user: domain='%v' email='%v'", u.AuthDomain, u.Email)
			return fmt.Errorf("%v is not authorized to view this. This incident will be reported.", u.Email)
		}
		return fn(c, w, r)
	}
}

type namespaceKey struct{}

func handleNamespace(fn contextHandler) contextHandler {
	return func(c context.Context, w http.ResponseWriter, r *http.Request) error {
		namespaces, err := datastoreNamespaces(c)
		if err != nil {
			return err
		}
		if len(namespaces) == 0 {
			return fmt.Errorf("no namespaces found")
		}
		ns := r.FormValue("namespace")
		if ns == "" {
			cookie, err := r.Cookie("namespace")
			if err == nil {
				ns = cookie.Value
			}
		}
		found := false
		for _, ns1 := range namespaces {
			if ns == ns1 {
				found = true
				break
			}
		}
		if !found {
			ns = namespaces[0]
		}
		http.SetCookie(w, &http.Cookie{
			Name:  "namespace",
			Value: ns,
		})
		nsc, err := appengine.Namespace(c, ns)
		if err != nil {
			return nil
		}
		nsc = context.WithValue(nsc, namespaceKey{}, ns)
		return fn(nsc, w, r)
	}
}

func datastoreNamespaces(c context.Context) ([]string, error) {
	namespaces, err := datastore.Namespaces(c)
	if err != nil {
		return nil, err
	}
	sort.Strings(namespaces)
	return namespaces, nil
}

func datastoreNamespace(c context.Context) (string, error) {
	ns, ok := c.Value(namespaceKey{}).(string)
	if !ok {
		return "", fmt.Errorf("missing namespace context key")
	}
	return ns, nil
}

type uiHeader struct {
	Namespace  string
	Namespaces []string
}

func commonHeader(c context.Context) (*uiHeader, error) {
	namespaces, err := datastoreNamespaces(c)
	if err != nil {
		return nil, err
	}
	ns, err := datastoreNamespace(c)
	if err != nil {
		return nil, err
	}
	h := &uiHeader{
		Namespace:  ns,
		Namespaces: namespaces,
	}
	return h, nil
}

var (
	templates = template.Must(template.New("").Funcs(templateFuncs).ParseGlob("*.html"))

	templateFuncs = template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 02 15:04")
		},
	}
)
