package server

import (
	"encoding/json"
	"log"
	"net/http"
	"rsslab/utils"
	"strconv"
)

type context struct {
	w http.ResponseWriter
	r *http.Request
}

func (c *context) VarInt(name string) (int, error) {
	n, err := strconv.Atoi(c.r.PathValue(name))
	if err != nil {
		return 0, &errBadRequest{err}
	}
	return n, nil
}

func (c *context) ParseBody(v any) error {
	err := json.NewDecoder(c.r.Body).Decode(v)
	if err != nil {
		return &errBadRequest{err}
	}
	return nil
}

func (c *context) ParseQuery(v any) error {
	if err := utils.ParseQuery(c.r.URL, v); err != nil {
		return &errBadRequest{err}
	}
	return nil
}

func (c *context) Write(bytes []byte) error {
	_, err := c.w.Write(bytes)
	if err != nil {
		log.Print(err)
	}
	return nil
}

func (c *context) JSON(v any) error {
	c.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(c.w).Encode(v)
}

func (c *context) NotFound() error {
	c.w.WriteHeader(http.StatusNotFound)
	return nil
}
