package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"homebudget/internal/repository"
)

func NewRouter(repos *repository.Repos, corsOrigin string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(corsMiddleware(corsOrigin))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	mh := &MemberHandler{repo: repos.Member}
	ah := &AccountHandler{repo: repos.Account}
	ch := &CategoryHandler{repo: repos.Category}
	gh := &SharedGroupHandler{repo: repos.SharedGroup}
	th := &TransactionHandler{repo: repos.Transaction}
	ph := &PlannedHandler{repo: repos.Planned, txRepo: repos.Transaction}
	an := &AnalyticsHandler{repo: repos.Analytics}
	mt := &MetaHandler{repo: repos.Lookup}
	lh := &LoanHandler{repo: repos.Loan}
	dh := &DashboardHandler{
		accounts: repos.Account, transactions: repos.Transaction,
		groups: repos.SharedGroup, planned: repos.Planned,
	}

	r.Route("/api", func(api chi.Router) {
		api.Get("/dashboard", dh.Get)
		api.Get("/meta", mt.GetMeta)

		api.Route("/lookups", func(lr chi.Router) {
			lr.Get("/", mt.ListGroup)
			lr.Post("/", mt.Create)
			lr.Put("/{id}", mt.Update)
			lr.Delete("/{id}", mt.Delete)
		})

		api.Route("/members", func(mr chi.Router) {
			mr.Get("/", mh.List)
			mr.Post("/", mh.Create)
			mr.Get("/{id}", mh.GetByID)
			mr.Put("/{id}", mh.Update)
			mr.Delete("/{id}", mh.Delete)
		})

		api.Route("/accounts", func(ar chi.Router) {
			ar.Get("/", ah.List)
			ar.Post("/", ah.Create)
			ar.Get("/{id}", ah.GetByID)
			ar.Put("/{id}", ah.Update)
			ar.Delete("/{id}", ah.Delete)
		})

		api.Route("/categories", func(cr chi.Router) {
			cr.Get("/", ch.List)
			cr.Post("/", ch.Create)
			cr.Get("/{id}", ch.GetByID)
			cr.Put("/{id}", ch.Update)
			cr.Delete("/{id}", ch.Delete)
		})

		api.Route("/groups", func(gr chi.Router) {
			gr.Get("/", gh.List)
			gr.Post("/", gh.Create)
			gr.Get("/{id}", gh.GetByID)
			gr.Put("/{id}", gh.Update)
			gr.Delete("/{id}", gh.Delete)
			gr.Get("/{id}/settlement", gh.Settlement)
			gr.Get("/{id}/turnover", gh.Turnover)
		})

		api.Route("/transactions", func(tr chi.Router) {
			tr.Get("/", th.List)
			tr.Post("/", th.Create)
			tr.Get("/{id}", th.GetByID)
			tr.Put("/{id}", th.Update)
			tr.Delete("/{id}", th.Delete)
		})

		api.Route("/planned", func(pr chi.Router) {
			pr.Get("/", ph.List)
			pr.Post("/", ph.Create)
			pr.Get("/upcoming", ph.Upcoming)
			pr.Get("/{id}", ph.GetByID)
			pr.Put("/{id}", ph.Update)
			pr.Delete("/{id}", ph.Delete)
			pr.Post("/{id}/execute", ph.Execute)
		})

		api.Route("/loans", func(lr chi.Router) {
			lr.Get("/", lh.List)
			lr.Post("/", lh.Create)
			lr.Get("/{id}", lh.GetByID)
			lr.Put("/{id}", lh.Update)
			lr.Delete("/{id}", lh.Delete)
			lr.Get("/{id}/schedule", lh.DailySchedule)
		})

		api.Route("/analytics", func(ar chi.Router) {
			ar.Get("/categories", an.Categories)
			ar.Get("/trends", an.Trends)
		})
	})

	return r
}