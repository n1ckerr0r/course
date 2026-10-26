package domain

type Reputation string

const (
	ReputationNew     Reputation = "new"
	ReputationTrusted Reputation = "trusted"
)

type Author struct {
	ID         int64
	Reputation Reputation
}

func NewAuthor(id int64) Author {
	return Author{ID: id, Reputation: ReputationNew}
}

func (a Author) CanSkipManualModeration() bool {
	return a.Reputation == ReputationTrusted
}
