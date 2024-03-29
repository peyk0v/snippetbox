package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/NPeykov/snippetbox/internal/models"
	"github.com/NPeykov/snippetbox/internal/validator"
	"github.com/julienschmidt/httprouter"
)

type snippetCreateForm struct {
    Title string        `form:"title"`
    Content string      `form:"content"`
    Expires int         `form:"expires"`
    validator.Validator `form:"-"`
}

type userSignupForm struct {
    Name     string     `form:"name"`
    Email    string     `form:"email"`
    Password string     `form:"password"`
    validator.Validator `form:"-"`
}

type userLoginForm struct {
    Email    string     `form:"email"`
    Password string     `form:"password"`
    validator.Validator `form:"-"`
}

type userUpdatePassword struct {
    CurrentPassword string     `form:"current_password"`
    NewPassword string     `form:"new_password"`
    ConfirmNewPassword string `form:"confirm_password"`
    validator.Validator `form:"-"`
}

func ping(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("OK"))
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
    snippets, err := app.snippets.Latest()

    if err != nil {
        app.serverError(w, err)
        return
    }

    tmplData := app.newTemplateData(r)
    tmplData.Snippets = snippets

    app.render(w, http.StatusOK, "home.tmpl.html", tmplData)
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
    tmplData := app.newTemplateData(r)
    app.render(w, http.StatusOK, "about.tmpl.html", tmplData)
}

