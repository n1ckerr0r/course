package domain

type Reputation string

const (
	ReputationNew     Reputation = "new"
	ReputationTrusted Reputation = "trusted"
)

type Author struct {
	id         int64
	reputation Reputation
}

func NewAuthor(id int64) Author {
	return Author{id: id, reputation: ReputationNew}
}

func RestoreAuthor(id int64, reputation Reputation) Author {
	return Author{id: id, reputation: reputation}
}

func (a Author) ID() int64 { return a.id }

func (a Author) Reputation() Reputation { return a.reputation }

func (a Author) CanSkipManualModeration() bool {
	return a.reputation == ReputationTrusted
}
