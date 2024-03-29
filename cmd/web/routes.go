package main

import (
	"net/http"

	"github.com/NPeykov/snippetbox/ui"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
    fileServer := http.FileServer(http.FS(ui.Files))
    router := httprouter.New()
    router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {app.notFound(w)})
    router.Handler(http.MethodGet, "/static/*filepath", fileServer)

    dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate)
    protected := dynamic.Append(app.requireAuthentication)

	router.HandlerFunc(http.MethodGet, "/ping", ping)
	router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.home))
	router.Handler(http.MethodGet, "/about", dynamic.ThenFunc(app.about))
    router.Handler(http.MethodGet, "/snippet/view/:id", dynamic.ThenFunc(app.snippetView))
    router.Handler(http.MethodGet, "/snippet/create", protected.ThenFunc(app.snippetCreate))
    router.Handler(http.MethodPost, "/snippet/create", protected.ThenFunc(app.snippetCreatePost))
    router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
    router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))
    router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
    router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))
    router.Handler(http.MethodGet, "/account/view", protected.ThenFunc(app.accountView))
    router.Handler(http.MethodPost, "/user/logout", protected.ThenFunc(app.userLogout))
    router.Handler(http.MethodGet, "/account/password/update", protected.ThenFunc(app.updatePassword))
    router.Handler(http.MethodPost, "/account/password/update", protected.ThenFunc(app.updatePasswordPost))
    return alice.New(app.recoverPanic, app.logRequest, secureHeaders).Then(router)
}
