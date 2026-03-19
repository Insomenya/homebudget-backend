package repository

type Repos struct {
	Member      *MemberRepo
	Account     *AccountRepo
	Category    *CategoryRepo
	SharedGroup *SharedGroupRepo
	Transaction *TransactionRepo
	Planned     *PlannedRepo
	Analytics   *AnalyticsRepo
	Lookup      *LookupRepo
	Loan        *LoanRepo
}