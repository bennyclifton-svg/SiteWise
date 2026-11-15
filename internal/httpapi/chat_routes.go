package httpapi

import "net/http"

// chatRoutes are the operations call and the chat routes. They are kept
// apart from the screen routes so the table in projects.go stays as other
// work left it.
func chatRoutes() map[string]func(http.ResponseWriter, *http.Request, Deps) {
	return map[string]func(http.ResponseWriter, *http.Request, Deps){
		"POST /projects/{id}/ops":                              postOps,
		"POST /projects/{id}/chat":                             postChat,
		"GET /projects/{id}/chat":                              getChat,
		"GET /projects/{id}/chat/read":                         getChatRead,
		"POST /projects/{id}/chat/warm":                        postChatWarm,
		"POST /projects/{id}/chat/prepared/{prepared}/apply":   postChatPrepared,
		"POST /projects/{id}/chat/prepared/{prepared}/dismiss": postChatPrepared,
	}
}
