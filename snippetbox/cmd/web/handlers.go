package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"snippetbox.otaviolemos.com/internal/models"
	"snippetbox.otaviolemos.com/internal/validator"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	snippets, err := app.snippets.Latest()
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData()
	data.Snippets = snippets

	app.render(w, http.StatusOK, "home.tmpl", data)
}

func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}

	snippet, err := app.snippets.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	data := app.newTemplateData()
	data.Snippet = snippet

	app.render(w, http.StatusOK, "view.tmpl", data)
}

func (app *application) snippetSearchGet(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	content := r.URL.Query().Get("content")
	expiresStr := r.URL.Query().Get("expires")

	var expires int
	var err error
	if expiresStr != "" {
		expires, err = strconv.Atoi(expiresStr)
		if err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	var pTitle, pContent *string
	var pExpires *int
	if title != "" {
		pTitle = &title
	}
	if content != "" {
		pContent = &content
	}
	if expires != 0 {
		pExpires = ToPointer(expires)
	}

	snippets, err := app.snippets.Search(pTitle, pContent, pExpires)
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData()
	data.Snippets = snippets
	data.Form = SnippetSearchForm{
		Title:   title,
		Content: content,
		Expires: expires,
	}
	app.render(w, http.StatusOK, "search.tmpl", data)
}

func (app *application) snippetSearchPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	title := r.PostForm.Get("title")
	content := r.PostForm.Get("content")
	expiresStr := r.PostForm.Get("expires")

	var expires int
	if expiresStr != "" {
		expires, err = strconv.Atoi(expiresStr)
		if err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	redirectURL := fmt.Sprintf("/snippet/search?title=%s&content=%s&expires=%d",
		url.QueryEscape(title),
		url.QueryEscape(content),
		expires)

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (app *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData()
	data.Form = &SnippetSearchForm{}

	app.render(w, http.StatusOK, "create.tmpl", data)
}

type SnippetCreateForm struct {
	Title   string
	Content string
	Expires int
	validator.Validator
}

type SnippetSearchForm struct {
	Title   string
	Content string
	Expires int
	validator.Validator
}

func (app *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	expires, err := strconv.Atoi(r.PostForm.Get("expires"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := SnippetCreateForm{
		Title:   r.PostForm.Get("title"),
		Content: r.PostForm.Get("content"),
		Expires: expires,
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 100), "title", "This field cannot be more than 100 characters long")
	form.CheckField(validator.NotBlank(form.Content), "content", "This field cannot be blank")
	form.CheckField(validator.PermittedInt(form.Expires, 1, 7, 365), "expires", "This field must equal 1, 7, or 365")

	if !form.Valid() {
		data := app.newTemplateData()
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "create.tmpl", data)
	}

	id, err := app.snippets.Insert(form.Title, form.Content, expires)
	if err != nil {
		app.serverError(w, err)
	}

	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", id), http.StatusSeeOther)
}

func ToPointer[T any](value T) *T {
	return &value
}
