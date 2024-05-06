package rest_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/rest"
)

func TestParseDeepObjectQuery(t *testing.T) {
	deepQueryURL, err := url.ParseRequestURI("/path?sort=name+asc&person[name]=john&person[age]=30")
	require.NoError(t, err)

	ctx := &gin.Context{
		Request: &http.Request{
			URL: deepQueryURL,
		},
	}
	personMap := rest.ParseQueryDeepObject(ctx)
	require.Equal(t, map[string]any{
		"person": map[string]string{
			"name": "john",
			"age":  "30",
		},
		"sort": "name asc",
	}, personMap)
}

func TestBindDeepObjectQuery(t *testing.T) {
	deepQueryURL, err := url.ParseRequestURI("/path?sort=name+asc&person[name]=john&person[age]=30")
	require.NoError(t, err)

	ctx := &gin.Context{
		Request: &http.Request{
			URL: deepQueryURL,
		},
	}
	var QueryParams struct {
		Person struct {
			Name string
			Age  int
		}
		Sort string
	}

	err = rest.BindDeepObjectQuery(ctx, &QueryParams)
	require.NoError(t, err)
	person := QueryParams.Person
	require.Equal(t, "john", person.Name)
	require.Equal(t, 30, person.Age)
	require.Equal(t, "name asc", QueryParams.Sort)
}
