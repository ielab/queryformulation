package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hscells/groove/eval"
	"github.com/hscells/groove/formulation"
	"github.com/hscells/groove/pipeline"
	"github.com/hscells/transmute"
	"github.com/hscells/trecresults"
	"github.com/ielab/searchrefiner"
	"gopkg.in/olivere/elastic.v7"
	"net/http"
	"strings"
)

type QueryFormulationPlugin struct {
}

type queryFormulationResponse struct {
	Query []string
}

func handleFormulationSettings(s searchrefiner.Server, c *gin.Context) {
	var pmids []string
	username := s.Perm.UserState().Username(c.Request)
	temppmids := s.Settings[username].Relevant
	for _, pmid := range temppmids {
		pmids = append(pmids, strings.TrimSpace(pmid.String()))
	}
	c.JSON(http.StatusOK, pmids)
}

func handleQueryFormulation(s searchrefiner.Server, c *gin.Context) {
	var q1Ret string
	var q2Ret string
	var pmids []string
	username := s.Perm.UserState().Username(c.Request)
	temppmids := s.Settings[username].Relevant
	if len(temppmids) == 0 {
		c.String(http.StatusInternalServerError, "No relevant PMIDs found.")
		return
	}
	lang := c.PostForm("lang")
	for _, pmid := range temppmids {
		pmids = append(pmids, strings.TrimSpace(pmid.String()))
	}
	qrels := make(map[string]*trecresults.Qrel)
	for _, pmid := range pmids {
		qrel := trecresults.Qrel{
			Topic:     "X",
			Iteration: "None",
			DocId:     pmid,
			Score:     1,
		}
		qrels[pmid] = &qrel
	}
	query := pipeline.Query{
		Topic: "X",
		Name:  "None",
		Query: nil,
	}

	esClient, err := elastic.NewSimpleClient(
		elastic.SetURL(s.Config.Services.ElasticsearchUMLSURL),
		elastic.SetBasicAuth(s.Config.Services.ElasticsearchUMLSUsername, s.Config.Services.ElasticsearchUMLSPassword))
	if err != nil {
		panic(err)
	}

	stat := s.Entrez
	population := formulation.NewPubMedSet(stat)
	optimisation := eval.F1Measure
	optionMinDocs := formulation.ObjectiveMinDocs(30)
	optionGrid := formulation.ObjectiveGrid([]float64{0.05, 0.15, 0.25}, []float64{0.01, 0.05, 0.10}, []int{1, 5, 10})
	optionQuery := formulation.ObjectiveQuery(query)
	objFormulator := formulation.NewObjectiveFormulator(s.Entrez, esClient, trecresults.QrelsFile{Qrels: map[string]trecresults.Qrels{"X": qrels}}, population, "None", "None", "cui_semantic_types.txt", s.Config.Services.MetaMapURL, optimisation, optionMinDocs, optionGrid, optionQuery)
	q1, q2, _, _, _, err := objFormulator.Derive()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if lang == "pubmed" {
		q1Ret, err = transmute.CompileCqr2PubMed(q1)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		q2Ret, err = transmute.CompileCqr2PubMed(q2)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	} else if lang == "medline" {
		q1Ret, err = transmute.CompileCqr2Medline(q1)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		q2Ret, err = transmute.CompileCqr2Medline(q2)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	}
	var strQueries = []string{q1Ret, q2Ret}
	//c.Header("Content-type", "application/json; charset=utf-8")
	//c.Header("Connection", "keep-alive")
	c.JSON(http.StatusOK, queryFormulationResponse{Query: strQueries})
}

func (QueryFormulationPlugin) Serve(s searchrefiner.Server, c *gin.Context) {
	if c.Request.Method == "POST" && c.Query("formulate") == "y" {
		handleQueryFormulation(s, c)
		return
	}
	if c.Request.Method == "GET" && c.Query("settings") == "y" {
		handleFormulationSettings(s, c)
		return
	}
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
		Title:       "AutoFormulate",
		Description: "Query formulation tool to formulate query from a set of PMIDs.",
		Author:      "ielab",
		Version:     "23.Jan.2020",
		ProjectURL:  "ielab.io/searchrefiner",
	}
}

var Queryformulation = QueryFormulationPlugin{}
