package apiHandler

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/agenvoy/kuradb/internal/database"
	databaseHandler "github.com/agenvoy/kuradb/internal/database/handler"
	"github.com/agenvoy/kuradb/internal/openai"
	"github.com/agenvoy/kuradb/internal/utils/segmenter"
)

func Search(dbs map[string]*database.DB, embedder openai.Embedder, qCache *openai.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.GetString("db")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "db is required",
			})
			return
		}

		q := c.Query("q")
		if q == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "q is required",
			})
			return
		}

		limit := queryLimit(c)
		target := strings.ToLower(c.GetString("target"))

		var (
			keywordResults  []databaseHandler.FileRow
			semanticResults []databaseHandler.FileRow
			kwErr           error
			semErr          error
			wg              sync.WaitGroup
		)

		runKeyword := target == "" || target == "keyword"
		runSemantic := target == "" || target == "semantic"

		if runKeyword {
			wg.Add(1)
		}
		if runSemantic {
			wg.Add(1)
		}

		if runKeyword {
			go func() {
				defer wg.Done()
				db := dbs[name]
				keywords, err := segmenter.Tokenize(q)
				if err != nil {
					kwErr = err
					return
				}
				if len(keywords) == 0 {
					return
				}
				keywordResults, kwErr = databaseHandler.SearchKeyword(db, c.Request.Context(), keywords, limit)
			}()
		}

		if runSemantic {
			go func() {
				defer wg.Done()
				semanticResults, semErr = getSemantic(c.Request.Context(), dbs, name, embedder, qCache, q, limit)
			}()
		}

		wg.Wait()

		if kwErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": kwErr.Error(),
			})
			return
		}
		if semErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": semErr.Error(),
			})
			return
		}

		resp := gin.H{}
		if runKeyword {
			resp["keyword"] = group(keywordResults)
		}
		if runSemantic {
			resp["semantic"] = group(semanticResults)
		}
		c.JSON(http.StatusOK, resp)
	}
}