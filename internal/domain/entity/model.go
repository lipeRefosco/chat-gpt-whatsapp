package entity

type Model struct {
	Name      string
	MaxTokens int
}

// Isso é um construtor
func NewModel(name string, maxTokens int) *Model {
	return &Model{
		Name:      name,
		MaxTokens: maxTokens,
	}
}

// Isso é um método da struct Model
func (m *Model) GetMaxTokens() int {
	return m.MaxTokens
}

func (m *Model) GetModelName() string {
	return m.Name
}