func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {
    params := httprouter.ParamsFromContext(r.Context())

    id, err := strconv.Atoi(params.ByName("id"))
    if err != nil || id < 0 {
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

    tmplData := app.newTemplateData(r)
    tmplData.Snippet = snippet

    app.render(w, http.StatusOK, "view.tmpl.html", tmplData)
}

func (app *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
    data := app.newTemplateData(r)
    data.Form = snippetCreateForm { 
        Expires: 7,
    }
    app.render(w, http.StatusOK, "create.tmpl.html", data)
}

func (app *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    var form snippetCreateForm

    err = app.decodePostForm(r, &form)

    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be empty")
    form.CheckField(validator.MaxChars(form.Title, 100), "title", "This field cannot be more than 100 characters long")
    form.CheckField(validator.NotBlank(form.Content), "content", "This field cannot be empty")
    form.CheckField(validator.PermittedValue(form.Expires, 1, 7, 365), "expires", "This field must be 1, 7 or 365")

    if !form.Valid() {
        data := app.newTemplateData(r)
        data.Form = form
        app.render(w, http.StatusUnprocessableEntity, "create.tmpl.html", data)
        return
    }

    id, err := app.snippets.Insert(form.Title, form.Content, form.Expires)
    if err != nil {
        app.serverError(w, err)
        return
    }

    app.sessionManager.Put(r.Context(), "flash", "Snippet created successfully!")
    http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", id), http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
    data := app.newTemplateData(r)
    data.Form = userLoginForm{}
    app.render(w, http.StatusOK, "login.tmpl.html", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
    var form userLoginForm

    err := app.decodePostForm(r, &form)
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be empty")
    form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be empty")
    form.CheckField(validator.Matches(form.Email, validator.EmailRegex), "email", "This email is not valid")

    if !form.Valid() {
        data := app.newTemplateData(r)
        data.Form = form
        app.render(w, http.StatusUnprocessableEntity, "login.tmpl.html", data)
        return
    }

    id, err := app.users.Authenticate(form.Email, form.Password)

    if err != nil {
        if errors.Is(err, models.ErrInvalidCredentials) {
            form.AddNonFieldError("Invalid credentials")
            data := app.newTemplateData(r)
            data.Form = form
            app.render(w, http.StatusUnprocessableEntity, "login.tmpl.html", data)
            return
        }
        app.serverError(w, err)
        return
    }

    err = app.sessionManager.RenewToken(r.Context())

    if err != nil {
        app.serverError(w, err)
        return
    }

    app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

    urlToRedirect := app.sessionManager.GetString(r.Context(), "urlToRedirectAfterLogin")
    
    if urlToRedirect != "" {
        http.Redirect(w, r, urlToRedirect, http.StatusSeeOther)
        return
    }

    http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
    data := app.newTemplateData(r)
    data.Form = userSignupForm {}
    app.render(w, http.StatusOK, "signup.tmpl.html", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
    var form userSignupForm

    err := app.decodePostForm(r, &form)

    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be empty")
    form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be empty")
    form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be empty")
    form.CheckField(validator.MinChars(form.Password, 8), "password", "The password must have more than 8 characters")
    form.CheckField(validator.Matches(form.Email, validator.EmailRegex), "email", "This email is not valid")

    if !form.Valid() {
        data := app.newTemplateData(r)
        data.Form = form
        app.render(w, http.StatusUnprocessableEntity, "signup.tmpl.html", data)
        return
    }

    err = app.users.Insert(form.Name, form.Email, form.Password)

    if err != nil {
        if errors.Is(err, models.ErrDuplicateEmail) {
            form.AddFieldError("email", "Email is alredy in use")

            data := app.newTemplateData(r)
            data.Form = form
            app.render(w, http.StatusUnprocessableEntity, "signup.tmpl.html", data)
            return
        }
        app.serverError(w, err)
        return
    }

    app.sessionManager.Put(r.Context(), "flash", "Successfully signedup. Now you can log in.")
    http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogout(w http.ResponseWriter, r *http.Request) {
    err := app.sessionManager.RenewToken(r.Context())

    if err != nil {
        app.serverError(w, err)
        return
    }

    app.sessionManager.Remove(r.Context(), "authenticatedUserID")
    app.sessionManager.Put(r.Context(), "flash", "Successfully logged out")

    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) accountView(w http.ResponseWriter, r *http.Request) {
    userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")

    user, err := app.users.Get(userID)

    if err != nil {
        if errors.Is(err, models.ErrNoRecord) {
            http.Redirect(w, r, "/user/login", http.StatusSeeOther)
        } else {
            app.serverError(w, err)
        }
        return
    }

    tmplData := app.newTemplateData(r)
    tmplData.User = user

    app.render(w, http.StatusOK, "account.tmpl.html", tmplData)
}

func (app *application) updatePassword(w http.ResponseWriter, r *http.Request) {
    tmplData := app.newTemplateData(r)
    tmplData.Form = userUpdatePassword{}
    app.render(w, http.StatusOK, "password.tmpl.html", tmplData)
}

func (app *application) updatePasswordPost(w http.ResponseWriter, r *http.Request) {
    var form userUpdatePassword

    err := app.decodePostForm(r, &form)
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form.CheckField(validator.NotBlank(form.CurrentPassword), "currentPassword", "This field cannot be empty")
    form.CheckField(validator.NotBlank(form.NewPassword), "newPassword", "This field cannot be empty")
    form.CheckField(validator.NotBlank(form.ConfirmNewPassword), "confirmPassword", "This field cannot be empty")
    form.CheckField(validator.MinChars(form.NewPassword, 8), "newPassword", "The password must have more than 8 characters")
    form.CheckField(validator.MatchString(form.NewPassword, form.ConfirmNewPassword), "confirmPassword", "New password doesn't match with confirmation")

    userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
    err = app.users.UpdatePassword(userID, form.CurrentPassword, form.NewPassword)

    if err != nil {
       if !errors.Is(err, models.ErrInvalidCredentials) {
           app.serverError(w, err)
           return
       }
       form.AddFieldError("currentPassword", "Wrong password")
    }

    if !form.Valid() {
        data := app.newTemplateData(r)
        data.Form = form
        app.render(w, http.StatusUnprocessableEntity, "password.tmpl.html", data)
        return
    }

    app.sessionManager.Put(r.Context(), "flash", "Password updated sucessfully!")
    http.Redirect(w, r, "/account/view", http.StatusSeeOther)
}

