package main

import (
	"github.com/gin-gonic/gin"
	"github.com/ielab/searchrefiner"
	"net/http"
)

type QueryFormulationPlugin struct {
}

func (QueryFormulationPlugin) Serve(s searchrefiner.Server, c *gin.Context) {
	rawQuery := c.PostForm("query")
	lang := c.PostForm("lang")
	c.Render(http.StatusOK, searchrefiner.RenderPlugin(searchrefiner.TemplatePlugin("plugin/queryformulation/index.html"), struct {
		searchrefiner.Query
		View string
	}{searchrefiner.Query{QueryString: rawQuery, Language: lang}, c.Query("view")}))
	return
}

func (QueryFormulationPlugin) PermissionType() searchrefiner.PluginPermission {
	return searchrefiner.PluginUser
}

func (QueryFormulationPlugin) Details() searchrefiner.PluginDetails {
	return searchrefiner.PluginDetails{
		Title:       "Query Formulation",
		Description: "Query formulation tool to formulate query from general input.",
		Author:      "ielab",
		Version:     "8.Jan.2020",
		ProjectURL:  "ielab.io/searchrefiner",
	}
}

var Queryformulation = QueryFormulationPlugin{}
