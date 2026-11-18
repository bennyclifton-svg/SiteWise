package httpapi

// Every write route has exactly one home: an operation (ops_*.go), this
// file's noOperation with the reason chat will never do it, or
// pendingOperation with the area whose operation is not written yet.
// TestWriteRoutesHaveOperations fails when a new write route has none.

// noOperation: route to the reason it is not an operation.
//
// The nine routes that start a Jev read (or the tender pass) left this map
// on 18 November 2026: chat prepares each as a card and the PM's click
// queues the read (ops_reads.go; docs/plans/2026-11-18-chat-agency.md
// Task 6). Each handler only queues; none reads while the PM waits.
var noOperation = map[string]string{
	"POST /projects/{id}/chat":                                     "the chat route itself",
	"POST /projects/{id}/chat/warm":                                "warms the model's cache; changes nothing",
	"POST /projects/{id}/chat/prepared/{prepared}/apply":           "the PM's click on a prepared chat change; it applies operations",
	"POST /projects/{id}/chat/prepared/{prepared}/dismiss":         "the PM's click dismissing a prepared chat change",
	"DELETE /projects/{id}":                                        "deleting or archiving a project is never done from chat",
	"DELETE /session":                                              "signing in, signing up and passwords are never done from chat",
	"DELETE /team/invites/{id}":                                    "team and invitations are never done from chat",
	"DELETE /team/members/{id}":                                    "team and invitations are never done from chat",
	"PATCH /projects/{id}":                                         "one conversation is one project; creating or renaming a project is done on the Projects page",
	"POST /billing/checkout":                                       "billing is never done from chat",
	"POST /billing/portal":                                         "billing is never done from chat",
	"POST /email/verify":                                           "signing in, signing up and passwords are never done from chat",
	"POST /email/verify/resend":                                    "signing in, signing up and passwords are never done from chat",
	"POST /invites/accept":                                         "signing in, signing up and passwords are never done from chat",
	"POST /password/forgot":                                        "signing in, signing up and passwords are never done from chat",
	"POST /password/reset":                                         "signing in, signing up and passwords are never done from chat",
	"POST /projects":                                               "one conversation is one project; creating or renaming a project is done on the Projects page",
	"POST /projects/{id}/changes/{change}/undo":                    "the undo route itself",
	"POST /projects/{id}/documents/delete":                         "deleting documents cannot be put back; the PM uses the screen (design 4.3)",
	"POST /projects/{id}/files":                                    "a file upload; chat does not receive files",
	"POST /projects/{id}/invoices":                                 "a file upload; chat does not receive files",
	"POST /projects/{id}/ops":                                      "the operations call itself",
	"POST /projects/{id}/procurement/packages/{pkg}/award/preview": "a read that happens to be a POST",
	"POST /projects/{id}/procurement/packages/{pkg}/submissions":   "a file upload; chat does not receive files",
	"POST /projects/{id}/program/import":                           "a file upload; chat does not receive files",
	"POST /projects/{id}/program/redo":                             "Program's own undo and redo; the change card calls them",
	"POST /projects/{id}/program/undo":                             "Program's own undo and redo; the change card calls them",
	"POST /projects/{id}/specifications/imports":                   "a file upload; chat does not receive files",
	"POST /reports/{r}/issue":                                      "issuing leaves the building; the PM clicks Issue on the screen (design 4.3)",
	"POST /session":                                                "signing in, signing up and passwords are never done from chat",
	"POST /signin":                                                 "signing in, signing up and passwords are never done from chat",
	"POST /signup":                                                 "signing in, signing up and passwords are never done from chat",
	"POST /stripe/webhook":                                         "billing is never done from chat",
	"POST /team/invites":                                           "team and invitations are never done from chat",
	"PUT /account/model":                                           "the org owner switches the model on or off on the Account page, not from chat",
	"PUT /projects/{id}/archived":                                  "deleting or archiving a project is never done from chat",
	"POST /projects/{id}/shares":                                   "sharing outside the org is the PM's press of Create link; chat opens Share filled in",
	"POST /guest/session":                                          "a guest opening a link; chat never signs anyone in",
	"POST /guest/quote":                                            "a guest's file upload; the guest page has no chat",
}

// pendingOperation: route to its area. Chat says "I can't do that from
// chat yet" for these.
var pendingOperation = map[string]string{
	"DELETE /reports/{r}/settings/{key}":                                            "reports",
	"PATCH /projects/{id}/packages/{pkg}/scope/{scope}":                             "procurement",
	"PATCH /projects/{id}/packages/{pkg}/stages/{stage}":                            "procurement",
	"PATCH /projects/{id}/submissions/{sub}":                                        "procurement",
	"PATCH /sites/{site}":                                                           "profile",
	"POST /projects/{id}/packages":                                                  "procurement",
	"POST /projects/{id}/packages/combine":                                          "procurement",
	"POST /projects/{id}/packages/proposed/accept":                                  "procurement",
	"POST /projects/{id}/packages/{pkg}/scope":                                      "procurement",
	"POST /projects/{id}/packages/{pkg}/stage-sync":                                 "procurement",
	"POST /projects/{id}/packages/{pkg}/stages":                                     "procurement",
	"POST /projects/{id}/packages/{pkg}/uncombine":                                  "procurement",
	"POST /projects/{id}/packages/{pkg}/ungroup":                                    "procurement",
	"POST /projects/{id}/procurement/adopt":                                         "procurement",
	"POST /projects/{id}/procurement/dismiss":                                       "procurement",
	"POST /projects/{id}/procurement/packages/{pkg}/firms":                          "procurement",
	"POST /projects/{id}/program/prepare":                                           "program",
	"POST /projects/{id}/program/question":                                          "program",
	"POST /projects/{id}/program/update":                                            "program",
	"POST /projects/{id}/submissions/{sub}/alignment":                               "procurement",
	"POST /projects/{id}/works/{wi}/split":                                          "profile",
	"POST /reports/{r}/settings":                                                    "reports",
	"POST /reports/{r}/stage-sync":                                                  "reports",
	"PUT /projects/{id}/procurement/packages/{pkg}/comparison-row":                  "procurement",
	"PUT /projects/{id}/procurement/packages/{pkg}/submissions/{sub}/ratings/{key}": "procurement",
	"PUT /projects/{id}/submissions/{sub}/appointment-row":                          "procurement",
	"PUT /projects/{id}/submissions/{sub}/cells/{key}":                              "procurement",
	"PUT /reports/{r}/facts/{fact}":                                                 "reports",
	"PUT /reports/{r}/settings/{key}":                                               "reports",
}
