package amazon

// kdpSpecs is the root type for kdp-specs.yaml.
type kdpSpecs struct {
	Name             string    `yaml:"name"`
	DisplayName      string    `yaml:"display_name"`
	CoverBoardInches float64   `yaml:"cover_board_inches"`
	Sizes            []sizeSpec `yaml:"sizes"`
}

type sizeSpec struct {
	ID           string               `yaml:"id"`
	WidthInches  float64              `yaml:"width_inches"`
	HeightInches float64              `yaml:"height_inches"`
	BleedInches  float64              `yaml:"bleed_inches"`
	Bindings     map[string]bindingSpec `yaml:"bindings"`
}

type bindingSpec struct {
	Supported  bool                   `yaml:"supported"`
	MinPages   int                    `yaml:"min_pages"`
	MaxPages   int                    `yaml:"max_pages"`
	PaperTypes map[string]paperSpec   `yaml:"paper_types"`
}

type paperSpec struct {
	Supported                bool    `yaml:"supported"`
	ThicknessPerPageInches   float64 `yaml:"thickness_per_page_inches"`
}