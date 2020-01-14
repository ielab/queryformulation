package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hscells/groove/eval"
	"github.com/hscells/groove/formulation"
	"github.com/hscells/groove/pipeline"
	"github.com/hscells/transmute"
	"github.com/hscells/trecresults"
	"github.com/ielab/searchrefiner"
	"net/http"
	"strings"
)

type QueryFormulationPlugin struct {
}

type queryFormulationResponse struct {
	Query 			[]string
}

func handleQueryFormulation(s searchrefiner.Server, c *gin.Context) {
	var q1Ret string
	var q2Ret string
	seedIDs := c.PostForm("seeds")
	lang := c.PostForm("lang")
	pmids := strings.Split(seedIDs, ",")
	for _, pmid := range pmids {
		pmid = strings.TrimSpace(pmid)
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
	stat := s.Entrez
	population := formulation.NewPubMedSet(stat)
	optimisation := eval.F1Measure
	optionMinDocs := formulation.ObjectiveMinDocs(30)
	optionGrid := formulation.ObjectiveGrid([]float64{0.05, 0.10, 0.15, 0.20, 0.25, 0.30},[]float64{0.001, 0.01, 0.02, 0.05, 0.10, 0.20},[]int{1, 5, 10, 15, 20, 25})
	objFormulator := formulation.NewObjectiveFormulator(query, stat, qrels, population, "None", "None", "cui_semantic_types.txt", "http://ielab-metamap.uqcloud.net", optimisation, optionMinDocs, optionGrid)
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
	c.Header("Content-type", "application/json; charset=utf-8")
	c.Header("Connection", "keep-alive")
	c.JSON(http.StatusOK, queryFormulationResponse{Query: strQueries})
}

func (QueryFormulationPlugin) Serve(s searchrefiner.Server, c *gin.Context) {
	if c.Request.Method == "POST" && c.Query("formulate") == "y" {
		handleQueryFormulation(s, c)
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
		Title:       "Query Formulation",
		Description: "Query formulation tool to formulate query from general input.",
		Author:      "ielab",
		Version:     "8.Jan.2020",
		ProjectURL:  "ielab.io/searchrefiner",
	}
}

var Queryformulation = QueryFormulationPlugin{}
