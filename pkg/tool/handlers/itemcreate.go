package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/gorilla/mux"

	toolAPIs "github.com/charlieegan3/tool-webhook-rss/pkg/apis"
)

func BuildItemCreateHandler(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	goquDB := goqu.New("postgres", db)

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		feed, ok := vars["feed"]
		if !ok || feed == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("feed var missing"))
			return
		}

		if !feedRegex.MatchString(feed) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("feed didn't match regex"))
			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to read request body"))
			return
		}

		var items []toolAPIs.PayloadNewItem
		arrErr := json.NewDecoder(bytes.NewBuffer(b)).Decode(&items)
		if arrErr != nil {
			// here we handle the case where a single item is sent.
			// regrettably, the apple shortcuts app can't send arrays, so we have to handle single items here.
			var item toolAPIs.PayloadNewItem
			err := json.NewDecoder(bytes.NewBuffer(b)).Decode(&item)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("failed to parse JSON data as as item array or item object"))
				w.Write([]byte(err.Error()))
				return
			}
			items = []toolAPIs.PayloadNewItem{item}
		}

		var records []goqu.Record

		for _, item := range items {
			if item.Title == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("title can't be blank"))
				return
			}

			if len(item.Title) > 500 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("title too long"))
				return
			}

			if len(item.Body) > 100000 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("body too long"))
				return
			}

			record := goqu.Record{
				"feed":  feed,
				"title": item.Title,
				"body":  item.Body,
				"url":   item.URL,
			}

			trimmedDate := strings.TrimSpace(item.Date)
			if len(trimmedDate) == 10 {
				date, err := time.Parse("2006-01-02", trimmedDate)
				if err == nil {
					record["created_at"] = date
				}
			} else if trimmedDate != "" {
				date, err := time.Parse("2006-01-02T15:04:05-07:00", trimmedDate)
				if err == nil {
					record["created_at"] = date
				}
			}

			records = append(records, record)
		}

		ins := goquDB.Insert("webhookrss.items").Rows(records)

		_, err = ins.Executor().Exec()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
